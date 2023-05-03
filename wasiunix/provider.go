package wasiunix

import (
	"context"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/stealthrocket/wasi"
	"github.com/stealthrocket/wasi/internal/descriptor"
	"golang.org/x/sys/unix"
)

// Provider is a WASI preview 1 implementation for Unix systems.
//
// It implements the wasi.Provider interface.
//
// The provider is not safe for concurrent use.
type Provider struct {
	// Args are the environment variables accessible via ArgsGet.
	Args []string

	// Environ is the environment variables accessible via EnvironGet.
	Environ []string

	// Realtime returns the realtime clock value.
	Realtime          func(context.Context) (uint64, error)
	RealtimePrecision time.Duration

	// Monotonic returns the monotonic clock value.
	Monotonic          func(context.Context) (uint64, error)
	MonotonicPrecision time.Duration

	// Yield is called when SchedYield is called. If Yield is nil,
	// SchedYield is a noop.
	Yield func(context.Context) error

	// Exit is called with an exit code when ProcExit is called.
	// If Exit is nil, ProcExit is a noop.
	Exit func(context.Context, int) error

	// Raise is called with a signal when ProcRaise is called.
	// If Raise is nil, ProcRaise is a noop.
	Raise func(context.Context, int) error

	// Rand is the source for RandomGet.
	Rand io.Reader

	fds      descriptor.Table[wasi.FD, *fdinfo]
	preopens descriptor.Table[wasi.FD, struct{}]
	pollfds  []unix.PollFd
}

type fdinfo struct {
	// path is the path of the file.
	path string

	// fd is the underlying OS file descriptor.
	fd int

	// stat is cached information about the file descriptor.
	stat wasi.FDStat

	// dirEntries are cached directory entries.
	dirEntries []os.DirEntry
}

// Preopen adds an open file to the list of pre-opens.
func (p *Provider) Preopen(hostfd int, path string, fdstat wasi.FDStat) {
	fdstat.RightsBase &= wasi.AllRights
	fdstat.RightsInheriting &= wasi.AllRights
	p.preopens.Assign(
		p.fds.Insert(&fdinfo{
			fd:   hostfd,
			path: path,
			stat: fdstat,
		}),
		struct{}{},
	)
}

func (p *Provider) isPreopen(fd wasi.FD) bool {
	_, ok := p.preopens.Lookup(fd)
	return ok
}

func (p *Provider) lookupFD(guestfd wasi.FD, rights wasi.Rights) (*fdinfo, wasi.Errno) {
	f, ok := p.fds.Lookup(guestfd)
	if !ok {
		return nil, wasi.EBADF
	}
	if !f.stat.RightsBase.Has(rights) {
		return nil, wasi.ENOTCAPABLE
	}
	return f, wasi.ESUCCESS
}

func (p *Provider) lookupPreopenFD(guestfd wasi.FD, rights wasi.Rights) (*fdinfo, wasi.Errno) {
	if !p.isPreopen(guestfd) {
		return nil, wasi.EBADF
	}
	f, errno := p.lookupFD(guestfd, rights)
	if errno != wasi.ESUCCESS {
		return nil, errno
	}
	if f.stat.FileType != wasi.DirectoryType {
		return nil, wasi.ENOTDIR
	}
	return f, wasi.ESUCCESS
}

func (p *Provider) lookupSocketFD(guestfd wasi.FD, rights wasi.Rights) (*fdinfo, wasi.Errno) {
	f, errno := p.lookupFD(guestfd, rights)
	if errno != wasi.ESUCCESS {
		return nil, errno
	}
	switch f.stat.FileType {
	case wasi.SocketStreamType, wasi.SocketDGramType:
		return f, wasi.ESUCCESS
	default:
		return nil, wasi.ENOTSOCK
	}
}

func (p *Provider) ArgsGet(ctx context.Context) ([]string, wasi.Errno) {
	return p.Args, wasi.ESUCCESS
}

func (p *Provider) EnvironGet(ctx context.Context) ([]string, wasi.Errno) {
	return p.Environ, wasi.ESUCCESS
}

func (p *Provider) ClockResGet(ctx context.Context, id wasi.ClockID) (wasi.Timestamp, wasi.Errno) {
	switch id {
	case wasi.Realtime:
		return wasi.Timestamp(p.RealtimePrecision), wasi.ESUCCESS
	case wasi.Monotonic:
		return wasi.Timestamp(p.MonotonicPrecision), wasi.ESUCCESS
	case wasi.ProcessCPUTimeID, wasi.ThreadCPUTimeID:
		return 0, wasi.ENOTSUP
	default:
		return 0, wasi.EINVAL
	}
}

