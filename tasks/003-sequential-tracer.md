# Feature 003: Sequential Tracer Implementation

**Feature ID:** F003
**Feature Name:** Sequential Tracer Implementation
**Priority:** P1 - CRITICAL
**Target Version:** v0.1.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Implement the classic sequential tracing mode where probes are sent one at a time with incrementing TTL values. This is the traditional traceroute approach: send TTL=1, wait for response, send TTL=2, wait for response, and so on until the destination is reached or max hops exceeded.

Sequential mode serves as both a reliable fallback and a debugging tool. While slower than concurrent mode, it produces predictable results and is easier to reason about when troubleshooting network issues.

## Goals
- Implement a complete sequential traceroute operation
- Send multiple probes per hop (configurable, default 3)
- Calculate RTT statistics (min, max, avg, jitter)
- Detect destination reached condition
- Handle timeouts gracefully
- Build Hop and TraceResult data structures

## Success Criteria
- [ ] All tasks completed (T014-T019)
- [ ] `poros google.com` produces valid trace output
- [ ] Multiple probes per hop work correctly
- [ ] Timeout handling is robust
- [ ] Statistics are accurately calculated
- [ ] Trace stops when destination reached

## Tasks

### T014: Implement Tracer Core Structure

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create the main Tracer struct that orchestrates trace operations. This includes configuration handling, prober management, and result collection.

#### Technical Details
```go
// internal/trace/tracer.go
type Tracer struct {
    config  *TracerConfig
    prober  probe.Prober
    enricher *enrich.Enricher  // nil if enrichment disabled
}

func NewTracer(config *TracerConfig) (*Tracer, error) {
    // Create appropriate prober based on config
    var prober probe.Prober
    switch config.ProbeMethod {
    case probe.ProbeICMP:
        prober, err = probe.NewICMPProber(probe.ICMPProberConfig{
            Timeout: config.Timeout,
            IPv6:    config.IPv6,
        })
    // future: UDP, TCP, Paris
    }
    
    return &Tracer{
        config: config,
        prober: prober,
    }, nil
}

func (t *Tracer) Trace(ctx context.Context, target string) (*TraceResult, error) {
    // 1. Resolve target to IP
    // 2. Run trace (sequential or concurrent based on config)
    // 3. Enrich results if enabled
    // 4. Return TraceResult
}

func (t *Tracer) Close() error {
    return t.prober.Close()
}
```

#### Files to Touch
- `internal/trace/tracer.go` (new)
- `internal/trace/tracer_test.go` (new)

#### Dependencies
- T002: Core data structures
- T003: Prober interface
- T011: ICMP probe implementation

#### Success Criteria
- [ ] Tracer can be created with config
- [ ] Prober is properly initialized
- [ ] Close releases resources
- [ ] Context cancellation works

---

### T015: Implement DNS Resolution

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Implement target hostname resolution with support for IPv4/IPv6 preference. Handle both hostname and direct IP address inputs.

#### Technical Details
```go
// internal/trace/resolve.go
func (t *Tracer) resolveTarget(ctx context.Context, target string) (net.IP, error) {
    // Check if target is already an IP
    if ip := net.ParseIP(target); ip != nil {
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
        network = "ip" // any
    }
    
    ips, err := net.DefaultResolver.LookupIP(ctx, network, target)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve %s: %w", target, err)
    }
    
    if len(ips) == 0 {
        return nil, fmt.Errorf("no IP addresses found for %s", target)
    }
    
    // Prefer IPv6 if not forced to IPv4
    if !t.config.IPv4 {
        for _, ip := range ips {
            if ip.To4() == nil {
                return ip, nil
            }
        }
    }
    
    return ips[0], nil
}
```

#### Files to Touch
- `internal/trace/resolve.go` (new)
- `internal/trace/resolve_test.go` (new)

#### Dependencies
- T014: Tracer core structure

#### Success Criteria
- [ ] Resolves hostnames correctly
- [ ] Handles direct IP input
- [ ] Respects IPv4/IPv6 preference
- [ ] Handles resolution failures gracefully

