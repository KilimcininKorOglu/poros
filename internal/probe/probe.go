// Package probe provides network probing implementations for traceroute.
package probe

import (
	"context"
	"net"
	"time"
)

// Prober defines the interface for different probe methods.
// Implementations include ICMP, UDP, TCP, and Paris traceroute.
type Prober interface {
	// Probe sends a probe packet with the given TTL and returns the result.
	// The dest parameter is the target IP address.
	// The ttl parameter is the Time-To-Live value for the IP packet.
	// Returns a ProbeResult on success, or an error on failure.
	Probe(ctx context.Context, dest net.IP, ttl int) (*Result, error)

	// Name returns the probe method name (e.g., "icmp", "udp", "tcp").
	Name() string

	// RequiresRoot returns true if this probe method requires root/admin privileges.
	RequiresRoot() bool

	// Close releases any resources held by the prober.
	Close() error
}

// Result contains the result of a single probe.
type Result struct {
	// ResponseIP is the IP address that responded
	ResponseIP net.IP

	// RTT is the round-trip time
	RTT time.Duration

	// ICMPType is the ICMP message type (for ICMP-based responses)
	ICMPType int

	// ICMPCode is the ICMP message code
	ICMPCode int

	// Reached indicates if the destination was reached
	// (Echo Reply for ICMP, Port Unreachable for UDP, SYN-ACK/RST for TCP)
	Reached bool

	// TTLExpired indicates if the response was a TTL exceeded message
	TTLExpired bool
}

// Method represents the type of probe to use.
type Method int

const (
	// MethodICMP uses ICMP Echo Request packets
	MethodICMP Method = iota
	// MethodUDP uses UDP packets to high ports
	MethodUDP
	// MethodTCP uses TCP SYN packets
	MethodTCP
)

// String returns the string representation of the probe method.
func (m Method) String() string {
	switch m {
	case MethodICMP:
		return "icmp"
	case MethodUDP:
		return "udp"
	case MethodTCP:
		return "tcp"
	default:
		return "unknown"
	}
}
