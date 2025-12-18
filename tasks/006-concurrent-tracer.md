# Feature 006: Concurrent Tracer Implementation

**Feature ID:** F006
**Feature Name:** Concurrent Tracer Implementation
**Priority:** P2 - HIGH
**Target Version:** v0.2.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Implement concurrent tracing mode that sends probes to all TTL levels simultaneously, dramatically reducing trace time. Instead of sequential TTL=1, wait, TTL=2, wait..., concurrent mode sends all probes in parallel and collects responses as they arrive.

This is Poros's default mode and its key differentiator from traditional traceroute. A typical 30-hop trace that takes 30+ seconds in sequential mode can complete in under 5 seconds with concurrent probing.

## Goals
- Send probes to all TTL levels concurrently using goroutines
- Manage goroutine pool with configurable concurrency
- Collect and order responses correctly
- Implement rate limiting to prevent network flooding
- Handle context cancellation mid-trace

## Success Criteria
- [ ] All tasks completed (T032-T038)
- [ ] 30-hop trace completes in < 5 seconds
- [ ] Results match sequential mode output
- [ ] Memory usage stays under 50MB
- [ ] Graceful handling of high packet loss

## Tasks

### T032: Implement Goroutine Pool Manager

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create a worker pool that manages concurrent probe goroutines with configurable pool size and proper shutdown handling.

#### Technical Details
```go
// internal/trace/pool.go
type ProbePool struct {
    workers   int
    taskCh    chan ProbeTask
    resultCh  chan ProbeTaskResult
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

type ProbeTask struct {
    TTL      int
    Dest     net.IP
    ProbeNum int  // Which probe attempt (1, 2, 3, etc.)
}

type ProbeTaskResult struct {
    TTL      int
    ProbeNum int
    Result   *probe.ProbeResult
    Err      error
}

func NewProbePool(ctx context.Context, workers int) *ProbePool {
    ctx, cancel := context.WithCancel(ctx)
    
    pool := &ProbePool{
        workers:  workers,
        taskCh:   make(chan ProbeTask, workers*2),
        resultCh: make(chan ProbeTaskResult, workers*2),
        ctx:      ctx,
        cancel:   cancel,
    }
    
    return pool
}

func (p *ProbePool) Start(prober probe.Prober) {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(prober)
    }
}

func (p *ProbePool) worker(prober probe.Prober) {
    defer p.wg.Done()
    
    for {
        select {
        case <-p.ctx.Done():
            return
        case task, ok := <-p.taskCh:
            if !ok {
                return
            }
            
            result, err := prober.Probe(p.ctx, task.Dest, task.TTL)
            p.resultCh <- ProbeTaskResult{
                TTL:      task.TTL,
                ProbeNum: task.ProbeNum,
                Result:   result,
                Err:      err,
            }
        }
    }
}

func (p *ProbePool) Submit(task ProbeTask) {
    select {
    case <-p.ctx.Done():
        return
    case p.taskCh <- task:
    }
}

func (p *ProbePool) Results() <-chan ProbeTaskResult {
    return p.resultCh
}

func (p *ProbePool) Stop() {
    close(p.taskCh)
    p.wg.Wait()
    close(p.resultCh)
}
```

#### Files to Touch
- `internal/trace/pool.go` (new)
- `internal/trace/pool_test.go` (new)

#### Dependencies
- T003: Prober interface
- T011: ICMP probe implementation

#### Success Criteria
- [ ] Pool starts and stops cleanly
- [ ] Workers process tasks correctly
- [ ] Context cancellation stops all workers
- [ ] No goroutine leaks

---

### T033: Implement Result Collector

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create a collector that gathers probe results from concurrent workers, orders them by TTL, and builds the Hop structures with aggregated statistics.

