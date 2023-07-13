package wasi_snapshot_preview1

import (
	"context"
	"encoding/binary"
	"io"

	"github.com/stealthrocket/wasi-go"
	"github.com/stealthrocket/wazergo"
	. "github.com/stealthrocket/wazergo/types"
	"github.com/tetratelabs/wazero/api"
)

var Wasix = Extension{
	"sock_status":       wazergo.F2((*Module).WasixSockStatus),
	"sock_addr_local":   wazergo.F2((*Module).WasixSockAddrLocal),
	"sock_addr_peer":    wazergo.F2((*Module).WasixSockAddrPeer),
	"sock_open":         wazergo.F4((*Module).WasixSockOpen),
	"sock_set_opt_flag": wazergo.F3((*Module).WasixSockSetOptFlag),
	"sock_get_opt_flag": wazergo.F3((*Module).WasixSockGetOptFlag),
	"sock_set_opt_time": wazergo.F3((*Module).WasixSockSetOptTime),
	"sock_get_opt_time": wazergo.F3((*Module).WasixSockGetOptTime),
	"sock_set_opt_size": wazergo.F3((*Module).WasixSockSetOptSize),
	"sock_get_opt_size": wazergo.F3((*Module).WasixSockGetOptSize),
	//"sock_join_multicast_v4":  wazergo.F3((*Module).WasixSockJoinMulticastV4),
	//"sock_leave_multicast_v4": wazergo.F3((*Module).WasixSockLeaveMulticastV4),
	//"sock_join_multicast_v6":  wazergo.F3((*Module).WasixSockJoinMulticastV6),
	//"sock_leave_multicast_v6": wazergo.F3((*Module).WasixSockLeaveMulticastV6),
	"sock_bind":      wazergo.F2((*Module).WasixSockBind),
	"sock_listen":    wazergo.F2((*Module).WasixSockListen),
	"sock_accept_v2": wazergo.F4((*Module).WasixSockAcceptV2),
	"sock_connect":   wazergo.F2((*Module).WasixSockConnect),
	"sock_recv_from": wazergo.F6((*Module).WasixSockRecvFrom),
	"sock_send_to":   wazergo.F5((*Module).WasixSockSendTo),
	"sock_send_file": wazergo.F5((*Module).WasixSockSendFile),
	//"resolve":        wazergo.F6((*Module).WasixResolve),
}

func (m *Module) WasixSockStatus(ctx context.Context, fd Int32, sockStatus Pointer[Uint8]) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockAddrLocal(ctx context.Context, fd Int32, addrPort Pointer[wasixAddrPort]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	sa, errno := s.SockLocalAddress(ctx, wasi.FD(fd))
	if errno != wasi.ESUCCESS {
		return Errno(errno)
	}
	m.wasixPutAddrPort(addrPort.Load(), sa)
	return Errno(wasi.ESUCCESS)
}

func (m *Module) WasixSockAddrPeer(ctx context.Context, fd Int32, addrPort Pointer[wasixAddrPort]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	sa, errno := s.SockRemoteAddress(ctx, wasi.FD(fd))
	if errno != wasi.ESUCCESS {
		return Errno(errno)
	}
	m.wasixPutAddrPort(addrPort.Load(), sa)
	return Errno(wasi.ESUCCESS)
}

func (m *Module) WasixSockOpen(ctx context.Context, family Int32, sockType Int32, protocol Int32, openfd Pointer[Int32]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	rightsBase := wasi.SockListenRights | wasi.SockConnectionRights
	rightsInheriting := wasi.SockConnectionRights
	result, errno := s.SockOpen(ctx, wasi.ProtocolFamily(family), wasi.SocketType(sockType), wasi.Protocol(protocol), rightsBase, rightsInheriting)
	if errno != wasi.ESUCCESS {
		return Errno(errno)
	}
	openfd.Store(Int32(result))
	return Errno(wasi.ESUCCESS)
}

func (m *Module) WasixSockSetOptFlag(ctx context.Context, fd Int32, opt Int32, flag Uint32) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockGetOptFlag(ctx context.Context, fd Int32, opt Int32, value Pointer[Int32]) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockSetOptTime(ctx context.Context, fd Int32, opt Int32, timeout Pointer[wasixOptionTimestamp]) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockGetOptTime(ctx context.Context, fd Int32, opt Int32, timeout Pointer[wasixOptionTimestamp]) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockSetOptSize(ctx context.Context, fd Int32, opt Int32, size Uint64) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockGetOptSize(ctx context.Context, fd Int32, opt Int32, size Pointer[Uint64]) Errno {
	return Errno(wasi.ENOSYS)
}

func (m *Module) WasixSockBind(ctx context.Context, fd Int32, addr Pointer[wasixAddrPort]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	socketAddr, ok := m.wasixGetAddrPort(addr.Load())
	if !ok {
		return Errno(wasi.EINVAL)
	}
	_, errno := s.SockBind(ctx, wasi.FD(fd), socketAddr)
	return Errno(errno)
}

func (m *Module) WasixSockListen(ctx context.Context, fd Int32, backlog Int32) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	return Errno(s.SockListen(ctx, wasi.FD(fd), int(backlog)))
}

func (m *Module) WasixSockAcceptV2(ctx context.Context, fd Int32, flags Int32, roFd Pointer[Int32], roAddr Pointer[wasixAddrPort]) Errno {
	result, _, addr, errno := m.WASI.SockAccept(ctx, wasi.FD(fd), wasi.FDFlags(flags))
	if errno != wasi.ESUCCESS {
		return Errno(errno)
	}
	roFd.Store(Int32(result))
	if ok := m.wasixPutAddrPort(roAddr.Load(), addr); !ok {
		return Errno(wasi.EINVAL)
	}
	return Errno(errno)
}

