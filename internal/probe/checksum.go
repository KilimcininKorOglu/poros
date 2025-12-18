package probe

// Checksum calculates the Internet Checksum (RFC 1071) for ICMP packets.
// This is used for ICMP, IP, UDP, and TCP header checksums.
func Checksum(data []byte) uint16 {
	var sum uint32

	// Sum all 16-bit words
	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	// Add left-over byte, if any (pad with zero)
	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}

	// Return one's complement
	return ^uint16(sum)
}

// ValidateChecksum verifies that a packet's checksum is correct.
// Returns true if the checksum is valid (sum including checksum equals 0xFFFF).
func ValidateChecksum(data []byte) bool {
	var sum uint32

	for i := 0; i < len(data)-1; i += 2 {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
	}

	if len(data)%2 == 1 {
		sum += uint32(data[len(data)-1]) << 8
	}

	for sum > 0xffff {
		sum = (sum >> 16) + (sum & 0xffff)
	}

	// Valid if result is 0xFFFF (all ones)
	return uint16(sum) == 0xffff
}