---

### T016: Implement Sequential Trace Logic

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Implement the sequential tracing algorithm that sends probes with incrementing TTL values, one hop at a time. This is the core traceroute logic.

#### Technical Details
```go
// internal/trace/sequential.go
func (t *Tracer) traceSequential(ctx context.Context, dest net.IP) ([]Hop, error) {
    hops := make([]Hop, 0, t.config.MaxHops)
    
    for ttl := t.config.FirstHop; ttl <= t.config.MaxHops; ttl++ {
        select {
        case <-ctx.Done():
            return hops, ctx.Err()
        default:
        }
        
        hop := Hop{Number: ttl}
        rtts := make([]float64, 0, t.config.ProbeCount)
        var lastResult *probe.ProbeResult
        
        // Send multiple probes for this TTL
        for i := 0; i < t.config.ProbeCount; i++ {
            result, err := t.prober.Probe(ctx, dest, ttl)
            if err != nil {
                if errors.Is(err, probe.ErrTimeout) {
                    rtts = append(rtts, -1) // timeout marker
                    continue
                }
                return hops, err
            }
            
            rtts = append(rtts, float64(result.RTT.Microseconds())/1000.0)
            lastResult = result
            hop.IP = result.ResponseIP
            hop.Responded = true
        }
        
        // Calculate statistics
        hop.RTTs = rtts
        hop.AvgRTT, hop.MinRTT, hop.MaxRTT, hop.Jitter = calculateStats(rtts)
        hop.LossPercent = calculateLoss(rtts)
        
        hops = append(hops, hop)
        
        // Check if destination reached
        if lastResult != nil && lastResult.Reached {
            break
        }
    }
    
    return hops, nil
}

func calculateStats(rtts []float64) (avg, min, max, jitter float64) {
    valid := make([]float64, 0, len(rtts))
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
```

#### Files to Touch
- `internal/trace/sequential.go` (new)
- `internal/trace/sequential_test.go` (new)
- `internal/trace/stats.go` (new)

#### Dependencies
- T014: Tracer core
- T015: DNS resolution
- T011: ICMP probe

#### Success Criteria
- [ ] Traces complete path to destination
- [ ] Handles timeouts at any hop
- [ ] Calculates accurate statistics
- [ ] Respects MaxHops limit
- [ ] Context cancellation works mid-trace

---

### T017: Implement Hop Statistics Calculation

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create robust statistics calculation for RTT measurements including handling of missing data points (timeouts) and edge cases.

#### Technical Details
```go
// internal/trace/stats.go
type HopStats struct {
    Avg     float64
    Min     float64
    Max     float64
    Jitter  float64
    StdDev  float64
    Loss    float64
    Count   int
    Valid   int
}

func CalculateHopStats(rtts []float64) HopStats {
    stats := HopStats{
        Count: len(rtts),
    }
    
    valid := make([]float64, 0, len(rtts))
    for _, rtt := range rtts {
        if rtt >= 0 {
            valid = append(valid, rtt)
        }
    }
    
    stats.Valid = len(valid)
    stats.Loss = float64(stats.Count-stats.Valid) / float64(stats.Count) * 100.0
    
    if len(valid) == 0 {
        return stats
    }
    
    // Sort for percentile calculations
    sort.Float64s(valid)
    
    stats.Min = valid[0]
    stats.Max = valid[len(valid)-1]
    
    // Calculate mean
    sum := 0.0
    for _, v := range valid {
        sum += v
    }
    stats.Avg = sum / float64(len(valid))
    
    // Calculate standard deviation
    if len(valid) > 1 {
        sumSq := 0.0
        for _, v := range valid {
            diff := v - stats.Avg
            sumSq += diff * diff
        }
        stats.StdDev = math.Sqrt(sumSq / float64(len(valid)))
    }
    
    stats.Jitter = stats.Max - stats.Min
    
    return stats
}
```

#### Files to Touch
- `internal/trace/stats.go` (update/new)
- `internal/trace/stats_test.go` (new)

#### Dependencies
- T016: Sequential trace needs stats

