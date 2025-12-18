package trace

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ProbeMethod != ProbeICMP {
		t.Errorf("ProbeMethod = %v, want %v", config.ProbeMethod, ProbeICMP)
	}
	if config.ProbeCount != 3 {
		t.Errorf("ProbeCount = %d, want 3", config.ProbeCount)
	}
	if config.MaxHops != 30 {
		t.Errorf("MaxHops = %d, want 30", config.MaxHops)
	}
	if config.FirstHop != 1 {
		t.Errorf("FirstHop = %d, want 1", config.FirstHop)
	}
	if config.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", config.Timeout)
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr error
	}{
		{
			name:    "valid config",
			config:  *DefaultConfig(),
			wantErr: nil,
		},
		{
			name:    "invalid max hops (0)",
			config:  Config{MaxHops: 0, ProbeCount: 3, Timeout: time.Second, FirstHop: 1},
			wantErr: ErrInvalidMaxHops,
		},
		{
			name:    "invalid max hops (>255)",
			config:  Config{MaxHops: 256, ProbeCount: 3, Timeout: time.Second, FirstHop: 1},
			wantErr: ErrInvalidMaxHops,
		},
		{
			name:    "invalid probe count (0)",
			config:  Config{MaxHops: 30, ProbeCount: 0, Timeout: time.Second, FirstHop: 1},
			wantErr: ErrInvalidProbeCount,
		},
		{
			name:    "invalid probe count (>10)",
			config:  Config{MaxHops: 30, ProbeCount: 11, Timeout: time.Second, FirstHop: 1},
			wantErr: ErrInvalidProbeCount,
		},
		{
			name:    "invalid timeout (too short)",
			config:  Config{MaxHops: 30, ProbeCount: 3, Timeout: 50 * time.Millisecond, FirstHop: 1},
			wantErr: ErrInvalidTimeout,
		},
		{
			name:    "invalid first hop (0)",
			config:  Config{MaxHops: 30, ProbeCount: 3, Timeout: time.Second, FirstHop: 0},
			wantErr: ErrInvalidFirstHop,
		},
		{
			name:    "invalid first hop (> max)",
			config:  Config{MaxHops: 30, ProbeCount: 3, Timeout: time.Second, FirstHop: 31},
			wantErr: ErrInvalidFirstHop,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateRTTStats(t *testing.T) {
	tests := []struct {
		name       string
		rtts       []float64
		wantAvg    float64
		wantMin    float64
		wantMax    float64
		wantJitter float64
	}{
		{
			name:       "single value",
			rtts:       []float64{10.0},
			wantAvg:    10.0,
			wantMin:    10.0,
			wantMax:    10.0,
			wantJitter: 0,
		},
		{
			name:       "multiple values",
			rtts:       []float64{10.0, 20.0, 30.0},
			wantAvg:    20.0,
			wantMin:    10.0,
			wantMax:    30.0,
			wantJitter: 20.0,
		},
		{
			name:       "with timeouts",
			rtts:       []float64{10.0, -1, 20.0, -1},
			wantAvg:    15.0,
			wantMin:    10.0,
			wantMax:    20.0,
			wantJitter: 10.0,
		},
		{
			name:       "all timeouts",
			rtts:       []float64{-1, -1, -1},
			wantAvg:    0,
			wantMin:    0,
			wantMax:    0,
			wantJitter: 0,
		},
		{
			name:       "empty",
			rtts:       []float64{},
			wantAvg:    0,
			wantMin:    0,
			wantMax:    0,
			wantJitter: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			avg, min, max, jitter := calculateRTTStats(tt.rtts)
			if avg != tt.wantAvg {
				t.Errorf("avg = %v, want %v", avg, tt.wantAvg)
			}
			if min != tt.wantMin {
				t.Errorf("min = %v, want %v", min, tt.wantMin)
			}
			if max != tt.wantMax {
				t.Errorf("max = %v, want %v", max, tt.wantMax)
			}
			if jitter != tt.wantJitter {
				t.Errorf("jitter = %v, want %v", jitter, tt.wantJitter)
			}
		})
	}
}

func TestCalculateLossPercent(t *testing.T) {
	tests := []struct {
		name string
		rtts []float64
		want float64
	}{
		{
			name: "no loss",
			rtts: []float64{10.0, 20.0, 30.0},
			want: 0,
		},
		{
			name: "50% loss",
			rtts: []float64{10.0, -1, 20.0, -1},
			want: 50,
		},
		{
			name: "100% loss",
			rtts: []float64{-1, -1, -1},
			want: 100,
		},
		{
			name: "empty",
			rtts: []float64{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateLossPercent(tt.rtts)
			if got != tt.want {
				t.Errorf("calculateLossPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew_InvalidConfig(t *testing.T) {
	config := &Config{
		MaxHops:    0, // Invalid
		ProbeCount: 3,
		Timeout:    time.Second,
		FirstHop:   1,
	}

	_, err := New(config)
	if err == nil {
		t.Error("New() should fail with invalid config")
	}
}

func TestTracer_ResolveTarget(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	tracer, err := New(DefaultConfig())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer tracer.Close()

	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{
			name:    "IPv4 address",
			target:  "127.0.0.1",
			wantErr: false,
		},
		{
			name:    "localhost",
			target:  "localhost",
			wantErr: false,
		},
		{
			name:    "invalid hostname",
			target:  "this.hostname.does.not.exist.example",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ip, err := tracer.resolveTarget(ctx, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("resolveTarget() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && ip == nil {
				t.Error("resolveTarget() returned nil IP without error")
			}
		})
	}
}

func TestTracer_TraceLocalhost(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultConfig()
	config.MaxHops = 5
	config.ProbeCount = 1
	config.Timeout = 2 * time.Second

	tracer, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer tracer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := tracer.Trace(ctx, "127.0.0.1")
	if err != nil {
		t.Fatalf("Trace() error = %v", err)
	}

	if result.Target != "127.0.0.1" {
		t.Errorf("Target = %q, want %q", result.Target, "127.0.0.1")
	}

	if !result.ResolvedIP.Equal(net.ParseIP("127.0.0.1")) {
		t.Errorf("ResolvedIP = %v, want 127.0.0.1", result.ResolvedIP)
	}

	if result.ProbeMethod != "icmp" {
		t.Errorf("ProbeMethod = %q, want %q", result.ProbeMethod, "icmp")
	}

	if !result.Completed {
		t.Error("Trace to localhost should complete")
	}

	if len(result.Hops) == 0 {
		t.Error("Trace should have at least one hop")
	}
}

// canCreateRawSocket checks if we can create raw ICMP sockets.
func canCreateRawSocket() bool {
	if runtime.GOOS == "windows" {
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return os.Getuid() == 0
}
