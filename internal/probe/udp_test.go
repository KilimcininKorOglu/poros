package probe

import (
	"context"
	"net"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestDefaultUDPProberConfig(t *testing.T) {
	config := DefaultUDPProberConfig()

	if config.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", config.Timeout)
	}
	if config.BasePort != 33434 {
		t.Errorf("BasePort = %d, want 33434", config.BasePort)
	}
	if config.PayloadSize != 32 {
		t.Errorf("PayloadSize = %d, want 32", config.PayloadSize)
	}
	if config.IPv6 != false {
		t.Error("IPv6 should be false by default")
	}
}

func TestNewUDPProber(t *testing.T) {
	if !canCreateRawSocketUDP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultUDPProberConfig()
	prober, err := NewUDPProber(config)
	if err != nil {
		t.Fatalf("NewUDPProber() error = %v", err)
	}
	defer prober.Close()

	if prober.Name() != "udp" {
		t.Errorf("Name() = %q, want %q", prober.Name(), "udp")
	}

	if !prober.RequiresRoot() {
		t.Error("RequiresRoot() should return true")
	}
}

func TestUDPProber_InvalidTTL(t *testing.T) {
	if !canCreateRawSocketUDP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultUDPProberConfig()
	prober, err := NewUDPProber(config)
	if err != nil {
		t.Fatalf("NewUDPProber() error = %v", err)
	}
	defer prober.Close()

	ctx := context.Background()
	dest := net.ParseIP("127.0.0.1")

	// Test TTL = 0 (invalid)
	_, err = prober.Probe(ctx, dest, 0)
	if err != ErrInvalidTTL {
		t.Errorf("Probe(ttl=0) error = %v, want ErrInvalidTTL", err)
	}

	// Test TTL = 256 (invalid)
	_, err = prober.Probe(ctx, dest, 256)
	if err != ErrInvalidTTL {
		t.Errorf("Probe(ttl=256) error = %v, want ErrInvalidTTL", err)
	}
}

func TestUDPProber_BuildPayload(t *testing.T) {
	if !canCreateRawSocketUDP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultUDPProberConfig()
	config.PayloadSize = 32

	prober, err := NewUDPProber(config)
	if err != nil {
		t.Fatalf("NewUDPProber() error = %v", err)
	}
	defer prober.Close()

	payload := prober.buildPayload(1)

	if len(payload) != 32 {
		t.Errorf("Payload length = %d, want 32", len(payload))
	}

	// Check that ID is set in payload
	if payload[0] == 0 && payload[1] == 0 {
		t.Error("ID should be non-zero in payload")
	}
}

func TestUDPProber_ProbeLocalhost(t *testing.T) {
	if !canCreateRawSocketUDP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultUDPProberConfig()
	config.Timeout = 2 * time.Second

	prober, err := NewUDPProber(config)
	if err != nil {
		t.Fatalf("NewUDPProber() error = %v", err)
	}
	defer prober.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dest := net.ParseIP("127.0.0.1")

	// Probe localhost - should get ICMP Port Unreachable (destination reached)
	// or timeout if ICMP is blocked
	result, err := prober.Probe(ctx, dest, 64)
	if err != nil {
		// Timeout is acceptable for localhost UDP
		if err == ErrTimeout {
			t.Log("Probe timed out (expected for some configurations)")
			return
		}
		t.Fatalf("Probe() error = %v", err)
	}

	t.Logf("Got response from %v, RTT=%v, Reached=%v", 
		result.ResponseIP, result.RTT, result.Reached)
}

func TestUDPProber_ContextCancellation(t *testing.T) {
	if !canCreateRawSocketUDP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultUDPProberConfig()
	config.Timeout = 5 * time.Second

	prober, err := NewUDPProber(config)
	if err != nil {
		t.Fatalf("NewUDPProber() error = %v", err)
	}
	defer prober.Close()

	// Create an already cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	dest := net.ParseIP("192.0.2.1") // TEST-NET, won't respond

	_, err = prober.Probe(ctx, dest, 1)
	if err == nil {
		t.Error("Probe() should fail with cancelled context")
	}
}

// canCreateRawSocketUDP checks if we have privileges to create raw sockets.
func canCreateRawSocketUDP() bool {
	if runtime.GOOS == "windows" {
		// On Windows, try to detect admin privileges
		_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
		return err == nil
	}
	return os.Getuid() == 0
}
