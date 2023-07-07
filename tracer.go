package wasi

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Trace wraps a System to log all calls to its methods in a human-readable
// format to the given io.Writer.
func Trace(w io.Writer, s System) System {
	return &tracer{writer: w, system: s}
}

type tracer struct {
	writer io.Writer
	system System
}

func (t *tracer) ArgsSizesGet(ctx context.Context) (int, int, Errno) {
	t.printf("ArgsSizesGet() => ")
	argCount, stringBytes, errno := t.system.ArgsSizesGet(ctx)
	if errno == ESUCCESS {
		t.printf("%d, %d", argCount, stringBytes)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return argCount, stringBytes, errno
}

func (t *tracer) ArgsGet(ctx context.Context) ([]string, Errno) {
	t.printf("ArgsGet() => ")
	args, errno := t.system.ArgsGet(ctx)
	if errno == ESUCCESS {
		t.printf("%q", args)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return args, errno
}

func (t *tracer) EnvironSizesGet(ctx context.Context) (int, int, Errno) {
	t.printf("EnvironSizesGet() => ")
	envCount, stringBytes, errno := t.system.EnvironSizesGet(ctx)
	if errno == ESUCCESS {
		t.printf("%d, %d", envCount, stringBytes)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return envCount, stringBytes, errno
}

func (t *tracer) EnvironGet(ctx context.Context) ([]string, Errno) {
	t.printf("EnvironGet() => ")
	environ, errno := t.system.EnvironGet(ctx)
	if errno == ESUCCESS {
		t.printf("%q", environ)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return environ, errno
}

func (t *tracer) ClockResGet(ctx context.Context, id ClockID) (Timestamp, Errno) {
	t.printf("ClockResGet(%d) => ", id)
	precision, errno := t.system.ClockResGet(ctx, id)
	if errno == ESUCCESS {
		t.printf("%d", precision)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return precision, errno
}

func (t *tracer) ClockTimeGet(ctx context.Context, id ClockID, precision Timestamp) (Timestamp, Errno) {
	t.printf("ClockTimeGet(%d, %d) => ", id, precision)
	timestamp, errno := t.system.ClockTimeGet(ctx, id, precision)
	if errno == ESUCCESS {
		t.printf("%d", timestamp)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return timestamp, errno
}

func (t *tracer) FDAdvise(ctx context.Context, fd FD, offset, length FileSize, advice Advice) Errno {
	t.printf("FDAdvise(%d, %d, %d, %s) => ", fd, offset, length, advice)
	errno := t.system.FDAdvise(ctx, fd, offset, length, advice)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDAllocate(ctx context.Context, fd FD, offset, length FileSize) Errno {
	t.printf("FDAllocate(%d, %d, %d) => ", fd, offset, length)
	errno := t.system.FDAllocate(ctx, fd, offset, length)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDClose(ctx context.Context, fd FD) Errno {
	t.printf("FDClose(%d) => ", fd)
	errno := t.system.FDClose(ctx, fd)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDDataSync(ctx context.Context, fd FD) Errno {
	t.printf("FDDataSync(%d) => ", fd)
	errno := t.system.FDDataSync(ctx, fd)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDStatGet(ctx context.Context, fd FD) (FDStat, Errno) {
	t.printf("FDStatGet(%d) => ", fd)
	fdstat, errno := t.system.FDStatGet(ctx, fd)
	if errno == ESUCCESS {
		t.printFDStat(fdstat)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return fdstat, errno
}

func (t *tracer) FDStatSetFlags(ctx context.Context, fd FD, flags FDFlags) Errno {
	t.printf("FDStatSetFlags(%d, %s) => ", fd, flags)
	errno := t.system.FDStatSetFlags(ctx, fd, flags)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDStatSetRights(ctx context.Context, fd FD, rightsBase, rightsInheriting Rights) Errno {
	t.printf("FDStatSetRights(%d, %s, %s) => ", fd, rightsBase, rightsInheriting)
	errno := t.system.FDStatSetRights(ctx, fd, rightsBase, rightsInheriting)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDFileStatGet(ctx context.Context, fd FD) (FileStat, Errno) {
	t.printf("FDFileStatGet(%d) => ", fd)
	filestat, errno := t.system.FDFileStatGet(ctx, fd)
	if errno == ESUCCESS {
		t.printFileStat(filestat)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return filestat, errno
}

func (t *tracer) FDFileStatSetSize(ctx context.Context, fd FD, size FileSize) Errno {
	t.printf("FDFileStatSetSize(%d, %d) => ", fd, size)
	errno := t.system.FDFileStatSetSize(ctx, fd, size)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDFileStatSetTimes(ctx context.Context, fd FD, accessTime, modifyTime Timestamp, flags FSTFlags) Errno {
	t.printf("FDFileStatSetTimes(%d, %d, %d, %s) => ", fd, accessTime, modifyTime, flags)
	errno := t.system.FDFileStatSetTimes(ctx, fd, accessTime, modifyTime, flags)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDPread(ctx context.Context, fd FD, iovecs []IOVec, offset FileSize) (Size, Errno) {
	t.printf("FDPread(%d, ", fd)
	t.printIOVecsProto(iovecs)
	t.printf("%d) => ", offset)
	n, errno := t.system.FDPread(ctx, fd, iovecs, offset)
	if errno == ESUCCESS {
		t.printf("[%d]byte: ", n)
		t.printIOVecs(iovecs, int(n))
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) FDPreStatGet(ctx context.Context, fd FD) (PreStat, Errno) {
	t.printf("FDPreStatGet(%d) => ", fd)
	prestat, errno := t.system.FDPreStatGet(ctx, fd)
	if errno == ESUCCESS {
		t.printf("{Type:%s,PreStatDir.NameLength:%d}", prestat.Type, prestat.PreStatDir.NameLength)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return prestat, errno
}

func (t *tracer) FDPreStatDirName(ctx context.Context, fd FD) (string, Errno) {
	t.printf("FDPreStatDirName(%d) => ", fd)
	name, errno := t.system.FDPreStatDirName(ctx, fd)
	if errno == ESUCCESS {
		t.printf("%q", name)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return name, errno
}

func (t *tracer) FDPwrite(ctx context.Context, fd FD, iovecs []IOVec, offset FileSize) (Size, Errno) {
	t.printf("FDPwrite(%d, ", fd)
	t.printIOVecs(iovecs, -1)
	t.printf(", %d) => ", offset)
	n, errno := t.system.FDPwrite(ctx, fd, iovecs, offset)
	if errno == ESUCCESS {
		t.printf("%d", n)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) FDRead(ctx context.Context, fd FD, iovecs []IOVec) (Size, Errno) {
	t.printf("FDRead(%d, ", fd)
	t.printIOVecsProto(iovecs)
	t.printf(") => ")
	n, errno := t.system.FDRead(ctx, fd, iovecs)
	if errno == ESUCCESS {
		t.printf("[%d]byte: ", n)
		t.printIOVecs(iovecs, int(n))
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) FDReadDir(ctx context.Context, fd FD, entries []DirEntry, cookie DirCookie, bufferSizeBytes int) (int, Errno) {
	t.printf("FDReadDir(%d, %d) => ", fd, cookie)
	n, errno := t.system.FDReadDir(ctx, fd, entries, cookie, bufferSizeBytes)
	if errno == ESUCCESS {
		t.printDirEntries(entries[:n], bufferSizeBytes)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) FDRenumber(ctx context.Context, from, to FD) Errno {
	t.printf("FDRenumber(%d, %d) => ", from, to)
	errno := t.system.FDRenumber(ctx, from, to)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDSeek(ctx context.Context, fd FD, offset FileDelta, whence Whence) (FileSize, Errno) {
	t.printf("FDSeek(%d, %d, %s) => ", fd, offset, whence)
	result, errno := t.system.FDSeek(ctx, fd, offset, whence)
	if errno == ESUCCESS {
		t.printf("%d", offset)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return result, errno
}

func (t *tracer) FDSync(ctx context.Context, fd FD) Errno {
	t.printf("FDSync(%d) => ", fd)
	errno := t.system.FDSync(ctx, fd)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) FDTell(ctx context.Context, fd FD) (FileSize, Errno) {
	t.printf("FDTell(%d) => ", fd)
	fileSize, errno := t.system.FDTell(ctx, fd)
	if errno == ESUCCESS {
		t.printf("%d", fileSize)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return fileSize, errno
}

func (t *tracer) FDWrite(ctx context.Context, fd FD, iovecs []IOVec) (Size, Errno) {
	t.printf("FDWrite(%d, ", fd)
	t.printIOVecs(iovecs, -1)
	t.printf(") => ")
	n, errno := t.system.FDWrite(ctx, fd, iovecs)
	if errno == ESUCCESS {
		t.printf("%d", n)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) PathCreateDirectory(ctx context.Context, fd FD, path string) Errno {
	t.printf("PathCreateDirectory(%d, %q) => ", fd, path)
	errno := t.system.PathCreateDirectory(ctx, fd, path)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PathFileStatGet(ctx context.Context, fd FD, lookupFlags LookupFlags, path string) (FileStat, Errno) {
	t.printf("PathFileStatGet(%d, %s, %q) => ", fd, lookupFlags, path)
	filestat, errno := t.system.PathFileStatGet(ctx, fd, lookupFlags, path)
	if errno == ESUCCESS {
		t.printFileStat(filestat)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return filestat, errno
}

func (t *tracer) PathFileStatSetTimes(ctx context.Context, fd FD, lookupFlags LookupFlags, path string, accessTime, modifyTime Timestamp, flags FSTFlags) Errno {
	t.printf("PathFileStatSetTimes(%d, %s, %q, %d, %d, %s) => ", fd, lookupFlags, path, accessTime, modifyTime, flags)
	errno := t.system.PathFileStatSetTimes(ctx, fd, lookupFlags, path, accessTime, modifyTime, flags)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PathLink(ctx context.Context, oldFD FD, oldFlags LookupFlags, oldPath string, newFD FD, newPath string) Errno {
	t.printf("PathLink(%d, %s, %q, %d, %q) => ", oldFD, oldFlags, oldPath, newFD, newPath)
	errno := t.system.PathLink(ctx, oldFD, oldFlags, oldPath, newFD, newPath)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PathOpen(ctx context.Context, fd FD, dirFlags LookupFlags, path string, openFlags OpenFlags, rightsBase, rightsInheriting Rights, fdFlags FDFlags) (FD, Errno) {
	t.printf("PathOpen(%d, %s, %q, %s, %s, %s, %s) => ", fd, dirFlags, path, openFlags, rightsBase, rightsInheriting, fdFlags)
	fd, errno := t.system.PathOpen(ctx, fd, dirFlags, path, openFlags, rightsBase, rightsInheriting, fdFlags)
	if errno == ESUCCESS {
		t.printf("%d", fd)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return fd, errno
}

func (t *tracer) PathReadLink(ctx context.Context, fd FD, path string, buffer []byte) (int, Errno) {
	t.printf("PathReadLink(%d, %q, [%d]byte) => ", fd, path, len(buffer))
	n, errno := t.system.PathReadLink(ctx, fd, path, buffer)
	if errno == ESUCCESS {
		t.printBytes(buffer[:n])
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) PathRemoveDirectory(ctx context.Context, fd FD, path string) Errno {
	t.printf("PathRemoveDirectory(%d, %q) => ", fd, path)
	errno := t.system.PathRemoveDirectory(ctx, fd, path)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PathRename(ctx context.Context, fd FD, oldPath string, newFD FD, newPath string) Errno {
	t.printf("PathRename(%d, %q, %d, %q) => ", fd, oldPath, newFD, newPath)
	errno := t.system.PathRename(ctx, fd, oldPath, newFD, newPath)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PathSymlink(ctx context.Context, oldPath string, fd FD, newPath string) Errno {
	t.printf("PathSymlink(%q, %d, %q) => ", oldPath, fd, newPath)
	errno := t.system.PathSymlink(ctx, oldPath, fd, newPath)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PathUnlinkFile(ctx context.Context, fd FD, path string) Errno {
	t.printf("PathUnlinkFile(%d, %q) => ", fd, path)
	errno := t.system.PathUnlinkFile(ctx, fd, path)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) PollOneOff(ctx context.Context, subscriptions []Subscription, events []Event) (int, Errno) {
	t.printf("PollOneoff(")
	for i, s := range subscriptions {
		if i > 0 {
			t.printf(",")
		}
		t.printSubscription(s)
	}
	t.printf(") => ")
	n, errno := t.system.PollOneOff(ctx, subscriptions, events)
	switch {
	case errno == ESUCCESS && n == 0:
		t.printf("{}")
	case errno == ESUCCESS:
		for i, e := range events[:n] {
			if i > 0 {
				t.printf(",")
			}
			t.printEvent(e)
		}
	default:
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) ProcExit(ctx context.Context, exitCode ExitCode) Errno {
	t.printf("ProcExit(%d) => ", exitCode)
	errno := t.system.ProcExit(ctx, exitCode)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) ProcRaise(ctx context.Context, signal Signal) Errno {
	t.printf("ProcRaise(%d) => ", signal)
	errno := t.system.ProcRaise(ctx, signal)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) SchedYield(ctx context.Context) Errno {
	t.printf("SchedYield() => ")
	errno := t.system.SchedYield(ctx)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) RandomGet(ctx context.Context, b []byte) Errno {
	t.printf("RandomGet([%d]byte) => ", len(b))
	errno := t.system.RandomGet(ctx, b)
	if errno == ESUCCESS {
		t.printBytes(b)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) SockAccept(ctx context.Context, fd FD, flags FDFlags) (FD, SocketAddress, SocketAddress, Errno) {
	t.printf("SockAccept(%d, %s) => ", fd, flags)
	newfd, peer, addr, errno := t.system.SockAccept(ctx, fd, flags)
	if errno == ESUCCESS {
		t.printf("%d, %s > %s", newfd, peer, addr)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return newfd, peer, addr, errno
}

func (t *tracer) SockShutdown(ctx context.Context, fd FD, flags SDFlags) Errno {
	t.printf("SockShutdown(%d, %s) => ", fd, flags)
	errno := t.system.SockShutdown(ctx, fd, flags)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) SockRecv(ctx context.Context, fd FD, iovecs []IOVec, iflags RIFlags) (Size, ROFlags, Errno) {
	t.printf("SockRecv(%d, ", fd)
	t.printIOVecsProto(iovecs)
	t.printf(", %s) => ", iflags)
	n, oflags, errno := t.system.SockRecv(ctx, fd, iovecs, iflags)
	if errno == ESUCCESS {
		t.printf("[%d]byte: ", n)
		t.printIOVecs(iovecs, int(n))
		t.printf(", %s", oflags)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, oflags, errno
}

func (t *tracer) SockSend(ctx context.Context, fd FD, iovecs []IOVec, iflags SIFlags) (Size, Errno) {
	t.printf("SockSend(%d, ", fd)
	t.printIOVecs(iovecs, -1)
	t.printf(", %s) => ", iflags)
	n, errno := t.system.SockSend(ctx, fd, iovecs, iflags)
	if errno == ESUCCESS {
		t.printf("%d", n)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) SockOpen(ctx context.Context, pf ProtocolFamily, socketType SocketType, protocol Protocol, rightsBase, rightsInheriting Rights) (FD, Errno) {
	t.printf("SockOpen(%s, %s, %s, %s, %s) => ", pf, socketType, protocol, rightsBase, rightsInheriting)
	fd, errno := t.system.SockOpen(ctx, pf, socketType, protocol, rightsBase, rightsInheriting)
	if errno == ESUCCESS {
		t.printf("%d", fd)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return fd, errno
}

func (t *tracer) SockBind(ctx context.Context, fd FD, addr SocketAddress) (SocketAddress, Errno) {
	t.printf("SockBind(%d, %s) => ", fd, addr)
	addr, errno := t.system.SockBind(ctx, fd, addr)
	if errno == ESUCCESS {
		t.printf("%s", addr)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return addr, errno
}

func (t *tracer) SockConnect(ctx context.Context, fd FD, peer SocketAddress) (SocketAddress, Errno) {
	t.printf("SockConnect(%d, %s) => ", fd, peer)
	addr, errno := t.system.SockConnect(ctx, fd, peer)
	if errno == EINPROGRESS {
		t.printf("%s (EINPROGRESS)", addr)
	} else if errno == ESUCCESS {
		t.printf("%s", addr)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return addr, errno
}

func (t *tracer) SockListen(ctx context.Context, fd FD, backlog int) Errno {
	t.printf("SockListen(%d, %d) => ", fd, backlog)
	errno := t.system.SockListen(ctx, fd, backlog)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) SockSendTo(ctx context.Context, fd FD, iovecs []IOVec, iflags SIFlags, addr SocketAddress) (Size, Errno) {
	t.printf("SockSendTo(%d, ", fd)
	t.printIOVecs(iovecs, -1)
	t.printf(", %s, %s) => ", iflags, addr)
	n, errno := t.system.SockSendTo(ctx, fd, iovecs, iflags, addr)
	if errno == ESUCCESS {
		t.printf("%d", n)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) SockRecvFrom(ctx context.Context, fd FD, iovecs []IOVec, iflags RIFlags) (Size, ROFlags, SocketAddress, Errno) {
	t.printf("SockRecvFrom(%d, ", fd)
	t.printIOVecsProto(iovecs)
	t.printf(", %s) => ", iflags)
	n, oflags, addr, errno := t.system.SockRecvFrom(ctx, fd, iovecs, iflags)
	if errno == ESUCCESS {
		t.printf("[%d]byte: ", n)
		t.printIOVecs(iovecs, int(n))
		t.printf(", %s, %s", oflags, addr)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, oflags, addr, errno
}

func (t *tracer) SockGetOpt(ctx context.Context, fd FD, option SocketOption) (SocketOptionValue, Errno) {
	t.printf("SockGetOpt(%d, %s) => ", fd, option)
	value, errno := t.system.SockGetOpt(ctx, fd, option)
	if errno == ESUCCESS {
		t.printf("%d", value)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return value, errno
}

func (t *tracer) SockSetOpt(ctx context.Context, fd FD, option SocketOption, value SocketOptionValue) Errno {
	t.printf("SockSetOpt(%d, %s, %s) => ", fd, option, value)
	errno := t.system.SockSetOpt(ctx, fd, option, value)
	if errno == ESUCCESS {
		t.printf("ok")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return errno
}

func (t *tracer) SockLocalAddress(ctx context.Context, fd FD) (SocketAddress, Errno) {
	t.printf("SockLocalAddress(%d) => ", fd)
	addr, errno := t.system.SockLocalAddress(ctx, fd)
	if errno == ESUCCESS {
		t.printf("%s", addr)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return addr, errno
}

func (t *tracer) SockRemoteAddress(ctx context.Context, fd FD) (SocketAddress, Errno) {
	t.printf("SockRemoteAddress(%d) => ", fd)
	addr, errno := t.system.SockRemoteAddress(ctx, fd)
	if errno == ESUCCESS {
		t.printf("%s", addr)
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return addr, errno
}

func (t *tracer) SockAddressInfo(ctx context.Context, name, service string, hints AddressInfo, results []AddressInfo) (int, Errno) {
	t.printf("SockAddressInfo(%s, %s, ", name, service)
	t.printAddressInfo(hints)
	t.printf(", [%d]AddressInfo) => ", len(results))
	n, errno := t.system.SockAddressInfo(ctx, name, service, hints, results)
	if errno == ESUCCESS {
		t.printf("[")
		for i := range results[:n] {
			if i > 0 {
				t.printf(", ")
			}
			t.printAddressInfo(results[i])
		}
		t.printf("]")
	} else {
		t.printErrno(errno)
	}
	t.printf("\n")
	return n, errno
}

func (t *tracer) Close(ctx context.Context) error {
	t.printf("Close() => ")
	err := t.system.Close(ctx)
	if err == nil {
		t.printf("ok\n")
	} else {
		t.printf("error (%s)\n", err)
	}
	return err
}

func (t *tracer) printf(msg string, args ...interface{}) {
	fmt.Fprintf(t.writer, msg, args...)
}

func (t *tracer) printErrno(errno Errno) {
	t.printf("%s (%s)", errno.Name(), errno.Error())
}

func (t *tracer) printSubscription(s Subscription) {
	t.printf("{EventType:%s,UserData:%#x,", s.EventType, s.UserData)
	if s.EventType == ClockEvent {
		c := s.GetClock()
		t.printf("ID:%s,", c.ID)
		if c.Flags != 0 {
			t.printf("Flags:%s,", c.Flags)
		}
		t.printf("Timeout:%d,Precision:%d}", c.Timeout, c.Precision)
	} else {
		fdrw := s.GetFDReadWrite()
		t.printf("FD:%d}", fdrw.FD)
	}
}

func (t *tracer) printEvent(e Event) {
	t.printf("{EventType:%s,UserData:%#x", e.EventType, e.UserData)
	if e.Errno != 0 {
		t.printf(",Errno:%s}", e.Errno.Name())
	}
	if e.EventType != ClockEvent {
		fdrw := e.FDReadWrite
		if fdrw.Flags != 0 {
			t.printf(",Flags:%s", fdrw.Flags)
		}
		t.printf(",NBytes:%d}", fdrw.NBytes)
	}
}

func (t *tracer) printFDStat(s FDStat) {
	t.printf("{FileType:%s", s.FileType)
	if s.Flags != 0 {
		t.printf(",Flags:%s", s.Flags)
	}
	t.printf(",RightsBase:%s", s.RightsBase)
	if s.RightsInheriting != 0 {
		t.printf(",RightsInheriting:%s", s.RightsInheriting)
	}
	t.printf("}")
}

func (t *tracer) printFileStat(s FileStat) {
	t.printf("%#v", s)
}

func (t *tracer) printIOVecsProto(iovecs []IOVec) {
	t.printf("[%d]IOVec{", len(iovecs))
	for i, iovec := range iovecs {
		if i > 0 {
			t.printf(",")
		}
		t.printf("[%d]Byte", len(iovec))
	}
	t.printf("}")
}

func (t *tracer) printIOVecs(iovecs []IOVec, size int) {
	t.printf("[%d]IOVec{", len(iovecs))
	for i, iovec := range iovecs {
		if i > 0 {
			t.printf(",")
		}
		switch {
		case size < 0:
			t.printBytes(iovec)
		case size > 0 && len(iovec) > size:
			t.printBytes(iovec[:size])
			size = 0
		case size > 0 && len(iovec) <= size:
			t.printBytes(iovec)
			size -= len(iovec)
		case size == 0:
			t.printf("[%d]Byte", len(iovec))
		}
	}
	t.printf("}")
}

func (t *tracer) printDirEntries(dirEntries []DirEntry, bufferSizeBytes int) {
	t.printf("{")
	for i, e := range dirEntries {
		if i > 0 {
			t.printf("},{")
		}
		t.printf("Name:%q,Type:%s,INode:%d,Next:%d", string(e.Name), e.Type, e.INode, e.Next)
		bufferSizeBytes -= SizeOfDirent + len(e.Name)
		if bufferSizeBytes < 0 {
			t.printf(",Partial")
		}
	}
	t.printf("}")
}

func (t *tracer) printAddressInfo(a AddressInfo) {
	t.printf("{")
	if a.Flags != 0 {
		t.printf("Flags:%s,", a.Flags)
	}
	t.printf("Family:%s,SocketType:%s,Protocol:%s", a.Family, a.SocketType, a.Protocol)
	if a.Address != nil {
		t.printf(",Address:%s", a.Address)
	}
	if a.CanonicalName != "" {
		t.printf(",CanonicalName:%q", a.CanonicalName)
	}
	t.printf("}")
}

var maxBytes = 32

func init() {
	if m := os.Getenv("TRACER_MAX_BYTES"); m != "" {
		if i, err := strconv.Atoi(m); err == nil {
			maxBytes = i
		}
	}
}

func (t *tracer) printBytes(b []byte) {
	t.printf("[%d]byte(\"", len(b))

	if len(b) > 0 {
		trunc := b
		if maxBytes > 0 && len(b) > maxBytes {
			trunc = trunc[:maxBytes]
		}
		for _, c := range trunc {
			if c < 32 || c >= 127 || c == '"' {
				t.printf("\\")
				switch {
				case c <= 9:
					t.printf("%d", c)
				case c == '"':
					t.printf("\"")
				case c == '\\':
					t.printf("\\")
				case c == '\r':
					t.printf("r")
				case c == '\n':
					t.printf("n")
				case c == '\t':
					t.printf("t")
				default:
					t.printf(`x%02x`, c)
				}
			} else {
				t.printf(`%c`, c)
			}
		}
	}
	t.printf("\"")
	if maxBytes > 0 && len(b) > maxBytes {
		t.printf("...")
	}
	t.printf(")")
}