#### Technical Details
```go
// internal/trace/collector.go
type ResultCollector struct {
    maxHops    int
    probeCount int
    results    map[int][]ProbeTaskResult  // TTL -> results
    mu         sync.Mutex
    complete   chan struct{}
    expected   int
    received   int
}

func NewResultCollector(maxHops, probeCount int) *ResultCollector {
    return &ResultCollector{
        maxHops:    maxHops,
        probeCount: probeCount,
        results:    make(map[int][]ProbeTaskResult),
        complete:   make(chan struct{}),
        expected:   maxHops * probeCount,
    }
}

func (c *ResultCollector) Collect(resultCh <-chan ProbeTaskResult, 
    destReached func(result *probe.ProbeResult) bool) {
    
    var reachedTTL int = -1
    
    for result := range resultCh {
        c.mu.Lock()
        c.results[result.TTL] = append(c.results[result.TTL], result)
        c.received++
        
        // Check if destination was reached
        if result.Result != nil && result.Result.Reached && reachedTTL < 0 {
            reachedTTL = result.TTL
        }
        
        // Check if we have all expected results or reached destination
        done := c.received >= c.expected
        if reachedTTL > 0 {
            // Count expected results up to reached TTL
            expectedUntilDest := reachedTTL * c.probeCount
            done = c.countResultsUntil(reachedTTL) >= expectedUntilDest
        }
        
        c.mu.Unlock()
        
        if done {
            close(c.complete)
            return
        }
    }
    
    close(c.complete)
}

func (c *ResultCollector) Wait(timeout time.Duration) {
    select {
    case <-c.complete:
    case <-time.After(timeout):
    }
}

func (c *ResultCollector) BuildHops() []Hop {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    hops := make([]Hop, 0, c.maxHops)
    
    for ttl := 1; ttl <= c.maxHops; ttl++ {
        results, ok := c.results[ttl]
        if !ok {
            // No response for this TTL - add timeout hop
            hops = append(hops, Hop{
                Number:    ttl,
                Responded: false,
            })
            continue
        }
        
        hop := c.buildHop(ttl, results)
        hops = append(hops, hop)
        
        // Stop if destination reached
        if hop.isDestination() {
            break
        }
    }
    
    return hops
}

func (c *ResultCollector) buildHop(ttl int, results []ProbeTaskResult) Hop {
    hop := Hop{
        Number: ttl,
        RTTs:   make([]float64, 0, len(results)),
    }
    
    for _, r := range results {
        if r.Err != nil || r.Result == nil {
            hop.RTTs = append(hop.RTTs, -1) // timeout
            continue
        }
        
        if hop.IP == nil {
            hop.IP = r.Result.ResponseIP
        }
        
        rtt := float64(r.Result.RTT.Microseconds()) / 1000.0
        hop.RTTs = append(hop.RTTs, rtt)
        hop.Responded = true
    }
    
    // Calculate statistics
    stats := CalculateHopStats(hop.RTTs)
    hop.AvgRTT = stats.Avg
    hop.MinRTT = stats.Min
    hop.MaxRTT = stats.Max
    hop.Jitter = stats.Jitter
    hop.LossPercent = stats.Loss
    
    return hop
}
```

#### Files to Touch
- `internal/trace/collector.go` (new)
- `internal/trace/collector_test.go` (new)

#### Dependencies
- T032: Goroutine pool
- T017: Statistics calculation

#### Success Criteria
- [ ] Results correctly aggregated by TTL
- [ ] Handles mixed responses and timeouts
- [ ] Early termination when destination reached
- [ ] Statistics calculated correctly

---

### T034: Implement Concurrent Trace Logic

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description
Implement the main concurrent tracing algorithm that coordinates the pool, collector, and result assembly.

#### Technical Details
```go
// internal/trace/concurrent.go
func (t *Tracer) traceConcurrent(ctx context.Context, dest net.IP) ([]Hop, error) {
    // Create worker pool
    workers := min(t.config.MaxHops, t.config.MaxConcurrency)
    pool := NewProbePool(ctx, workers)
    pool.Start(t.prober)
    defer pool.Stop()
    
    // Create result collector
    collector := NewResultCollector(t.config.MaxHops, t.config.ProbeCount)
    
    // Start collecting results in background
    go collector.Collect(pool.Results(), func(r *probe.ProbeResult) bool {
        return r != nil && r.Reached
    })
    
    // Submit all probe tasks
    for ttl := t.config.FirstHop; ttl <= t.config.MaxHops; ttl++ {
        for probeNum := 0; probeNum < t.config.ProbeCount; probeNum++ {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            default:
                pool.Submit(ProbeTask{
                    TTL:      ttl,
                    Dest:     dest,
                    ProbeNum: probeNum,
                })
            }
            
            // Rate limiting
            if t.config.RateLimit > 0 {
                time.Sleep(t.config.RateLimitDelay)
            }
        }
    }
    
    // Wait for all results with timeout
    maxWait := time.Duration(t.config.MaxHops) * t.config.Timeout
    collector.Wait(maxWait)
    
    // Build and return hops
    return collector.BuildHops(), nil
}

// Update tracer to use concurrent by default
func (t *Tracer) Trace(ctx context.Context, target string) (*TraceResult, error) {
    dest, err := t.resolveTarget(ctx, target)
    if err != nil {
        return nil, err
    }
    
    var hops []Hop
    if t.config.Sequential {
        hops, err = t.traceSequential(ctx, dest)
    } else {
        hops, err = t.traceConcurrent(ctx, dest)
    }
    
    if err != nil {
        return nil, err
    }
    
    result := t.buildResult(target, dest, hops)
    
    // Enrich if enabled
    if t.enricher != nil {
        t.enricher.EnrichHops(ctx, result.Hops)
    }
    
    return result, nil
}
```