#### Success Criteria
- [ ] Handles empty RTT array
- [ ] Handles all timeouts
- [ ] Accurate calculations
- [ ] Unit tests cover edge cases

---

### T018: Build TraceResult Structure

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Assemble the complete TraceResult structure from collected hops data, including summary statistics for the entire trace.

#### Technical Details
```go
// internal/trace/result.go
func (t *Tracer) buildResult(target string, dest net.IP, hops []Hop) *TraceResult {
    result := &TraceResult{
        Target:      target,
        ResolvedIP:  dest,
        Timestamp:   time.Now(),
        ProbeMethod: t.prober.Name(),
        Hops:        hops,
        Completed:   false,
    }
    
    // Check if trace completed
    if len(hops) > 0 {
        lastHop := hops[len(hops)-1]
        result.Completed = lastHop.IP.Equal(dest)
    }
    
    // Calculate summary
    result.Summary = t.calculateSummary(hops, dest)
    
    return result
}

func (t *Tracer) calculateSummary(hops []Hop, dest net.IP) Summary {
    summary := Summary{
        TotalHops: len(hops),
    }
    
    totalLoss := 0.0
    for _, hop := range hops {
        totalLoss += hop.LossPercent
        if hop.AvgRTT > 0 {
            summary.TotalTimeMs += hop.AvgRTT
        }
    }
    
    if len(hops) > 0 {
        summary.PacketLossPercent = totalLoss / float64(len(hops))
    }
    
    return summary
}

type Summary struct {
    TotalHops         int     `json:"total_hops"`
    TotalTimeMs       float64 `json:"total_time_ms"`
    PacketLossPercent float64 `json:"packet_loss_percent"`
}
```

#### Files to Touch
- `internal/trace/result.go` (new)
- `internal/trace/types.go` (update - add Summary)

#### Dependencies
- T016: Sequential trace
- T017: Statistics

#### Success Criteria
- [ ] Complete TraceResult populated
- [ ] Summary statistics accurate
- [ ] Completed flag correct
- [ ] JSON serialization works

---

### T019: Add Sequential Tracer Integration Test

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create integration tests that verify the complete sequential trace flow against real network targets.

#### Technical Details
```go
// internal/trace/tracer_integration_test.go
//go:build integration

func TestSequentialTrace_Localhost(t *testing.T) {
    if os.Getuid() != 0 {
        t.Skip("requires root")
    }
    
    tracer, err := NewTracer(&TracerConfig{
        ProbeMethod: probe.ProbeICMP,
        MaxHops:     5,
        ProbeCount:  1,
        Timeout:     time.Second,
        Concurrent:  false,
    })
    require.NoError(t, err)
    defer tracer.Close()
    
    result, err := tracer.Trace(context.Background(), "127.0.0.1")
    require.NoError(t, err)
    
    assert.True(t, result.Completed)
    assert.Equal(t, 1, len(result.Hops))
}

func TestSequentialTrace_External(t *testing.T) {
    // Test against 8.8.8.8 or similar
}
```

#### Files to Touch
- `internal/trace/tracer_integration_test.go` (new)
- `scripts/test-trace.sh` (new)

#### Dependencies
- T018: Complete TraceResult building

#### Success Criteria
- [ ] Localhost trace works
- [ ] External target trace works
- [ ] Handles unreachable targets
- [ ] Tests are skipped appropriately without root

---

## Performance Targets
- 30-hop sequential trace: < 90 seconds (worst case with timeouts)
- 30-hop sequential trace: < 15 seconds (typical, responsive hops)
- Memory usage during trace: < 10MB

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Slow traces on lossy networks | Medium | Medium | Good timeout handling, progress indication |
| DNS resolution blocking | Low | Medium | Use context with timeout |
| Memory growth with many hops | Low | Low | Fixed buffer sizes |

## Notes
- Sequential mode is the foundation - concurrent mode will build on this
- Keep the code modular to allow concurrent mode to reuse components
- Consider adding a progress callback for long traces
- The sequential mode is also useful for debugging probe issues
