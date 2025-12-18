package trace

import (
	"net"
	"time"
)

// ProbeMethod represents the type of probe to use.
type ProbeMethod int

const (
	// ProbeICMP uses ICMP Echo Request packets
	ProbeICMP ProbeMethod = iota
	// ProbeUDP uses UDP packets to high ports
	ProbeUDP
	// ProbeTCP uses TCP SYN packets
	ProbeTCP
	// ProbeParis uses Paris traceroute algorithm
	ProbeParis
)

// String returns the string representation of the probe method.
func (p ProbeMethod) String() string {
	switch p {
	case ProbeICMP:
		return "icmp"
	case ProbeUDP:
		return "udp"
	case ProbeTCP:
		return "tcp"
	case ProbeParis:
		return "paris"
	default:
		return "unknown"
	}
}

// Config holds the configuration for a trace operation.
type Config struct {
	// Probe settings
	ProbeMethod ProbeMethod   // Probe method to use (default: ICMP)
	ProbeCount  int           // Number of probes per hop (default: 3)
	MaxHops     int           // Maximum TTL/hops (default: 30)
	FirstHop    int           // Starting TTL (default: 1)
	Timeout     time.Duration // Per-probe timeout (default: 3s)

	// Network settings
	Interface string // Specific network interface to use
	SourceIP  net.IP // Source IP address to use
	DestPort  int    // Destination port (for UDP/TCP probes)
	IPv4      bool   // Force IPv4
	IPv6      bool   // Force IPv6

	// Mode settings
	Sequential     bool // Use sequential mode instead of concurrent
	MaxConcurrency int  // Maximum concurrent probes (default: 30)
	Paris          bool // Use Paris traceroute algorithm

	// Rate limiting
	PacketsPerSecond int // Rate limit (0 = unlimited)

	// Enrichment settings
	EnableEnrichment bool // Enable any enrichment
	EnableRDNS       bool // Enable reverse DNS lookup
	EnableASN        bool // Enable ASN lookup
	EnableGeoIP      bool // Enable GeoIP lookup

	// MaxMind database (optional, for offline/faster lookups)
	MaxMindDB interface{} // *enrich.MaxMindDB - use interface to avoid import cycle

	// Callback for real-time hop updates (streaming output)
	OnHop func(hop *Hop) // Called after each hop is probed
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		ProbeMethod:      ProbeICMP,
		ProbeCount:       3,
		MaxHops:          30,
		FirstHop:         1,
		Timeout:          3 * time.Second,
		DestPort:         33434, // Standard traceroute UDP port
		MaxConcurrency:   30,
		EnableEnrichment: true,
		EnableRDNS:       true,
		EnableASN:        true,
		EnableGeoIP:      true,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.MaxHops < 1 || c.MaxHops > 255 {
		return ErrInvalidMaxHops
	}
	if c.ProbeCount < 1 || c.ProbeCount > 10 {
		return ErrInvalidProbeCount
	}
	if c.Timeout < 100*time.Millisecond {
		return ErrInvalidTimeout
	}
	if c.FirstHop < 1 || c.FirstHop > c.MaxHops {
		return ErrInvalidFirstHop
	}
	return nil
}