#### Files to Touch
- `internal/trace/concurrent.go` (new)
- `internal/trace/concurrent_test.go` (new)
- `internal/trace/tracer.go` (update)
- `internal/trace/config.go` (update - add MaxConcurrency, RateLimit)

#### Dependencies
- T032: Goroutine pool
- T033: Result collector
- T016: Sequential trace (for comparison)

#### Success Criteria
- [ ] Concurrent trace is significantly faster
- [ ] Results match sequential mode
- [ ] Rate limiting works
- [ ] Context cancellation stops trace

---

### T035: Implement Rate Limiting

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Add rate limiting to prevent overwhelming the network or triggering rate-limit protections on routers.

#### Technical Details
```go
// internal/trace/ratelimit.go
type RateLimiter struct {
    ticker     *time.Ticker
    tokens     chan struct{}
    maxTokens  int
    refillRate time.Duration
}

func NewRateLimiter(packetsPerSecond int) *RateLimiter {
    if packetsPerSecond <= 0 {
        return nil // No rate limiting
    }
    
    refillRate := time.Second / time.Duration(packetsPerSecond)
    
    rl := &RateLimiter{
        ticker:     time.NewTicker(refillRate),
        tokens:     make(chan struct{}, packetsPerSecond),
        maxTokens:  packetsPerSecond,
        refillRate: refillRate,
    }
    
    // Fill initial tokens
    for i := 0; i < packetsPerSecond; i++ {
        rl.tokens <- struct{}{}
    }
    
    // Start refill goroutine
    go rl.refill()
    
    return rl
}

func (rl *RateLimiter) refill() {
    for range rl.ticker.C {
        select {
        case rl.tokens <- struct{}{}:
        default:
            // Token bucket full
        }
    }
}

func (rl *RateLimiter) Acquire(ctx context.Context) error {
    if rl == nil {
        return nil
    }
    
    select {
    case <-ctx.Done():
        return ctx.Err()
    case <-rl.tokens:
        return nil
    }
}

func (rl *RateLimiter) Stop() {
    if rl != nil {
        rl.ticker.Stop()
    }
}

// TracerConfig addition
type TracerConfig struct {
    // ...existing fields...
    PacketsPerSecond int // 0 = no limit, default 100
}
```

#### Files to Touch
- `internal/trace/ratelimit.go` (new)
- `internal/trace/ratelimit_test.go` (new)
- `internal/trace/config.go` (update)
- `internal/trace/concurrent.go` (integrate)

#### Dependencies
- T034: Concurrent trace logic

#### Success Criteria
- [ ] Rate limiting constrains packet rate
- [ ] Can be disabled (0 = unlimited)
- [ ] Integrates smoothly with concurrent trace
- [ ] Token bucket algorithm works correctly

---

### T036: Implement Early Termination

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Optimize concurrent tracing to stop sending probes once the destination is reached, avoiding wasted packets.

#### Technical Details
```go
// internal/trace/concurrent.go (update)
func (t *Tracer) traceConcurrent(ctx context.Context, dest net.IP) ([]Hop, error) {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    
    // Create channels
    taskCh := make(chan ProbeTask, t.config.MaxConcurrency)
    resultCh := make(chan ProbeTaskResult, t.config.MaxConcurrency)
    reachedCh := make(chan int, 1) // Signals when destination reached
    
    // Start workers
    var wg sync.WaitGroup
    for i := 0; i < t.config.MaxConcurrency; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for task := range taskCh {
                result, err := t.prober.Probe(ctx, task.Dest, task.TTL)
                select {
                case resultCh <- ProbeTaskResult{
                    TTL:      task.TTL,
                    ProbeNum: task.ProbeNum,
                    Result:   result,
                    Err:      err,
                }:
                    // Check if destination reached
                    if result != nil && result.Reached {
                        select {
                        case reachedCh <- task.TTL:
                        default:
                        }
                    }
                case <-ctx.Done():
                    return
                }
            }
        }()
    }
    
    // Producer goroutine with early termination
    go func() {
        defer close(taskCh)
        
        var reachedTTL int = -1
        
        for ttl := t.config.FirstHop; ttl <= t.config.MaxHops; ttl++ {
            // Check if we should stop
            select {
            case reached := <-reachedCh:
                reachedTTL = reached
            default:
            }
            
            if reachedTTL > 0 && ttl > reachedTTL {
                break // Stop submitting tasks beyond reached TTL
            }
            
            for probeNum := 0; probeNum < t.config.ProbeCount; probeNum++ {
                select {
                case <-ctx.Done():
                    return
                case taskCh <- ProbeTask{TTL: ttl, Dest: dest, ProbeNum: probeNum}:
                }
            }
        }
    }()
    
    // Collect results
    // ...
}
```

