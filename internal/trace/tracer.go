// Package trace provides traceroute functionality.
package trace

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/KilimcininKorOglu/poros/internal/enrich"
	"github.com/KilimcininKorOglu/poros/internal/probe"
)

// Tracer performs network path tracing operations.
type Tracer struct {
	config   *Config
	prober   probe.Prober
	enricher *enrich.Enricher
}

// New creates a new Tracer with the given configuration.
func New(config *Config) (*Tracer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// Create the appropriate prober based on configuration
	var prober probe.Prober
	var err error

	switch config.ProbeMethod {
	case ProbeICMP:
		prober, err = probe.NewICMPProber(probe.ICMPProberConfig{
			Timeout: config.Timeout,
			IPv6:    config.IPv6,
		})
	case ProbeUDP:
		prober, err = probe.NewUDPProber(probe.UDPProberConfig{
			Timeout:  config.Timeout,
			BasePort: config.DestPort,
			IPv6:     config.IPv6,
		})
	case ProbeTCP:
		prober, err = probe.NewTCPProber(probe.TCPProberConfig{
			Timeout: config.Timeout,
			Port:    config.DestPort,
			IPv6:    config.IPv6,
		})
	case ProbeParis:
		// Paris traceroute - determine underlying method
		var method probe.Method
		if config.Paris {
			method = probe.MethodUDP // Default Paris uses UDP
		}
		prober, err = probe.NewParisProber(probe.ParisProberConfig{
			Timeout: config.Timeout,
			Method:  method,
			Port:    config.DestPort,
			IPv6:    config.IPv6,
		})
	default:
		return nil, fmt.Errorf("unknown probe method: %v", config.ProbeMethod)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create prober: %w", err)
	}

	// Create enricher if enabled
	var enricher *enrich.Enricher
	if config.EnableEnrichment {
		enricher = enrich.NewEnricher(enrich.EnricherConfig{
			EnableRDNS:  config.EnableRDNS,
			EnableASN:   config.EnableASN,
			EnableGeoIP: config.EnableGeoIP,
		})
	}

	return &Tracer{
		config:   config,
		prober:   prober,
		enricher: enricher,
	}, nil
}

// Trace performs a traceroute to the specified target.
func (t *Tracer) Trace(ctx context.Context, target string) (*TraceResult, error) {
	// Resolve target to IP
	dest, err := t.resolveTarget(ctx, target)
	if err != nil {
		return nil, err
	}

	// Perform the trace
	// Note: ICMP concurrent mode has issues with shared socket on Windows,
	// so we use sequential mode for ICMP by default unless explicitly requested
	var hops []Hop
	useConcurrent := !t.config.Sequential
	if useConcurrent && t.config.ProbeMethod == ProbeICMP {
		// ICMP concurrent mode is problematic on Windows due to shared socket
		// responses getting mixed up between goroutines
		useConcurrent = false
	}
	
	if useConcurrent {
		hops, err = t.traceConcurrent(ctx, dest)
	} else {
		hops, err = t.traceSequential(ctx, dest)
	}

	if err != nil {
		return nil, err
	}

	// Enrich hops with rDNS, ASN, GeoIP
	if t.enricher != nil {
		// Collect IPs from hops
		ips := make([]net.IP, 0, len(hops))
		for _, hop := range hops {
			if hop.IP != nil {
				ips = append(ips, hop.IP)
			}
		}

		// Get enrichment results
		enrichResults := t.enricher.EnrichIPs(ctx, ips)

		// Apply results to hops
		for i := range hops {
			if hops[i].IP != nil {
				if result := enrichResults[hops[i].IP.String()]; result != nil {
					hops[i].Hostname = result.Hostname
					if result.ASN != nil {
						hops[i].ASN = &ASNInfo{
							Number:  result.ASN.Number,
							Org:     result.ASN.Org,
							Country: result.ASN.Country,
						}
					}
					if result.Geo != nil {
						hops[i].Geo = &GeoInfo{
							Country:     result.Geo.Country,
							CountryCode: result.Geo.CountryCode,
							City:        result.Geo.City,
							Latitude:    result.Geo.Latitude,
							Longitude:   result.Geo.Longitude,
						}
					}
				}
			}
		}
	}

	// Build and return the result
	return t.buildResult(target, dest, hops), nil
}

