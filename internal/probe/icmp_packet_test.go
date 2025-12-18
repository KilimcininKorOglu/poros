package probe

import (
	"testing"
	"time"
)

func TestNewICMPEchoRequest(t *testing.T) {
	payload := []byte("test payload")
	pkt := NewICMPEchoRequest(1234, 5678, payload)

	if pkt.Type != ICMPv4EchoRequest {
		t.Errorf("Type = %d, want %d", pkt.Type, ICMPv4EchoRequest)
	}
	if pkt.Code != 0 {
		t.Errorf("Code = %d, want 0", pkt.Code)
	}
	if pkt.Identifier != 1234 {
		t.Errorf("Identifier = %d, want 1234", pkt.Identifier)
	}
	if pkt.Sequence != 5678 {
		t.Errorf("Sequence = %d, want 5678", pkt.Sequence)
	}
	if string(pkt.Payload) != "test payload" {
		t.Errorf("Payload = %q, want %q", pkt.Payload, "test payload")
	}
}

func TestICMPPacket_Marshal(t *testing.T) {
	pkt := &ICMPPacket{
		Type:       ICMPv4EchoRequest,
		Code:       0,
		Identifier: 1,
		Sequence:   1,
		Payload:    nil,
	}

	data, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	if len(data) != 8 {
		t.Errorf("len(data) = %d, want 8", len(data))
	}

	// Verify type and code
	if data[0] != ICMPv4EchoRequest {
		t.Errorf("data[0] = %d, want %d", data[0], ICMPv4EchoRequest)
	}
	if data[1] != 0 {
		t.Errorf("data[1] = %d, want 0", data[1])
	}

	// Verify checksum is valid
	if !ValidateChecksum(data) {
		t.Error("Checksum validation failed")
	}
}

func TestICMPPacket_MarshalWithPayload(t *testing.T) {
	payload := []byte("Hello, ICMP!")
	pkt := NewICMPEchoRequest(100, 200, payload)

	data, err := pkt.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	expectedLen := 8 + len(payload)
	if len(data) != expectedLen {
		t.Errorf("len(data) = %d, want %d", len(data), expectedLen)
	}

	// Verify payload is at the end
	if string(data[8:]) != "Hello, ICMP!" {
		t.Errorf("Payload in marshaled data = %q, want %q", data[8:], "Hello, ICMP!")
	}

	// Verify checksum is valid
	if !ValidateChecksum(data) {
		t.Error("Checksum validation failed")
	}
}

func TestParseICMPPacket(t *testing.T) {
	// Create and marshal a packet
	original := NewICMPEchoRequest(1000, 2000, []byte("test"))
	data, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	// Parse it back
	parsed, err := ParseICMPPacket(data)
	if err != nil {
		t.Fatalf("ParseICMPPacket() error = %v", err)
	}

	if parsed.Type != original.Type {
		t.Errorf("Type = %d, want %d", parsed.Type, original.Type)
	}
	if parsed.Identifier != original.Identifier {
		t.Errorf("Identifier = %d, want %d", parsed.Identifier, original.Identifier)
	}
	if parsed.Sequence != original.Sequence {
		t.Errorf("Sequence = %d, want %d", parsed.Sequence, original.Sequence)
	}
	if string(parsed.Payload) != string(original.Payload) {
		t.Errorf("Payload = %q, want %q", parsed.Payload, original.Payload)
	}
}

func TestParseICMPPacket_TooShort(t *testing.T) {
	_, err := ParseICMPPacket([]byte{1, 2, 3})
	if err != ErrInvalidPacket {
		t.Errorf("ParseICMPPacket() error = %v, want %v", err, ErrInvalidPacket)
	}
}

func TestTimestampPayload(t *testing.T) {
	before := time.Now()
	payload := TimestampPayload([]byte("extra"))
	after := time.Now()

	if len(payload) != 8+5 {
		t.Errorf("len(payload) = %d, want 13", len(payload))
	}

	ts, ok := ExtractTimestamp(payload)
	if !ok {
		t.Fatal("ExtractTimestamp() failed")
	}

	if ts.Before(before) || ts.After(after) {
		t.Errorf("Timestamp %v not in range [%v, %v]", ts, before, after)
	}

	// Verify extra data
	if string(payload[8:]) != "extra" {
		t.Errorf("Extra data = %q, want %q", payload[8:], "extra")
	}
}

func TestExtractTimestamp_TooShort(t *testing.T) {
	_, ok := ExtractTimestamp([]byte{1, 2, 3})
	if ok {
		t.Error("ExtractTimestamp() should fail for short payload")
	}
}

func TestICMPPacket_TypeChecks(t *testing.T) {
	tests := []struct {
		name           string
		pkt            *ICMPPacket
		isEchoReply    bool
		isTimeExceeded bool
		isUnreachable  bool
	}{
		{
			name:           "Echo Reply v4",
			pkt:            &ICMPPacket{Type: ICMPv4EchoReply},
			isEchoReply:    true,
			isTimeExceeded: false,
			isUnreachable:  false,
		},
		{
			name:           "Time Exceeded v4",
			pkt:            &ICMPPacket{Type: ICMPv4TimeExceeded},
			isEchoReply:    false,
			isTimeExceeded: true,
			isUnreachable:  false,
		},
		{
			name:           "Unreachable v4",
			pkt:            &ICMPPacket{Type: ICMPv4Unreachable},
			isEchoReply:    false,
			isTimeExceeded: false,
			isUnreachable:  true,
		},
		{
			name:           "Echo Request v4",
			pkt:            &ICMPPacket{Type: ICMPv4EchoRequest},
			isEchoReply:    false,
			isTimeExceeded: false,
			isUnreachable:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.pkt.IsEchoReply(); got != tt.isEchoReply {
				t.Errorf("IsEchoReply() = %v, want %v", got, tt.isEchoReply)
			}
			if got := tt.pkt.IsTimeExceeded(); got != tt.isTimeExceeded {
				t.Errorf("IsTimeExceeded() = %v, want %v", got, tt.isTimeExceeded)
			}
			if got := tt.pkt.IsUnreachable(); got != tt.isUnreachable {
				t.Errorf("IsUnreachable() = %v, want %v", got, tt.isUnreachable)
			}
		})
	}
}

func BenchmarkICMPPacket_Marshal(b *testing.B) {
	payload := make([]byte, 56) // Standard ping payload size
	pkt := NewICMPEchoRequest(1, 1, payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pkt.Marshal()
	}
}

func BenchmarkParseICMPPacket(b *testing.B) {
	pkt := NewICMPEchoRequest(1, 1, make([]byte, 56))
	data, _ := pkt.Marshal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseICMPPacket(data)
	}
}
