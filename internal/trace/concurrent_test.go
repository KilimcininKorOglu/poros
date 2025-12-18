package trace

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestTraceConcurrent_Localhost(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultConfig()
	config.MaxHops = 5
	config.ProbeCount = 1
	config.Timeout = 2 * time.Second
	config.Sequential = false // Concurrent mode

	tracer, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer tracer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := tracer.Trace(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	if result.Target != "127.0.0.1" {
		t.Errorf("Target = %q, want %q", result.Target, "127.0.0.1")
	}

	if !result.Completed {
		t.Error("Trace to localhost should complete")
	}

	if len(result.Hops) == 0 {
		t.Error("Trace should have at least one hop")
	}

	// Verify hops are in order
	for i, hop := range result.Hops {
		expectedNum := i + 1
		if hop.Number != expectedNum {
			t.Errorf("Hop[%d].Number = %d, want %d", i, hop.Number, expectedNum)
		}
	}
}

func TestBuildHopList(t *testing.T) {
	config := DefaultConfig()
	tracer := &Tracer{config: config}

	tests := []struct {
		name           string
		hopMap         map[int]Hop
		destReached    bool
		destTTL        int
		expectedLen    int
		expectedLast   int
	}{
		{
			name: "all hops, no destination",
			hopMap: map[int]Hop{
				1: {Number: 1, IP: net.ParseIP("10.0.0.1"), Responded: true},
				2: {Number: 2, IP: net.ParseIP("10.0.0.2"), Responded: true},
				3: {Number: 3, IP: net.ParseIP("10.0.0.3"), Responded: true},
			},
			destReached:  false,
			destTTL:      31,
			expectedLen:  3,
			expectedLast: 3,
		},
		{
			name: "destination at hop 2",
			hopMap: map[int]Hop{
				1: {Number: 1, IP: net.ParseIP("10.0.0.1"), Responded: true},
				2: {Number: 2, IP: net.ParseIP("8.8.8.8"), Responded: true},
				3: {Number: 3, IP: net.ParseIP("10.0.0.3"), Responded: true},
				4: {Number: 4, Responded: false},
			},
			destReached:  true,
			destTTL:      2,
			expectedLen:  2,
			expectedLast: 2,
		},
		{
			name: "out of order hops",
			hopMap: map[int]Hop{
				3: {Number: 3, IP: net.ParseIP("10.0.0.3"), Responded: true},
				1: {Number: 1, IP: net.ParseIP("10.0.0.1"), Responded: true},
				2: {Number: 2, IP: net.ParseIP("10.0.0.2"), Responded: true},
			},
			destReached:  false,
			destTTL:      31,
			expectedLen:  3,
			expectedLast: 3,
		},
		{
			name:         "empty map",
			hopMap:       map[int]Hop{},
			destReached:  false,
			destTTL:      31,
			expectedLen:  0,
			expectedLast: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hops := tracer.buildHopList(tt.hopMap, tt.destReached, tt.destTTL)

			if len(hops) != tt.expectedLen {
				t.Errorf("len(hops) = %d, want %d", len(hops), tt.expectedLen)
			}

			if tt.expectedLen > 0 {
				lastHop := hops[len(hops)-1]
				if lastHop.Number != tt.expectedLast {
					t.Errorf("last hop number = %d, want %d", lastHop.Number, tt.expectedLast)
				}

				// Verify ordering
				for i := 1; i < len(hops); i++ {
					if hops[i].Number <= hops[i-1].Number {
						t.Errorf("hops not in order: hop[%d].Number=%d <= hop[%d].Number=%d",
							i, hops[i].Number, i-1, hops[i-1].Number)
					}
				}
			}
		})
	}
}

func TestConcurrentVsSequential(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	target := "127.0.0.1"

	// Test sequential
	seqConfig := DefaultConfig()
	seqConfig.MaxHops = 3
	seqConfig.ProbeCount = 1
	seqConfig.Sequential = true

	seqTracer, err := New(seqConfig)
	if err != nil {
		t.Fatalf("New(sequential) error = %v", err)
	}

	ctx := context.Background()
	seqStart := time.Now()
	seqResult, err := seqTracer.Trace(ctx, target)
	seqDuration := time.Since(seqStart)
	seqTracer.Close()

	if err != nil {
		t.Fatalf("Sequential Trace() error = %v", err)
	}

	// Test concurrent
	conConfig := DefaultConfig()
	conConfig.MaxHops = 3
	conConfig.ProbeCount = 1
	conConfig.Sequential = false

	conTracer, err := New(conConfig)
	if err != nil {
		t.Fatalf("New(concurrent) error = %v", err)
	}

	conStart := time.Now()
	conResult, err := conTracer.Trace(ctx, target)
	conDuration := time.Since(conStart)
	conTracer.Close()

	if err != nil {
		t.Fatalf("Concurrent Trace() error = %v", err)
	}

	// Results should be equivalent
	if len(seqResult.Hops) != len(conResult.Hops) {
		t.Errorf("Different hop counts: sequential=%d, concurrent=%d",
			len(seqResult.Hops), len(conResult.Hops))
	}

	t.Logf("Sequential: %d hops in %v", len(seqResult.Hops), seqDuration)
	t.Logf("Concurrent: %d hops in %v", len(conResult.Hops), conDuration)
}

func TestConcurrentContextCancellation(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultConfig()
	config.MaxHops = 30
	config.ProbeCount = 3
	config.Timeout = 5 * time.Second
	config.Sequential = false

	tracer, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer tracer.Close()

	// Cancel context quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err = tracer.Trace(ctx, "192.0.2.1") // TEST-NET, won't respond
	duration := time.Since(start)

	// Should complete quickly due to cancellation
	if duration > 2*time.Second {
		t.Errorf("Trace took %v, expected quick cancellation", duration)
	}
}

// canCreateRawSocket is defined in tracer_test.go