func (p *Provider) ClockTimeGet(ctx context.Context, id wasi.ClockID, precision wasi.Timestamp) (wasi.Timestamp, wasi.Errno) {
	switch id {
	case wasi.Realtime:
		if p.Realtime == nil {
			return 0, wasi.ENOTSUP
		}
		t, err := p.Realtime(ctx)
		return wasi.Timestamp(t), makeErrno(err)
	case wasi.Monotonic:
		if p.Monotonic == nil {
			return 0, wasi.ENOTSUP
		}
		t, err := p.Monotonic(ctx)
		return wasi.Timestamp(t), makeErrno(err)
	case wasi.ProcessCPUTimeID, wasi.ThreadCPUTimeID:
		return 0, wasi.ENOTSUP
	default:
		return 0, wasi.EINVAL
	}
}

func (p *Provider) FDAdvise(ctx context.Context, fd wasi.FD, offset wasi.FileSize, length wasi.FileSize, advice wasi.Advice) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDAdviseRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := fdadvise(f.fd, int64(offset), int64(length), advice)
	return makeErrno(err)
}

func (p *Provider) FDAllocate(ctx context.Context, fd wasi.FD, offset wasi.FileSize, length wasi.FileSize) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDAllocateRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := fallocate(f.fd, int64(offset), int64(length))
	return makeErrno(err)
}

func (p *Provider) FDClose(ctx context.Context, fd wasi.FD) wasi.Errno {
	f, errno := p.lookupFD(fd, 0)
	if errno != wasi.ESUCCESS {
		return errno
	}
	p.fds.Delete(fd)
	// Note: closing pre-opens is allowed.
	// See github.com/WebAssembly/wasi-testsuite/blob/1b1d4a5/tests/rust/src/bin/close_preopen.rs
	p.preopens.Delete(fd)
	err := unix.Close(f.fd)
	return makeErrno(err)
}

func (p *Provider) FDDataSync(ctx context.Context, fd wasi.FD) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDDataSyncRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := fdatasync(f.fd)
	return makeErrno(err)
}

func (p *Provider) FDStatGet(ctx context.Context, fd wasi.FD) (wasi.FDStat, wasi.Errno) {
	f, errno := p.lookupFD(fd, 0)
	if errno != wasi.ESUCCESS {
		return wasi.FDStat{}, errno
	}
	return f.stat, wasi.ESUCCESS
}

func (p *Provider) FDStatSetFlags(ctx context.Context, fd wasi.FD, flags wasi.FDFlags) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDStatSetFlagsRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	changes := flags ^ f.stat.Flags
	if changes == 0 {
		return wasi.ESUCCESS
	}
	if changes.Has(wasi.Sync | wasi.DSync | wasi.RSync) {
		return wasi.ENOSYS // TODO: support changing {Sync,DSync,Rsync}
	}
	fl, err := unix.FcntlInt(uintptr(f.fd), unix.F_GETFL, 0)
	if err != nil {
		return makeErrno(err)
	}
	if flags.Has(wasi.Append) {
		fl |= unix.O_APPEND
	} else {
		fl &^= unix.O_APPEND
	}
	if flags.Has(wasi.NonBlock) {
		fl |= unix.O_NONBLOCK
	} else {
		fl &^= unix.O_NONBLOCK
	}
	if _, err := unix.FcntlInt(uintptr(f.fd), unix.F_SETFL, fl); err != nil {
		return makeErrno(err)
	}
	f.stat.Flags ^= changes
	return wasi.ESUCCESS
}

func (p *Provider) FDStatSetRights(ctx context.Context, fd wasi.FD, rightsBase, rightsInheriting wasi.Rights) wasi.Errno {
	f, errno := p.lookupFD(fd, 0)
	if errno != wasi.ESUCCESS {
		return errno
	}
	// Rights can only be preserved or removed, not added.
	rightsBase &= wasi.AllRights
	rightsInheriting &= wasi.AllRights
	if (rightsBase &^ f.stat.RightsBase) != 0 {
		return wasi.ENOTCAPABLE
	}
	if (rightsInheriting &^ f.stat.RightsInheriting) != 0 {
		return wasi.ENOTCAPABLE
	}
	f.stat.RightsBase &= rightsBase
	f.stat.RightsInheriting &= rightsInheriting
	return wasi.ESUCCESS
}

