package probe

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDefaultParisProberConfig(t *testing.T) {
	config := DefaultParisProberConfig()

	if config.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", config.Timeout)
	}
	if config.Method != MethodUDP {
		t.Errorf("Method = %v, want UDP", config.Method)
	}
	if config.Port != 33434 {
		t.Errorf("Port = %d, want 33434", config.Port)
	}
	if config.IPv6 != false {
		t.Error("IPv6 should be false by default")
	}
}

func TestNewParisProber(t *testing.T) {
	if !canCreateRawSocketParis() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultParisProberConfig()
	prober, err := NewParisProber(config)
	if err != nil {
		t.Fatalf("NewParisProber() error = %v", err)
	}
	defer prober.Close()

	if prober.Name() != "paris-udp" {
		t.Errorf("Name() = %q, want %q", prober.Name(), "paris-udp")
	}

	if !prober.RequiresRoot() {
		t.Error("RequiresRoot() should return true")
	}

	// Check flow ID was generated
	if prober.FlowID() == 0 {
		t.Error("FlowID should be non-zero")
	}
}

func TestParisProber_ConstantFlowID(t *testing.T) {
	if !canCreateRawSocketParis() {
		t.Skip("Skipping: requires elevated privileges")
	}

	// Create with explicit flow ID
	config := ParisProberConfig{
		Timeout: 2 * time.Second,
		Method:  MethodUDP,
		Port:    33434,
		FlowID:  12345,
	}

	prober, err := NewParisProber(config)
	if err != nil {
		t.Fatalf("NewParisProber() error = %v", err)
	}
	defer prober.Close()

	if prober.FlowID() != 12345 {
		t.Errorf("FlowID() = %d, want 12345", prober.FlowID())
	}
}

func TestParisProber_InvalidTTL(t *testing.T) {
	if !canCreateRawSocketParis() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultParisProberConfig()
	prober, err := NewParisProber(config)
	if err != nil {
		t.Fatalf("NewParisProber() error = %v", err)
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

func TestParisProber_ICMPMethod(t *testing.T) {
	if !canCreateRawSocketParis() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := ParisProberConfig{
		Timeout: 2 * time.Second,
		Method:  MethodICMP,
		FlowID:  54321,
	}

	prober, err := NewParisProber(config)
	if err != nil {
		t.Fatalf("NewParisProber() error = %v", err)
	}
	defer prober.Close()

	if prober.Name() != "paris-icmp" {
		t.Errorf("Name() = %q, want %q", prober.Name(), "paris-icmp")
	}
}

func TestParisProber_BuildPayload(t *testing.T) {
	if !canCreateRawSocketParis() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := ParisProberConfig{
		Timeout: 2 * time.Second,
		Method:  MethodUDP,
		FlowID:  0xABCD,
	}

	prober, err := NewParisProber(config)
	if err != nil {
		t.Fatalf("NewParisProber() error = %v", err)
	}
	defer prober.Close()

	payload := prober.buildParisUDPPayload()

	if len(payload) != 32 {
		t.Errorf("Payload length = %d, want 32", len(payload))
	}

	// Check flow ID is embedded
	flowID := uint16(payload[0])<<8 | uint16(payload[1])
	if flowID != 0xABCD {
		t.Errorf("Embedded FlowID = 0x%04X, want 0xABCD", flowID)
	}
}

func TestParisProber_BuildICMPPacket(t *testing.T) {
	if !canCreateRawSocketParis() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := ParisProberConfig{
		Timeout: 2 * time.Second,
		Method:  MethodICMP,
		FlowID:  0x1234,
	}

	prober, err := NewParisProber(config)
	if err != nil {
		t.Fatalf("NewParisProber() error = %v", err)
	}
	defer prober.Close()

	packet := prober.buildParisICMPPacket(0x1234, 1)

	// Check packet structure
	if len(packet) != 18 {
		t.Errorf("Packet length = %d, want 18", len(packet))
	}

	// Check type (Echo Request for IPv4)
	if packet[0] != 8 {
		t.Errorf("ICMP type = %d, want 8", packet[0])
	}

	// Check ID
	id := uint16(packet[4])<<8 | uint16(packet[5])
	if id != 0x1234 {
		t.Errorf("ICMP ID = 0x%04X, want 0x1234", id)
	}

	// Check sequence
	seq := uint16(packet[6])<<8 | uint16(packet[7])
	if seq != 1 {
		t.Errorf("ICMP Seq = %d, want 1", seq)
	}
}

// canCreateRawSocketParis checks if we can create raw sockets for Paris.
func canCreateRawSocketParis() bool {
	conn, err := icmpListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// Helper to avoid import cycle
func icmpListenPacket(network, address string) (net.PacketConn, error) {
	return net.ListenPacket(network, address)
}
