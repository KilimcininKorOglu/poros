//go:build linux || darwin || freebsd || netbsd || openbsd

package probe

import (
	"syscall"
)

// setIPv4TTL sets the TTL for an IPv4 socket on Unix systems.
func setIPv4TTL(fd uintptr, ttl int) error {
	return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, syscall.IP_TTL, ttl)
}

// setIPv6HopLimit sets the hop limit for an IPv6 socket on Unix systems.
func setIPv6HopLimit(fd uintptr, hopLimit int) error {
	return syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IPV6, syscall.IPV6_UNICAST_HOPS, hopLimit)
}
