package probe

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestNewICMPProber(t *testing.T) {
	// Skip if not running as admin/root
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	prober, err := NewICMPProber(ICMPProberConfig{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewICMPProber() error = %v", err)
	}
	defer prober.Close()

	if prober.Name() != "icmp" {
		t.Errorf("Name() = %q, want %q", prober.Name(), "icmp")
	}

	if !prober.RequiresRoot() {
		t.Error("RequiresRoot() = false, want true")
	}
}

func TestICMPProber_ProbeLocalhost(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	prober, err := NewICMPProber(ICMPProberConfig{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewICMPProber() error = %v", err)
	}
	defer prober.Close()

	ctx := context.Background()
	result, err := prober.Probe(ctx, net.ParseIP("127.0.0.1"), 64)
	if err != nil {
		t.Fatalf("Probe() error = %v", err)
	}

	if !result.Reached {
		t.Error("Probe to localhost should reach destination")
	}

	if result.RTT > time.Second {
		t.Errorf("RTT to localhost = %v, expected < 1s", result.RTT)
	}
}

func TestICMPProber_InvalidTTL(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	prober, err := NewICMPProber(ICMPProberConfig{
		Timeout: 2 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewICMPProber() error = %v", err)
	}
	defer prober.Close()

	ctx := context.Background()

	// Test TTL = 0
	_, err = prober.Probe(ctx, net.ParseIP("127.0.0.1"), 0)
	if err != ErrInvalidTTL {
		t.Errorf("Probe(TTL=0) error = %v, want ErrInvalidTTL", err)
	}

	// Test TTL > 255
	_, err = prober.Probe(ctx, net.ParseIP("127.0.0.1"), 256)
	if err != ErrInvalidTTL {
		t.Errorf("Probe(TTL=256) error = %v, want ErrInvalidTTL", err)
	}
}

func TestICMPProber_ContextCancellation(t *testing.T) {
	if !canCreateRawSocket() {
		t.Skip("Skipping: requires elevated privileges")
	}

	prober, err := NewICMPProber(ICMPProberConfig{
		Timeout: 10 * time.Second, // Long timeout
	})
	if err != nil {
		t.Fatalf("NewICMPProber() error = %v", err)
	}
	defer prober.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = prober.Probe(ctx, net.ParseIP("192.0.2.1"), 64) // TEST-NET, should not respond
	if err == nil {
		t.Error("Probe with cancelled context should return error")
	}
}

// canCreateRawSocket checks if we can create raw ICMP sockets.
func canCreateRawSocket() bool {
	// On Windows, check if running as administrator
	if runtime.GOOS == "windows" {
		// Try to open a privileged resource
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}

	// On Unix-like systems, check if running as root
	return os.Getuid() == 0
}