func (p *Provider) FDFileStatGet(ctx context.Context, fd wasi.FD) (wasi.FileStat, wasi.Errno) {
	f, errno := p.lookupFD(fd, wasi.FDFileStatGetRight)
	if errno != wasi.ESUCCESS {
		return wasi.FileStat{}, errno
	}
	var sysStat unix.Stat_t
	if err := unix.Fstat(f.fd, &sysStat); err != nil {
		return wasi.FileStat{}, makeErrno(err)
	}
	stat := makeFileStat(&sysStat)
	switch f.fd {
	case syscall.Stdin, syscall.Stdout, syscall.Stderr:
		// Override stdio size/times.
		// See github.com/WebAssembly/wasi-testsuite/blob/1b1d4a5/tests/rust/src/bin/fd_filestat_get.rs
		stat.Size = 0
		stat.AccessTime = 0
		stat.ModifyTime = 0
		stat.ChangeTime = 0
	}
	return stat, wasi.ESUCCESS
}

func (p *Provider) FDFileStatSetSize(ctx context.Context, fd wasi.FD, size wasi.FileSize) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDFileStatSetSizeRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := unix.Ftruncate(f.fd, int64(size))
	return makeErrno(err)
}

func (p *Provider) FDFileStatSetTimes(ctx context.Context, fd wasi.FD, accessTime, modifyTime wasi.Timestamp, flags wasi.FSTFlags) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDFileStatSetTimesRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	var sysStat unix.Stat_t
	if err := unix.Fstat(f.fd, &sysStat); err != nil {
		return makeErrno(err)
	}
	ts := [2]unix.Timespec{sysStat.Atim, sysStat.Mtim}
	if flags.Has(wasi.AccessTimeNow) || flags.Has(wasi.ModifyTimeNow) {
		if p.Monotonic == nil {
			return wasi.ENOSYS
		}
		now, err := p.Monotonic(ctx)
		if err != nil {
			return makeErrno(err)
		}
		if flags.Has(wasi.AccessTimeNow) {
			accessTime = wasi.Timestamp(now)
		}
		if flags.Has(wasi.ModifyTimeNow) {
			modifyTime = wasi.Timestamp(now)
		}
	}
	if flags.Has(wasi.AccessTime) || flags.Has(wasi.AccessTimeNow) {
		ts[0] = unix.NsecToTimespec(int64(accessTime))
	}
	if flags.Has(wasi.ModifyTime) || flags.Has(wasi.ModifyTimeNow) {
		ts[1] = unix.NsecToTimespec(int64(modifyTime))
	}
	err := futimens(f.fd, &ts)
	return makeErrno(err)
}

func (p *Provider) FDPread(ctx context.Context, fd wasi.FD, iovecs []wasi.IOVec, offset wasi.FileSize) (wasi.Size, wasi.Errno) {
	f, errno := p.lookupFD(fd, wasi.FDReadRight|wasi.FDSeekRight)
	if errno != wasi.ESUCCESS {
		return 0, errno
	}
	n, err := preadv(f.fd, makeIOVecs(iovecs), int64(offset))
	return wasi.Size(n), makeErrno(err)
}

func (p *Provider) FDPreStatGet(ctx context.Context, fd wasi.FD) (wasi.PreStat, wasi.Errno) {
	f, errno := p.lookupPreopenFD(fd, 0)
	if errno != wasi.ESUCCESS {
		return wasi.PreStat{}, errno
	}
	stat := wasi.PreStat{
		Type: wasi.PreOpenDir,
		PreStatDir: wasi.PreStatDir{
			NameLength: wasi.Size(len(f.path)),
		},
	}
	return stat, wasi.ESUCCESS
}

func (p *Provider) FDPreStatDirName(ctx context.Context, fd wasi.FD) (string, wasi.Errno) {
	f, errno := p.lookupPreopenFD(fd, 0)
	if errno != wasi.ESUCCESS {
		return "", errno
	}
	return f.path, wasi.ESUCCESS
}

func (p *Provider) FDPwrite(ctx context.Context, fd wasi.FD, iovecs []wasi.IOVec, offset wasi.FileSize) (wasi.Size, wasi.Errno) {
	f, errno := p.lookupFD(fd, wasi.FDWriteRight|wasi.FDSeekRight)
	if errno != wasi.ESUCCESS {
		return 0, errno
	}
	n, err := pwritev(f.fd, makeIOVecs(iovecs), int64(offset))
	return wasi.Size(n), makeErrno(err)
}

