package probe

import (
	"testing"
)

func TestChecksum(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint16
	}{
		{
			name: "ICMP Echo Request example",
			// Type=8, Code=0, Checksum=0, ID=1, Seq=1
			data:     []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01},
			expected: 0xf7fd,
		},
		{
			name:     "Simple even length",
			data:     []byte{0x00, 0x01, 0x00, 0x02},
			expected: 0xfffc,
		},
		{
			name:     "Odd length data",
			data:     []byte{0x00, 0x01, 0xf2},
			expected: 0x0dfe,
		},
		{
			name:     "All zeros",
			data:     []byte{0x00, 0x00, 0x00, 0x00},
			expected: 0xffff,
		},
		{
			name:     "All ones",
			data:     []byte{0xff, 0xff, 0xff, 0xff},
			expected: 0x0000,
		},
		{
			name:     "Empty data",
			data:     []byte{},
			expected: 0xffff,
		},
		{
			name:     "Single byte",
			data:     []byte{0x45},
			expected: 0xbaff,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Checksum(tt.data)
			if result != tt.expected {
				t.Errorf("Checksum(%v) = 0x%04x, want 0x%04x", tt.data, result, tt.expected)
			}
		})
	}
}

func TestValidateChecksum(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		valid bool
	}{
		{
			name: "Valid ICMP packet with correct checksum",
			// Type=8, Code=0, Checksum=0xf7fd, ID=1, Seq=1
			data:  []byte{0x08, 0x00, 0xf7, 0xfd, 0x00, 0x01, 0x00, 0x01},
			valid: true,
		},
		{
			name: "Invalid checksum",
			// Type=8, Code=0, Checksum=0x0000, ID=1, Seq=1
			data:  []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01},
			valid: false,
		},
		{
			name:  "All zeros is valid",
			data:  []byte{0x00, 0x00, 0xff, 0xff},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateChecksum(tt.data)
			if result != tt.valid {
				t.Errorf("ValidateChecksum(%v) = %v, want %v", tt.data, result, tt.valid)
			}
		})
	}
}

func TestChecksumRoundTrip(t *testing.T) {
	// Create a packet, calculate checksum, insert it, and validate
	packet := []byte{0x08, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x01}

	// Calculate checksum (with checksum field as zero)
	checksum := Checksum(packet)

	// Insert checksum into packet (bytes 2-3)
	packet[2] = byte(checksum >> 8)
	packet[3] = byte(checksum & 0xff)

	// Validate
	if !ValidateChecksum(packet) {
		t.Errorf("Round-trip checksum validation failed for packet %v", packet)
	}
}

func BenchmarkChecksum(b *testing.B) {
	// Typical ICMP packet with 56 bytes of data
	data := make([]byte, 64)
	data[0] = 0x08 // ICMP Echo Request

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Checksum(data)
	}
}
