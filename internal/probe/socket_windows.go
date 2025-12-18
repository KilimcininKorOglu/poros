//go:build windows

package probe

import (
	"syscall"
	"unsafe"
)

const (
	IPPROTO_IP   = 0
	IP_TTL       = 4
	IPPROTO_IPV6 = 41
	IPV6_UNICAST_HOPS = 4
)

// setIPv4TTL sets the TTL for an IPv4 socket on Windows.
func setIPv4TTL(fd uintptr, ttl int) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), IPPROTO_IP, IP_TTL, ttl)
}

// setIPv6HopLimit sets the hop limit for an IPv6 socket on Windows.
func setIPv6HopLimit(fd uintptr, hopLimit int) error {
	return syscall.SetsockoptInt(syscall.Handle(fd), IPPROTO_IPV6, IPV6_UNICAST_HOPS, hopLimit)
}

// setSocketOption is a helper for setting socket options on Windows.
func setSocketOption(fd uintptr, level, name int, value int) error {
	val := int32(value)
	return syscall.Setsockopt(
		syscall.Handle(fd),
		int32(level),
		int32(name),
		(*byte)(unsafe.Pointer(&val)),
		int32(unsafe.Sizeof(val)),
	)
}
