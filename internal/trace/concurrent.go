package trace

import (
	"context"
	"net"
	"sort"
	"sync"
)

// hopResult holds the result of probing a single hop.
type hopResult struct {
	ttl int
	hop Hop
}

// traceConcurrent performs a concurrent traceroute.
// It launches multiple goroutines to probe different hops simultaneously,
// which significantly speeds up the trace for paths with many hops.
func (t *Tracer) traceConcurrent(ctx context.Context, dest net.IP) ([]Hop, error) {
	// Create context with cancellation for early termination
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Calculate concurrency limit
	concurrency := t.config.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 30
	}
	if concurrency > t.config.MaxHops {
		concurrency = t.config.MaxHops
	}

	// Create channels
	jobs := make(chan int, t.config.MaxHops)
	results := make(chan hopResult, t.config.MaxHops)

	// Start worker pool
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			t.worker(ctx, dest, jobs, results)
		}()
	}

	// Submit jobs for all TTLs
	go func() {
		for ttl := t.config.FirstHop; ttl <= t.config.MaxHops; ttl++ {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- ttl:
			}
		}
		close(jobs)
	}()

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	hopMap := make(map[int]Hop)
	destinationReached := false
	destinationTTL := t.config.MaxHops + 1

	for result := range results {
		hopMap[result.ttl] = result.hop

		// Check if we reached the destination
		if result.hop.Responded && result.hop.IP != nil && result.hop.IP.Equal(dest) {
			destinationReached = true
			if result.ttl < destinationTTL {
				destinationTTL = result.ttl
			}
		}
	}

	// Build ordered hop list
	hops := t.buildHopList(hopMap, destinationReached, destinationTTL)

	return hops, nil
}

// worker processes probe jobs from the jobs channel.
func (t *Tracer) worker(ctx context.Context, dest net.IP, jobs <-chan int, results chan<- hopResult) {
	for ttl := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		hop := t.probeHop(ctx, dest, ttl)
		results <- hopResult{ttl: ttl, hop: hop}
	}
}

// buildHopList builds an ordered list of hops from the result map.
func (t *Tracer) buildHopList(hopMap map[int]Hop, destinationReached bool, destinationTTL int) []Hop {
	// Get sorted TTL values
	ttls := make([]int, 0, len(hopMap))
	for ttl := range hopMap {
		ttls = append(ttls, ttl)
	}
	sort.Ints(ttls)

	// Build hop list, stopping at destination if reached
	hops := make([]Hop, 0, len(ttls))
	for _, ttl := range ttls {
		// Skip hops beyond the destination
		if destinationReached && ttl > destinationTTL {
			continue
		}
		hops = append(hops, hopMap[ttl])
	}

	return hops
}

// traceConcurrentAdaptive uses adaptive concurrency based on response times.
// It starts with lower concurrency and increases it if responses are fast,
// or decreases it if responses are slow or timing out.
func (t *Tracer) traceConcurrentAdaptive(ctx context.Context, dest net.IP) ([]Hop, error) {
	// For now, use regular concurrent mode
	// Adaptive logic can be added later based on RTT feedback
	return t.traceConcurrent(ctx, dest)
}
