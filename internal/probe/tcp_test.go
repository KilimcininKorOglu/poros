package probe

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestDefaultTCPProberConfig(t *testing.T) {
	config := DefaultTCPProberConfig()

	if config.Timeout != 3*time.Second {
		t.Errorf("Timeout = %v, want 3s", config.Timeout)
	}
	if config.Port != 80 {
		t.Errorf("Port = %d, want 80", config.Port)
	}
	if config.IPv6 != false {
		t.Error("IPv6 should be false by default")
	}
}

func TestNewTCPProber(t *testing.T) {
	if !canCreateRawSocketTCP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultTCPProberConfig()
	prober, err := NewTCPProber(config)
	if err != nil {
		t.Fatalf("NewTCPProber() error = %v", err)
	}
	defer prober.Close()

	if prober.Name() != "tcp" {
		t.Errorf("Name() = %q, want %q", prober.Name(), "tcp")
	}

	if !prober.RequiresRoot() {
		t.Error("RequiresRoot() should return true")
	}
}

func TestTCPProber_InvalidTTL(t *testing.T) {
	if !canCreateRawSocketTCP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultTCPProberConfig()
	prober, err := NewTCPProber(config)
	if err != nil {
		t.Fatalf("NewTCPProber() error = %v", err)
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

func TestTCPProber_BuildSYNPacket(t *testing.T) {
	if !canCreateRawSocketTCP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := DefaultTCPProberConfig()
	prober, err := NewTCPProber(config)
	if err != nil {
		t.Fatalf("NewTCPProber() error = %v", err)
	}
	defer prober.Close()

	src := net.ParseIP("192.168.1.1")
	dst := net.ParseIP("8.8.8.8")
	srcPort := uint16(12345)
	dstPort := uint16(80)
	seq := uint32(1)

	packet := prober.buildSYNPacket(src, dst, srcPort, dstPort, seq)

	// Check packet length (20 bytes TCP header)
	if len(packet) != 20 {
		t.Errorf("Packet length = %d, want 20", len(packet))
	}

	// Check source port
	pktSrcPort := uint16(packet[0])<<8 | uint16(packet[1])
	if pktSrcPort != srcPort {
		t.Errorf("Source port = %d, want %d", pktSrcPort, srcPort)
	}

	// Check destination port
	pktDstPort := uint16(packet[2])<<8 | uint16(packet[3])
	if pktDstPort != dstPort {
		t.Errorf("Destination port = %d, want %d", pktDstPort, dstPort)
	}

	// Check SYN flag (byte 13, bit 1)
	if packet[13] != 0x02 {
		t.Errorf("Flags = 0x%02x, want 0x02 (SYN)", packet[13])
	}

	// Check data offset (byte 12, upper nibble should be 5)
	dataOffset := packet[12] >> 4
	if dataOffset != 5 {
		t.Errorf("Data offset = %d, want 5", dataOffset)
	}
}

func TestTCPProber_Port443(t *testing.T) {
	if !canCreateRawSocketTCP() {
		t.Skip("Skipping: requires elevated privileges")
	}

	config := TCPProberConfig{
		Timeout: 2 * time.Second,
		Port:    443,
		IPv6:    false,
	}

	prober, err := NewTCPProber(config)
	if err != nil {
		t.Fatalf("NewTCPProber() error = %v", err)
	}
	defer prober.Close()

	if prober.config.Port != 443 {
		t.Errorf("Port = %d, want 443", prober.config.Port)
	}
}

func TestGetOutboundIP(t *testing.T) {
	ip := getOutboundIP(false)

	if ip == nil {
		t.Error("getOutboundIP() returned nil")
		return
	}

	// Should be a valid IPv4 address
	if ip.To4() == nil && !ip.Equal(net.ParseIP("0.0.0.0")) {
		t.Errorf("Expected IPv4 address, got %v", ip)
	}

	t.Logf("Outbound IP: %v", ip)
}

// canCreateRawSocketTCP checks if we have privileges for raw TCP sockets.
func canCreateRawSocketTCP() bool {
	// Try to create a raw TCP socket
	conn, err := net.ListenPacket("ip4:tcp", "0.0.0.0")
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
