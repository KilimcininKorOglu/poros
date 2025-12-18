// Package trace provides traceroute functionality.
package trace

import (
	"net"
	"time"
)

// Hop represents a single hop in the trace path.
type Hop struct {
	// Number is the hop number (TTL value that triggered the response)
	Number int `json:"hop"`

	// IP is the IP address of the responding router/host
	IP net.IP `json:"ip,omitempty"`

	// Hostname is the reverse DNS name (if resolved)
	Hostname string `json:"hostname,omitempty"`

	// ASN contains Autonomous System information
	ASN *ASNInfo `json:"asn,omitempty"`

	// Geo contains geographic information
	Geo *GeoInfo `json:"geo,omitempty"`

	// RTTs contains individual round-trip times in milliseconds
	// A value of -1 indicates a timeout
	RTTs []float64 `json:"rtts"`

	// AvgRTT is the average RTT in milliseconds
	AvgRTT float64 `json:"avg_rtt"`

	// MinRTT is the minimum RTT in milliseconds
	MinRTT float64 `json:"min_rtt"`

	// MaxRTT is the maximum RTT in milliseconds
	MaxRTT float64 `json:"max_rtt"`

	// Jitter is the difference between max and min RTT
	Jitter float64 `json:"jitter"`

	// LossPercent is the packet loss percentage (0-100)
	LossPercent float64 `json:"loss_percent"`

	// Responded indicates if at least one probe got a response
	Responded bool `json:"responded"`
}

// ASNInfo contains Autonomous System Number information.
type ASNInfo struct {
	// Number is the AS number
	Number int `json:"number"`

	// Org is the organization name
	Org string `json:"org"`

	// Country is the country code (optional)
	Country string `json:"country,omitempty"`
}

// GeoInfo contains geographic location information.
type GeoInfo struct {
	// Country is the full country name
	Country string `json:"country"`

	// CountryCode is the ISO country code (e.g., "US", "TR")
	CountryCode string `json:"country_code"`

	// City is the city name (if available)
	City string `json:"city,omitempty"`

	// Latitude is the geographic latitude
	Latitude float64 `json:"latitude,omitempty"`

	// Longitude is the geographic longitude
	Longitude float64 `json:"longitude,omitempty"`
}

// TraceResult contains the complete result of a trace operation.
type TraceResult struct {
	// Target is the original target (hostname or IP)
	Target string `json:"target"`

	// ResolvedIP is the resolved IP address of the target
	ResolvedIP net.IP `json:"resolved_ip"`

	// Timestamp is when the trace was performed
	Timestamp time.Time `json:"timestamp"`

	// ProbeMethod is the probe method used (icmp, udp, tcp)
	ProbeMethod string `json:"probe_method"`

	// Hops contains all the hops in the trace
	Hops []Hop `json:"hops"`

	// Completed indicates if the trace reached the destination
	Completed bool `json:"completed"`

	// Summary contains aggregate statistics
	Summary Summary `json:"summary"`
}

// Summary contains aggregate statistics for a trace.
type Summary struct {
	// TotalHops is the number of hops in the trace
	TotalHops int `json:"total_hops"`

	// TotalTimeMs is the total trace time in milliseconds
	TotalTimeMs float64 `json:"total_time_ms"`

	// PacketLossPercent is the average packet loss across all hops
	PacketLossPercent float64 `json:"packet_loss_percent"`
}

// IsDestination checks if this hop is the final destination.
func (h *Hop) IsDestination(dest net.IP) bool {
	if h.IP == nil {
		return false
	}
	return h.IP.Equal(dest)
}
