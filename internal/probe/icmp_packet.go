package probe

import (
	"encoding/binary"
	"time"
)

// ICMP message types for IPv4
const (
	ICMPv4EchoReply      = 0
	ICMPv4Unreachable    = 3
	ICMPv4EchoRequest    = 8
	ICMPv4TimeExceeded   = 11
	ICMPv4ParameterProblem = 12
)

// ICMP unreachable codes
const (
	ICMPv4NetUnreachable     = 0
	ICMPv4HostUnreachable    = 1
	ICMPv4ProtocolUnreachable = 2
	ICMPv4PortUnreachable    = 3
)

// ICMP message types for IPv6
const (
	ICMPv6Unreachable   = 1
	ICMPv6TimeExceeded  = 3
	ICMPv6EchoRequest   = 128
	ICMPv6EchoReply     = 129
)

// ICMPPacket represents an ICMP Echo Request/Reply packet.
type ICMPPacket struct {
	Type       uint8
	Code       uint8
	Checksum   uint16
	Identifier uint16
	Sequence   uint16
	Payload    []byte
	IPv6       bool // Indicates if this is an IPv6 ICMP packet
}

// NewICMPEchoRequest creates a new ICMP Echo Request packet.
func NewICMPEchoRequest(id, seq uint16, payload []byte) *ICMPPacket {
	return &ICMPPacket{
		Type:       ICMPv4EchoRequest,
		Code:       0,
		Identifier: id,
		Sequence:   seq,
		Payload:    payload,
	}
}

// NewICMPv6EchoRequest creates a new ICMPv6 Echo Request packet.
func NewICMPv6EchoRequest(id, seq uint16, payload []byte) *ICMPPacket {
	return &ICMPPacket{
		Type:       ICMPv6EchoRequest,
		Code:       0,
		Identifier: id,
		Sequence:   seq,
		Payload:    payload,
	}
}

// Marshal serializes the ICMP packet to bytes, calculating the checksum.
func (p *ICMPPacket) Marshal() ([]byte, error) {
	// ICMP header is 8 bytes + payload
	buf := make([]byte, 8+len(p.Payload))

	buf[0] = p.Type
	buf[1] = p.Code
	// Checksum at bytes 2-3 (set to 0 for calculation)
	buf[2] = 0
	buf[3] = 0
	binary.BigEndian.PutUint16(buf[4:6], p.Identifier)
	binary.BigEndian.PutUint16(buf[6:8], p.Sequence)

	// Copy payload
	if len(p.Payload) > 0 {
		copy(buf[8:], p.Payload)
	}

	// Calculate and set checksum
	p.Checksum = Checksum(buf)
	binary.BigEndian.PutUint16(buf[2:4], p.Checksum)

	return buf, nil
}

// MarshalWithoutChecksum serializes without calculating checksum (for IPv6 where
// checksum is calculated by the kernel using pseudo-header).
func (p *ICMPPacket) MarshalWithoutChecksum() ([]byte, error) {
	buf := make([]byte, 8+len(p.Payload))

	buf[0] = p.Type
	buf[1] = p.Code
	binary.BigEndian.PutUint16(buf[2:4], p.Checksum)
	binary.BigEndian.PutUint16(buf[4:6], p.Identifier)
	binary.BigEndian.PutUint16(buf[6:8], p.Sequence)

	if len(p.Payload) > 0 {
		copy(buf[8:], p.Payload)
	}

	return buf, nil
}

// ParseICMPPacket parses an ICMP packet from bytes.
func ParseICMPPacket(data []byte) (*ICMPPacket, error) {
	if len(data) < 8 {
		return nil, ErrInvalidPacket
	}

	p := &ICMPPacket{
		Type:       data[0],
		Code:       data[1],
		Checksum:   binary.BigEndian.Uint16(data[2:4]),
		Identifier: binary.BigEndian.Uint16(data[4:6]),
		Sequence:   binary.BigEndian.Uint16(data[6:8]),
	}

	if len(data) > 8 {
		p.Payload = make([]byte, len(data)-8)
		copy(p.Payload, data[8:])
	}

	return p, nil
}

// TimestampPayload creates a payload containing the current timestamp.
// This is used to calculate RTT when the response is received.
func TimestampPayload(extraData []byte) []byte {
	// 8 bytes for timestamp + extra data
	payload := make([]byte, 8+len(extraData))
	binary.BigEndian.PutUint64(payload[0:8], uint64(time.Now().UnixNano()))
	if len(extraData) > 0 {
		copy(payload[8:], extraData)
	}
	return payload
}

// ExtractTimestamp extracts the timestamp from a payload.
func ExtractTimestamp(payload []byte) (time.Time, bool) {
	if len(payload) < 8 {
		return time.Time{}, false
	}
	nanos := binary.BigEndian.Uint64(payload[0:8])
	return time.Unix(0, int64(nanos)), true
}

// IsEchoReply checks if this is an ICMP Echo Reply.
func (p *ICMPPacket) IsEchoReply() bool {
	if p.IPv6 {
		return p.Type == ICMPv6EchoReply
	}
	return p.Type == ICMPv4EchoReply
}

// IsTimeExceeded checks if this is a Time Exceeded message.
func (p *ICMPPacket) IsTimeExceeded() bool {
	if p.IPv6 {
		return p.Type == ICMPv6TimeExceeded
	}
	return p.Type == ICMPv4TimeExceeded
}

// IsUnreachable checks if this is a Destination Unreachable message.
func (p *ICMPPacket) IsUnreachable() bool {
	if p.IPv6 {
		return p.Type == ICMPv6Unreachable
	}
	return p.Type == ICMPv4Unreachable
}