func (p *Provider) FDRead(ctx context.Context, fd wasi.FD, iovecs []wasi.IOVec) (wasi.Size, wasi.Errno) {
	f, errno := p.lookupFD(fd, wasi.FDReadRight)
	if errno != wasi.ESUCCESS {
		return 0, errno
	}
	n, err := readv(f.fd, makeIOVecs(iovecs))
	return wasi.Size(n), makeErrno(err)
}

func (p *Provider) FDReadDir(ctx context.Context, fd wasi.FD, buffer []wasi.DirEntryName, bufferSizeBytes int, cookie wasi.DirCookie) ([]wasi.DirEntryName, wasi.Errno) {
	f, errno := p.lookupFD(fd, wasi.FDReadDirRight)
	if errno != wasi.ESUCCESS {
		return nil, errno
	}

	// TODO: use a readdir iterator
	// This is all very tricky to get right, so let's cheat for now
	// and use os.ReadDir.
	if cookie == 0 {
		entries, err := os.ReadDir(f.path)
		if err != nil {
			return buffer, makeErrno(err)
		}
		f.dirEntries = entries
		// Add . and .. entries, since they're stripped by os.ReadDir
		if info, err := os.Stat(f.path); err == nil {
			f.dirEntries = append(f.dirEntries, &statDirEntry{".", info})
		}
		if info, err := os.Stat(filepath.Join(f.path, "..")); err == nil {
			f.dirEntries = append(f.dirEntries, &statDirEntry{"..", info})
		}
	}
	if cookie > math.MaxInt {
		return buffer, wasi.EINVAL
	}
	var n int
	pos := int(cookie)
	for ; pos < len(f.dirEntries) && n < bufferSizeBytes; pos++ {
		e := f.dirEntries[pos]
		name := e.Name()
		info, err := e.Info()
		if err != nil {
			return buffer, makeErrno(err)
		}
		s := info.Sys().(*syscall.Stat_t)
		buffer = append(buffer, wasi.DirEntryName{
			Entry: wasi.DirEntry{
				Type:       makeFileType(uint32(s.Mode)),
				INode:      wasi.INode(s.Ino),
				NameLength: wasi.DirNameLength(len(name)),
				Next:       wasi.DirCookie(pos + 1),
			},
			Name: name,
		})
		n += int(unsafe.Sizeof(wasi.DirEntry{})) + len(name)
	}
	return buffer, wasi.ESUCCESS
}

func (p *Provider) FDRenumber(ctx context.Context, from, to wasi.FD) wasi.Errno {
	if p.isPreopen(from) || p.isPreopen(to) {
		return wasi.ENOTSUP
	}
	f, errno := p.lookupFD(from, 0)
	if errno != wasi.ESUCCESS {
		return errno
	}
	// TODO: limit max file descriptor number
	f, replaced := p.fds.Assign(to, f)
	if replaced {
		unix.Close(f.fd)
	}
	return wasi.ENOSYS
}