#### Files to Touch
- `internal/trace/concurrent.go` (update)

#### Dependencies
- T034: Concurrent trace logic

#### Success Criteria
- [ ] Stops sending probes after destination reached
- [ ] Doesn't miss any valid responses
- [ ] Handles race conditions correctly

---

### T037: Add Concurrent Mode CLI Flag

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Add CLI flags to control concurrent mode, including enabling sequential mode as fallback and setting concurrency limits.

#### Technical Details
```go
// cmd/poros/root.go
func init() {
    rootCmd.Flags().Bool("sequential", false, 
        "Use sequential tracing instead of concurrent")
    rootCmd.Flags().Int("max-concurrency", 30, 
        "Maximum concurrent probes (default: 30)")
    rootCmd.Flags().Int("rate-limit", 0, 
        "Max packets per second (0 = unlimited)")
}

func buildTracerConfig(cmd *cobra.Command, target string) (*trace.TracerConfig, error) {
    config := &trace.TracerConfig{
        // ... existing fields ...
        Sequential:     getBool(cmd, "sequential"),
        MaxConcurrency: getInt(cmd, "max-concurrency"),
        PacketsPerSecond: getInt(cmd, "rate-limit"),
    }
    
    return config, nil
}
```

#### Files to Touch
- `cmd/poros/root.go` (update)
- `cmd/poros/flags.go` (update)

#### Dependencies
- T034: Concurrent trace implementation

#### Success Criteria
- [ ] `--sequential` enables sequential mode
- [ ] `--max-concurrency` limits workers
- [ ] `--rate-limit` controls packet rate
- [ ] Help text is clear

---

### T038: Add Concurrent Tracer Performance Test

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create benchmarks comparing concurrent vs sequential tracing and verifying performance targets.

#### Technical Details
```go
// internal/trace/concurrent_bench_test.go
func BenchmarkSequentialTrace(b *testing.B) {
    // Mock prober with simulated latency
    prober := &MockProber{
        LatencyPerHop: 10 * time.Millisecond,
        MaxHops:       15,
    }
    
    tracer := &Tracer{
        config: &TracerConfig{
            Sequential: true,
            MaxHops:    30,
            ProbeCount: 3,
        },
        prober: prober,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        tracer.Trace(context.Background(), "test")
    }
}

func BenchmarkConcurrentTrace(b *testing.B) {
    prober := &MockProber{
        LatencyPerHop: 10 * time.Millisecond,
        MaxHops:       15,
    }
    
    tracer := &Tracer{
        config: &TracerConfig{
            Sequential:     false,
            MaxConcurrency: 30,
            MaxHops:        30,
            ProbeCount:     3,
        },
        prober: prober,
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        tracer.Trace(context.Background(), "test")
    }
}

// internal/trace/concurrent_integration_test.go
func TestConcurrentVsSequential(t *testing.T) {
    // Verify both modes produce equivalent results
    target := "8.8.8.8"
    
    seqResult, _ := traceSequential(target)
    concResult, _ := traceConcurrent(target)
    
    // Compare hop IPs (may differ slightly due to load balancing)
    assert.Equal(t, len(seqResult.Hops), len(concResult.Hops))
}
```

#### Files to Touch
- `internal/trace/concurrent_bench_test.go` (new)
- `internal/trace/concurrent_integration_test.go` (new)

#### Dependencies
- T034: Concurrent implementation complete

#### Success Criteria
- [ ] Concurrent mode is 5x+ faster than sequential
- [ ] Memory usage stays under 50MB
- [ ] No goroutine leaks detected
- [ ] Results are consistent

---

## Performance Targets
- 30-hop concurrent trace: < 5 seconds
- Memory usage: < 50MB
- Goroutine count: < 100 during trace
- CPU usage: < 25% single core

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Goroutine leaks | Medium | High | Careful context handling, testing |
| Response ordering issues | Medium | Medium | Thorough collector testing |
| Rate limiting too aggressive | Low | Medium | Sensible defaults, user override |
| High CPU with many probes | Low | Low | Efficient channel usage |

## Notes
- Concurrent mode is the default and key differentiator of Poros
- Consider adding adaptive mode that starts sequential then switches to concurrent
- Rate limiting is important for being a "good network citizen"
- The collector design allows for live updates (useful for TUI later)