func (m *Module) WasixSockConnect(ctx context.Context, fd Int32, addr Pointer[wasixAddrPort]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	socketAddr, ok := m.wasixGetAddrPort(addr.Load())
	if !ok {
		return Errno(wasi.EINVAL)
	}
	_, errno := s.SockConnect(ctx, wasi.FD(fd), socketAddr)
	return Errno(errno)
}

func (m *Module) WasixSockRecvFrom(ctx context.Context, fd Int32, iovecs List[wasi.IOVec], iflags Int32, odatalen Pointer[Uint32], oflags Pointer[Int32], oaddr Pointer[wasixAddrPort]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	m.iovecs = iovecs.Append(m.iovecs[:0])
	size, roflags, sa, errno := s.SockRecvFrom(ctx, wasi.FD(fd), m.iovecs, wasi.RIFlags(iflags))
	if errno != wasi.ESUCCESS {
		return Errno(errno)
	}
	if ok := m.wasixPutAddrPort(oaddr.Load(), sa); !ok {
		return Errno(wasi.EINVAL)
	}
	odatalen.Store(Uint32(size))
	oflags.Store(Int32(roflags))
	return Errno(wasi.ESUCCESS)
}

func (m *Module) WasixSockSendTo(ctx context.Context, fd Int32, iovecs List[wasi.IOVec], iflags Int32, addr Pointer[wasixAddrPort], nwritten Pointer[Int32]) Errno {
	s, ok := m.WASI.(wasi.SocketsExtension)
	if !ok {
		return Errno(wasi.ENOSYS)
	}
	socketAddr, ok := m.wasixGetAddrPort(addr.Load())
	if !ok {
		return Errno(wasi.EINVAL)
	}
	m.iovecs = iovecs.Append(m.iovecs[:0])
	size, errno := s.SockSendTo(ctx, wasi.FD(fd), m.iovecs, wasi.SIFlags(iflags), socketAddr)
	if errno != wasi.ESUCCESS {
		return Errno(errno)
	}
	nwritten.Store(Int32(size))
	return Errno(wasi.ESUCCESS)
}

func (m *Module) WasixSockSendFile(ctx context.Context, fd Int32, inFd Int32, offset Uint64, count Uint64, retSent Pointer[Int64]) Errno {
	return Errno(wasi.ENOSYS)
}

//func (m *Module) WasixResolve(ctx context.Context, host String, port Uint16, addr Pointer[wasixAddrPort], addrLen Pointer[Uint32]) Errno {
//	return Errno(wasi.ENOSYS)
//}

func (m *Module) wasixGetAddrPort(b wasixAddrPort) (sa wasi.SocketAddress, ok bool) {
	port := binary.LittleEndian.Uint16(b[1:3])

	switch wasi.ProtocolFamily(b[0]) {
	case wasi.Inet:
		b = b[3:7]
	case wasi.Inet6:
		b = b[3:19]
	}

	switch len(b) {
	case 4:
		m.inet4addr.Port = int(port)
		copy(m.inet4addr.Addr[:], b)
		sa = &m.inet4addr
		ok = true
	case 16:
		m.inet6addr.Port = int(port)
		copy(m.inet6addr.Addr[:], b)
		sa = &m.inet6addr
		ok = true
	}
	return
}

func (m *Module) wasixPutAddrPort(b wasixAddrPort, sa wasi.SocketAddress) (ok bool) {
	switch t := sa.(type) {
	case *wasi.Inet4Address:
		b[0] = byte(wasi.Inet)
		binary.LittleEndian.PutUint16(b[1:3], uint16(t.Port))
		copy(b[3:], t.Addr[:])
		ok = true

	case *wasi.Inet6Address:
		b[0] = byte(wasi.Inet6)
		binary.LittleEndian.PutUint16(b[1:3], uint16(t.Port))
		copy(b[3:], t.Addr[:])
		ok = true
	}
	return
}

// ;;; Union that makes a generic IP address and port
// (typename $addr_port
//
//	(union
//	  (@witx tag $address_family) u8
//	  $addr_unspec_port $ip_port:u16 $addr_unspec:u8
//	  $addr_ip4_port $ip_port:u16 $addr_ip4:u32(4*u8) 2 + 4
//	  $addr_ip6_port $ip_port:u16 $addr_ip6:u128(8*u16) 2 + 16
//	  $addr_unix_port $ip_port:u16 $addr_unix:u128(16*u8) 2 + 16
//	)
//
// )
type wasixAddrPort []byte

func (addr wasixAddrPort) ObjectSize() int {
	return 8
}

func (addr wasixAddrPort) LoadObject(memory api.Memory, object []byte) wasixAddrPort {
	return nil
}

func (addr wasixAddrPort) StoreObject(memory api.Memory, object []byte) {
	panic("BUG: socket addresses cannot be stored back to wasm memory")
}

func (addr wasixAddrPort) FormatObject(w io.Writer, memory api.Memory, object []byte) {
}

type wasixOptionTimestamp []byte

func (ts wasixOptionTimestamp) ObjectSize() int {
	return 0
}

func (ts wasixOptionTimestamp) LoadObject(memory api.Memory, object []byte) wasixOptionTimestamp {
	return wasixOptionTimestamp{}
}

func (ts wasixOptionTimestamp) StoreObject(memory api.Memory, object []byte) {
}

func (ts wasixOptionTimestamp) FormatObject(w io.Writer, memory api.Memory, object []byte) {
}