func (p *Provider) FDSync(ctx context.Context, fd wasi.FD) wasi.Errno {
	f, errno := p.lookupFD(fd, wasi.FDSyncRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := fsync(f.fd)
	return makeErrno(err)
}

func (p *Provider) FDSeek(ctx context.Context, fd wasi.FD, delta wasi.FileDelta, whence wasi.Whence) (wasi.FileSize, wasi.Errno) {
	return p.fdseek(fd, wasi.FDSeekRight, delta, whence)
}

func (p *Provider) FDTell(ctx context.Context, fd wasi.FD) (wasi.FileSize, wasi.Errno) {
	return p.fdseek(fd, wasi.FDTellRight, 0, wasi.SeekCurrent)
}

func (p *Provider) fdseek(fd wasi.FD, rights wasi.Rights, delta wasi.FileDelta, whence wasi.Whence) (wasi.FileSize, wasi.Errno) {
	// Note: FDSeekRight implies FDTellRight. FDTellRight also includes the
	// right to invoke FDSeek in such a way that the file offset remains
	// unaltered.
	f, errno := p.lookupFD(fd, rights)
	if errno != wasi.ESUCCESS {
		return 0, errno
	}
	var sysWhence int
	switch whence {
	case wasi.SeekStart:
		sysWhence = unix.SEEK_SET
	case wasi.SeekCurrent:
		sysWhence = unix.SEEK_CUR
	case wasi.SeekEnd:
		sysWhence = unix.SEEK_END
	default:
		return 0, wasi.EINVAL
	}
	off, err := lseek(f.fd, int64(delta), sysWhence)
	return wasi.FileSize(off), makeErrno(err)
}

func (p *Provider) FDWrite(ctx context.Context, fd wasi.FD, iovecs []wasi.IOVec) (wasi.Size, wasi.Errno) {
	f, errno := p.lookupFD(fd, wasi.FDWriteRight)
	if errno != wasi.ESUCCESS {
		return 0, errno
	}
	n, err := writev(f.fd, makeIOVecs(iovecs))
	return wasi.Size(n), makeErrno(err)
}

func (p *Provider) PathCreateDirectory(ctx context.Context, fd wasi.FD, path string) wasi.Errno {
	d, errno := p.lookupFD(fd, wasi.PathCreateDirectoryRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := unix.Mkdirat(d.fd, path, 0755)
	return makeErrno(err)
}

func (p *Provider) PathFileStatGet(ctx context.Context, fd wasi.FD, flags wasi.LookupFlags, path string) (wasi.FileStat, wasi.Errno) {
	d, errno := p.lookupFD(fd, wasi.PathFileStatGetRight)
	if errno != wasi.ESUCCESS {
		return wasi.FileStat{}, errno
	}
	var sysStat unix.Stat_t
	var sysFlags int
	if !flags.Has(wasi.SymlinkFollow) {
		sysFlags |= unix.AT_SYMLINK_NOFOLLOW
	}
	err := unix.Fstatat(d.fd, path, &sysStat, sysFlags)
	return makeFileStat(&sysStat), makeErrno(err)
}

func (p *Provider) PathFileStatSetTimes(ctx context.Context, fd wasi.FD, lookupFlags wasi.LookupFlags, path string, accessTime, modifyTime wasi.Timestamp, fstFlags wasi.FSTFlags) wasi.Errno {
	d, errno := p.lookupFD(fd, wasi.PathFileStatSetTimesRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	if fstFlags.Has(wasi.AccessTimeNow) || fstFlags.Has(wasi.ModifyTimeNow) {
		now := wasi.Timestamp(time.Now().UnixNano())
		if fstFlags.Has(wasi.AccessTimeNow) {
			accessTime = now
		}
		if fstFlags.Has(wasi.ModifyTimeNow) {
			modifyTime = now
		}
	}
	var sysFlags int
	if !lookupFlags.Has(wasi.SymlinkFollow) {
		sysFlags |= unix.AT_SYMLINK_NOFOLLOW
	}
	var ts [2]unix.Timespec
	changeAccessTime := fstFlags.Has(wasi.AccessTime) || fstFlags.Has(wasi.AccessTimeNow)
	changeModifyTime := fstFlags.Has(wasi.ModifyTime) || fstFlags.Has(wasi.ModifyTimeNow)
	if !changeAccessTime || !changeModifyTime {
		var stat unix.Stat_t
		err := unix.Fstatat(d.fd, path, &stat, sysFlags)
		if err != nil {
			return makeErrno(err)
		}
		ts[0] = stat.Atim
		ts[1] = stat.Mtim
	}
	if changeAccessTime {
		ts[0] = unix.NsecToTimespec(int64(accessTime))
	}
	if changeModifyTime {
		ts[1] = unix.NsecToTimespec(int64(modifyTime))
	}
	err := unix.UtimesNanoAt(d.fd, path, ts[:], sysFlags)
	return makeErrno(err)
}

func (p *Provider) PathLink(ctx context.Context, fd wasi.FD, flags wasi.LookupFlags, oldPath string, newFD wasi.FD, newPath string) wasi.Errno {
	oldDir, errno := p.lookupFD(fd, wasi.PathLinkSourceRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	newDir, errno := p.lookupFD(newFD, wasi.PathLinkTargetRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	sysFlags := 0
	if flags.Has(wasi.SymlinkFollow) {
		sysFlags |= unix.AT_SYMLINK_FOLLOW
	}
	err := unix.Linkat(oldDir.fd, oldPath, newDir.fd, newPath, sysFlags)
	return makeErrno(err)
}

func (p *Provider) PathOpen(ctx context.Context, fd wasi.FD, lookupFlags wasi.LookupFlags, path string, openFlags wasi.OpenFlags, rightsBase, rightsInheriting wasi.Rights, fdFlags wasi.FDFlags) (wasi.FD, wasi.Errno) {
	d, errno := p.lookupFD(fd, wasi.PathOpenRight)
	if errno != wasi.ESUCCESS {
		return -1, errno
	}
	clean := filepath.Clean(path)
	if strings.HasPrefix(clean, "/") || strings.HasPrefix(clean, "../") {
		return -1, wasi.EPERM
	}

	// Rights can only be preserved or removed, not added.
	rightsBase &= wasi.AllRights
	rightsInheriting &= wasi.AllRights
	if (rightsBase &^ d.stat.RightsInheriting) != 0 {
		return -1, wasi.ENOTCAPABLE
	} else if (rightsInheriting &^ d.stat.RightsInheriting) != 0 {
		return -1, wasi.ENOTCAPABLE
	}
	rightsBase &= d.stat.RightsInheriting
	rightsInheriting &= d.stat.RightsInheriting

	oflags := unix.O_CLOEXEC
	if openFlags.Has(wasi.OpenDirectory) {
		oflags |= unix.O_DIRECTORY
		// Directories cannot have FDSeekRight (and possibly other rights).
		// See github.com/WebAssembly/wasi-testsuite/blob/1b1d4a5/tests/rust/src/bin/directory_seek.rs
		rightsBase &^= wasi.FDSeekRight
	}
	if openFlags.Has(wasi.OpenCreate) {
		if !d.stat.RightsBase.Has(wasi.PathCreateFileRight) {
			return -1, wasi.ENOTCAPABLE
		}
		oflags |= unix.O_CREAT
	}
	if openFlags.Has(wasi.OpenExclusive) {
		oflags |= unix.O_EXCL
	}
	if openFlags.Has(wasi.OpenTruncate) {
		if !d.stat.RightsBase.Has(wasi.PathFileStatSetSizeRight) {
			return -1, wasi.ENOTCAPABLE
		}
		oflags |= unix.O_TRUNC
	}
	if fdFlags.Has(wasi.Append) {
		oflags |= unix.O_APPEND
	}
	if fdFlags.Has(wasi.DSync) {
		oflags |= unix.O_DSYNC
	}
	if fdFlags.Has(wasi.Sync) {
		oflags |= unix.O_SYNC
	}
	// TODO: handle O_RSYNC
	if fdFlags.Has(wasi.NonBlock) {
		oflags |= unix.O_NONBLOCK
	}
	if !lookupFlags.Has(wasi.SymlinkFollow) {
		oflags |= unix.O_NOFOLLOW
	}
	switch {
	case openFlags.Has(wasi.OpenDirectory):
		oflags |= unix.O_RDONLY
	case rightsBase.HasAny(wasi.ReadRights) && rightsBase.HasAny(wasi.WriteRights):
		oflags |= unix.O_RDWR
	case rightsBase.HasAny(wasi.ReadRights):
		oflags |= unix.O_RDONLY
	case rightsBase.HasAny(wasi.WriteRights):
		oflags |= unix.O_WRONLY
	default:
		oflags |= unix.O_RDONLY
	}

	mode := uint32(0644)
	fileType := wasi.RegularFileType
	if (oflags & unix.O_DIRECTORY) != 0 {
		fileType = wasi.DirectoryType
		mode = 0
	}
	hostfd, err := unix.Openat(d.fd, path, oflags, mode)
	if err != nil {
		return -1, makeErrno(err)
	}

	guestfd := p.fds.Insert(&fdinfo{
		fd:   hostfd,
		path: filepath.Join(d.path, path),
		stat: wasi.FDStat{
			FileType:         fileType,
			Flags:            fdFlags,
			RightsBase:       rightsBase,
			RightsInheriting: rightsInheriting,
		},
	})
	return guestfd, wasi.ESUCCESS
}

func (p *Provider) PathReadLink(ctx context.Context, fd wasi.FD, path string, buffer []byte) ([]byte, wasi.Errno) {
	d, errno := p.lookupFD(fd, wasi.PathReadLinkRight)
	if errno != wasi.ESUCCESS {
		return buffer, errno
	}
	n, err := unix.Readlinkat(d.fd, path, buffer)
	if err != nil {
		return buffer, makeErrno(err)
	} else if n == len(buffer) {
		return buffer, wasi.ERANGE
	}
	return buffer[:n], wasi.ESUCCESS
}

func (p *Provider) PathRemoveDirectory(ctx context.Context, fd wasi.FD, path string) wasi.Errno {
	d, errno := p.lookupFD(fd, wasi.PathRemoveDirectoryRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := unix.Unlinkat(d.fd, path, unix.AT_REMOVEDIR)
	return makeErrno(err)
}

func (p *Provider) PathRename(ctx context.Context, fd wasi.FD, oldPath string, newFD wasi.FD, newPath string) wasi.Errno {
	oldDir, errno := p.lookupFD(fd, wasi.PathRenameSourceRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	newDir, errno := p.lookupFD(newFD, wasi.PathRenameTargetRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := unix.Renameat(oldDir.fd, oldPath, newDir.fd, newPath)
	return makeErrno(err)
}

func (p *Provider) PathSymlink(ctx context.Context, oldPath string, fd wasi.FD, newPath string) wasi.Errno {
	d, errno := p.lookupFD(fd, wasi.PathSymlinkRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := unix.Symlinkat(oldPath, d.fd, newPath)
	return makeErrno(err)
}

func (p *Provider) PathUnlinkFile(ctx context.Context, fd wasi.FD, path string) wasi.Errno {
	d, errno := p.lookupFD(fd, wasi.PathUnlinkFileRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	err := unix.Unlinkat(d.fd, path, 0)
	return makeErrno(err)
}

func (p *Provider) PollOneOff(ctx context.Context, subscriptions []wasi.Subscription, events []wasi.Event) ([]wasi.Event, wasi.Errno) {
	if len(subscriptions) == 0 {
		return events, wasi.EINVAL
	}
	timeout := time.Duration(-1)
	p.pollfds = p.pollfds[:0]
	for i := range subscriptions {
		s := &subscriptions[i]

		switch s.EventType {
		case wasi.FDReadEvent, wasi.FDWriteEvent:
			f, errno := p.lookupFD(s.GetFDReadWrite().FD, wasi.PollFDReadWriteRight)
			if errno != wasi.ESUCCESS {
				// TODO: set the error on the event instead of aborting the call
				return events, errno
			}
			var pollevent int16 = unix.POLLIN
			if s.EventType == wasi.FDWriteEvent {
				pollevent = unix.POLLOUT
			}
			p.pollfds = append(p.pollfds, unix.PollFd{
				Fd:     int32(f.fd),
				Events: pollevent,
			})
		case wasi.ClockEvent:
			c := s.GetClock()
			switch {
			case c.ID != wasi.Monotonic || c.Flags.Has(wasi.Abstime):
				return events, wasi.ENOSYS // not implemented
			case timeout < 0:
				timeout = time.Duration(c.Timeout)
			case timeout >= 0 && time.Duration(c.Timeout) < timeout:
				timeout = time.Duration(c.Timeout)
			}
		}
	}

	if len(p.pollfds) == 0 {
		// Just sleep if there's no FD events to poll.
		if timeout >= 0 {
			t := time.NewTimer(timeout)
			defer t.Stop()
			select {
			case <-t.C:
			case <-ctx.Done():
				return events, makeErrno(ctx.Err())
			}
		}
		return events, wasi.ESUCCESS
	}

	var timeoutMillis int
	if timeout < 0 {
		timeoutMillis = -1
	} else {
		timeoutMillis = int(timeout.Milliseconds())
	}
	// TODO: allow ctx to unblock when canceled
	n, err := unix.Poll(p.pollfds, timeoutMillis)
	if err != nil {
		return events, makeErrno(err)
	}

	j := 0
	for i := range subscriptions {
		s := &subscriptions[i]
		if s.EventType == wasi.ClockEvent {
			continue
		}
		pf := &p.pollfds[j]
		j++
		if pf.Revents == 0 {
			continue
		}
		e := wasi.Event{UserData: s.UserData, EventType: s.EventType}

		// TODO: review cases where Revents contains many flags
		if s.EventType == wasi.FDReadEvent && (pf.Revents&unix.POLLIN) != 0 {
			e.FDReadWrite.NBytes = 1 // we don't know how many, so just say 1
		}
		if s.EventType == wasi.FDWriteEvent && (pf.Revents&unix.POLLOUT) != 0 {
			e.FDReadWrite.NBytes = 1 // we don't know how many, so just say 1
		}
		if (pf.Revents & unix.POLLERR) != 0 {
			e.Errno = wasi.ECANCELED // we don't know what error, just pass something
		}
		if (pf.Revents & unix.POLLHUP) != 0 {
			e.FDReadWrite.Flags |= wasi.Hangup
		}
		events = append(events, e)
	}
	if n != len(events) {
		panic("unexpected unix.Poll result")
	}
	return events, wasi.ESUCCESS
}

func (p *Provider) ProcExit(ctx context.Context, code wasi.ExitCode) wasi.Errno {
	if p.Exit != nil {
		return makeErrno(p.Exit(ctx, int(code)))
	}
	return wasi.ENOSYS
}

func (p *Provider) ProcRaise(ctx context.Context, signal wasi.Signal) wasi.Errno {
	if p.Raise != nil {
		return makeErrno(p.Raise(ctx, int(signal)))
	}
	return wasi.ENOSYS
}

func (p *Provider) SchedYield(ctx context.Context) wasi.Errno {
	if p.Yield != nil {
		return makeErrno(p.Yield(ctx))
	}
	return wasi.ENOSYS
}

func (p *Provider) RandomGet(ctx context.Context, b []byte) wasi.Errno {
	if _, err := io.ReadFull(p.Rand, b); err != nil {
		return wasi.EIO
	}
	return wasi.ESUCCESS
}

func (p *Provider) SockAccept(ctx context.Context, fd wasi.FD, flags wasi.FDFlags) (wasi.FD, wasi.Errno) {
	socket, errno := p.lookupSocketFD(fd, wasi.SockAcceptRight)
	if errno != wasi.ESUCCESS {
		return -1, errno
	}
	if (flags & ^wasi.NonBlock) != 0 {
		return -1, wasi.EINVAL
	}
	// TODO: use accept4 on linux to set O_CLOEXEC and O_NONBLOCK
	connfd, _, err := unix.Accept(socket.fd)
	if err != nil {
		return -1, makeErrno(err)
	}
	if err := unix.SetNonblock(connfd, flags.Has(wasi.NonBlock)); err != nil {
		unix.Close(connfd)
		return -1, makeErrno(err)
	}
	guestfd := p.fds.Insert(&fdinfo{
		fd: connfd,
		stat: wasi.FDStat{
			FileType:         wasi.SocketStreamType,
			Flags:            flags,
			RightsBase:       socket.stat.RightsInheriting,
			RightsInheriting: socket.stat.RightsInheriting,
		},
	})
	return guestfd, wasi.ESUCCESS
}

func (p *Provider) SockRecv(ctx context.Context, fd wasi.FD, iovecs []wasi.IOVec, flags wasi.RIFlags) (wasi.Size, wasi.ROFlags, wasi.Errno) {
	socket, errno := p.lookupSocketFD(fd, wasi.FDReadRight)
	if errno != wasi.ESUCCESS {
		return 0, 0, errno
	}
	_ = socket
	return 0, 0, wasi.ENOSYS // TODO: implement SockRecv
}

func (p *Provider) SockSend(ctx context.Context, fd wasi.FD, iovecs []wasi.IOVec, flags wasi.SIFlags) (wasi.Size, wasi.Errno) {
	socket, errno := p.lookupSocketFD(fd, wasi.FDWriteRight)
	if errno != wasi.ESUCCESS {
		return 0, errno
	}
	_ = socket
	return 0, wasi.ENOSYS // TODO: implement SockSend
}

func (p *Provider) SockShutdown(ctx context.Context, fd wasi.FD, flags wasi.SDFlags) wasi.Errno {
	socket, errno := p.lookupSocketFD(fd, wasi.SockShutdownRight)
	if errno != wasi.ESUCCESS {
		return errno
	}
	var sysHow int
	switch {
	case flags.Has(wasi.ShutdownRD | wasi.ShutdownWR):
		sysHow = unix.SHUT_RDWR
	case flags.Has(wasi.ShutdownRD):
		sysHow = unix.SHUT_RD
	case flags.Has(wasi.ShutdownWR):
		sysHow = unix.SHUT_WR
	default:
		return wasi.EINVAL
	}
	err := unix.Shutdown(socket.fd, sysHow)
	return makeErrno(err)
}

func (p *Provider) Close(ctx context.Context) error {
	p.fds.Range(func(fd wasi.FD, f *fdinfo) bool {
		unix.Close(f.fd)
		return true
	})
	p.fds.Reset()
	p.preopens.Reset()
	return nil
}

type statDirEntry struct {
	name string
	info os.FileInfo
}

func (d *statDirEntry) Name() string               { return d.name }
func (d *statDirEntry) IsDir() bool                { return d.info.IsDir() }
func (d *statDirEntry) Type() os.FileMode          { return d.info.Mode().Type() }
func (d *statDirEntry) Info() (os.FileInfo, error) { return d.info, nil }