// Close releases resources held by the tracer.
func (t *Tracer) Close() error {
	var errs []error

	if t.prober != nil {
		if err := t.prober.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if t.enricher != nil {
		if err := t.enricher.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// resolveTarget resolves a hostname or IP string to a net.IP.
func (t *Tracer) resolveTarget(ctx context.Context, target string) (net.IP, error) {
	// Check if target is already an IP address
	if ip := net.ParseIP(target); ip != nil {
		// Apply IPv4/IPv6 preference
		if t.config.IPv4 && ip.To4() == nil {
			return nil, fmt.Errorf("%s is an IPv6 address but IPv4 was requested", target)
		}
		if t.config.IPv6 && ip.To4() != nil {
			return nil, fmt.Errorf("%s is an IPv4 address but IPv6 was requested", target)
		}
		return ip, nil
	}

	// Resolve hostname
	var network string
	switch {
	case t.config.IPv6:
		network = "ip6"
	case t.config.IPv4:
		network = "ip4"
	default:
		network = "ip" // Any
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, network, target)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", target, err)
	}

	if len(ips) == 0 {
		return nil, fmt.Errorf("no IP addresses found for %s", target)
	}

	// Prefer IPv4 unless IPv6 is explicitly requested
	if !t.config.IPv6 {
		for _, ip := range ips {
			if ip.To4() != nil {
				return ip, nil
			}
		}
	}

	return ips[0], nil
}

// traceSequential performs a sequential traceroute.
func (t *Tracer) traceSequential(ctx context.Context, dest net.IP) ([]Hop, error) {
	hops := make([]Hop, 0, t.config.MaxHops)

	for ttl := t.config.FirstHop; ttl <= t.config.MaxHops; ttl++ {
		select {
		case <-ctx.Done():
			return hops, ctx.Err()
		default:
		}

		hop := t.probeHop(ctx, dest, ttl)
		hops = append(hops, hop)

		// Check if we've reached the destination
		if hop.Responded && hop.IP != nil && hop.IP.Equal(dest) {
			break
		}
	}

	return hops, nil
}

// probeHop sends multiple probes for a single hop and aggregates the results.
func (t *Tracer) probeHop(ctx context.Context, dest net.IP, ttl int) Hop {
	hop := Hop{
		Number: ttl,
		RTTs:   make([]float64, 0, t.config.ProbeCount),
	}

	var lastIP net.IP
	successCount := 0

	for i := 0; i < t.config.ProbeCount; i++ {
		select {
		case <-ctx.Done():
			break
		default:
		}

		result, err := t.prober.Probe(ctx, dest, ttl)
		if err != nil {
			// Timeout or error - record as -1
			hop.RTTs = append(hop.RTTs, -1)
			continue
		}

		// Record successful probe
		rtt := float64(result.RTT.Microseconds()) / 1000.0 // Convert to ms
		hop.RTTs = append(hop.RTTs, rtt)
		successCount++

		if result.ResponseIP != nil {
			lastIP = result.ResponseIP
		}
	}

	// Set hop IP if we got any response
	if lastIP != nil {
		hop.IP = lastIP
		hop.Responded = true
	}

	// Calculate statistics
	hop.AvgRTT, hop.MinRTT, hop.MaxRTT, hop.Jitter = calculateRTTStats(hop.RTTs)
	hop.LossPercent = calculateLossPercent(hop.RTTs)

	return hop
}

// buildResult creates a TraceResult from the collected hops.
func (t *Tracer) buildResult(target string, dest net.IP, hops []Hop) *TraceResult {
	result := &TraceResult{
		Target:      target,
		ResolvedIP:  dest,
		Timestamp:   time.Now(),
		ProbeMethod: t.prober.Name(),
		Hops:        hops,
		Completed:   false,
	}

	// Check if trace completed (reached destination)
	if len(hops) > 0 {
		lastHop := hops[len(hops)-1]
		if lastHop.IP != nil && lastHop.IP.Equal(dest) {
			result.Completed = true
		}
	}

	// Calculate summary statistics
	result.Summary = t.calculateSummary(hops)

	return result
}

// calculateSummary calculates aggregate statistics for the trace.
func (t *Tracer) calculateSummary(hops []Hop) Summary {
	summary := Summary{
		TotalHops: len(hops),
	}

	var totalRTT float64
	var totalLoss float64
	respondingHops := 0

	for _, hop := range hops {
		if hop.AvgRTT > 0 {
			totalRTT += hop.AvgRTT
			respondingHops++
		}
		totalLoss += hop.LossPercent
	}

	if len(hops) > 0 {
		summary.PacketLossPercent = totalLoss / float64(len(hops))
	}

	// Total time is the RTT to the last responding hop
	for i := len(hops) - 1; i >= 0; i-- {
		if hops[i].AvgRTT > 0 {
			summary.TotalTimeMs = hops[i].AvgRTT
			break
		}
	}

	return summary
}

// calculateRTTStats calculates RTT statistics from a slice of RTT values.
// Negative values are treated as timeouts and excluded from calculations.
func calculateRTTStats(rtts []float64) (avg, min, max, jitter float64) {
	var valid []float64
	for _, rtt := range rtts {
		if rtt >= 0 {
			valid = append(valid, rtt)
		}
	}

	if len(valid) == 0 {
		return 0, 0, 0, 0
	}

	min = valid[0]
	max = valid[0]
	sum := 0.0

	for _, rtt := range valid {
		sum += rtt
		if rtt < min {
			min = rtt
		}
		if rtt > max {
			max = rtt
		}
	}

	avg = sum / float64(len(valid))
	jitter = max - min

	return
}

// calculateLossPercent calculates packet loss percentage.
// Negative RTT values indicate timeouts.
func calculateLossPercent(rtts []float64) float64 {
	if len(rtts) == 0 {
		return 0
	}

	timeouts := 0
	for _, rtt := range rtts {
		if rtt < 0 {
			timeouts++
		}
	}

	return float64(timeouts) / float64(len(rtts)) * 100
}
