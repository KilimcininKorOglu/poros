package probe

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// UDPProberConfig holds configuration for the UDP prober.
type UDPProberConfig struct {
	// Timeout is the maximum time to wait for a response
	Timeout time.Duration

	// BasePort is the starting destination port (default: 33434)
	BasePort int

	// IPv6 enables IPv6 mode
	IPv6 bool

	// PayloadSize is the size of the UDP payload in bytes
	PayloadSize int
}

// DefaultUDPProberConfig returns a default UDP prober configuration.
func DefaultUDPProberConfig() UDPProberConfig {
	return UDPProberConfig{
		Timeout:     3 * time.Second,
		BasePort:    33434,
		IPv6:        false,
		PayloadSize: 32,
	}
}

// UDPProber implements the Prober interface using UDP packets.
// It sends UDP packets to high-numbered ports and listens for
// ICMP responses (Time Exceeded or Destination Unreachable).
type UDPProber struct {
	config   UDPProberConfig
	icmpConn *icmp.PacketConn
	udpConn  *net.UDPConn
	sequence uint32
	id       uint16
}

// NewUDPProber creates a new UDP prober.
func NewUDPProber(config UDPProberConfig) (*UDPProber, error) {
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}
	if config.BasePort == 0 {
		config.BasePort = 33434
	}
	if config.PayloadSize == 0 {
		config.PayloadSize = 32
	}

	// Create ICMP listener for responses
	var icmpConn *icmp.PacketConn
	var err error

	if config.IPv6 {
		icmpConn, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
	} else {
		icmpConn, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create ICMP listener: %w", err)
	}

	// Create UDP socket for sending probes
	var udpConn *net.UDPConn
	if config.IPv6 {
		udpConn, err = net.ListenUDP("udp6", nil)
	} else {
		udpConn, err = net.ListenUDP("udp4", nil)
	}
	if err != nil {
		icmpConn.Close()
		return nil, fmt.Errorf("failed to create UDP socket: %w", err)
	}

	return &UDPProber{
		config:   config,
		icmpConn: icmpConn,
		udpConn:  udpConn,
		sequence: 0,
		id:       uint16(udpConn.LocalAddr().(*net.UDPAddr).Port),
	}, nil
}

// Probe sends a UDP probe with the specified TTL.
func (p *UDPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*Result, error) {
	if ttl < 1 || ttl > 255 {
		return nil, ErrInvalidTTL
	}

	// Set TTL on the UDP socket
	if err := p.setTTL(ttl); err != nil {
		return nil, fmt.Errorf("failed to set TTL: %w", err)
	}

	// Calculate destination port (increment for each probe)
	seq := atomic.AddUint32(&p.sequence, 1)
	destPort := p.config.BasePort + int(seq%100)

	// Build UDP payload with identifier
	payload := p.buildPayload(seq)

	// Prepare destination address
	destAddr := &net.UDPAddr{
		IP:   dest,
		Port: destPort,
	}

	// Set read deadline
	deadline := time.Now().Add(p.config.Timeout)
	if err := p.icmpConn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	// Record send time
	sendTime := time.Now()

	// Send UDP packet
	if _, err := p.udpConn.WriteToUDP(payload, destAddr); err != nil {
		return nil, fmt.Errorf("failed to send UDP packet: %w", err)
	}

	// Wait for ICMP response
	return p.receiveResponse(ctx, dest, destPort, sendTime, seq)
}

// setTTL sets the TTL on the UDP socket.
func (p *UDPProber) setTTL(ttl int) error {
	rawConn, err := p.udpConn.SyscallConn()
	if err != nil {
		return err
	}

	var setErr error
	if p.config.IPv6 {
		err = rawConn.Control(func(fd uintptr) {
			setErr = setIPv6HopLimit(fd, ttl)
		})
	} else {
		err = rawConn.Control(func(fd uintptr) {
			setErr = setIPv4TTL(fd, ttl)
		})
	}

	if err != nil {
		return err
	}
	return setErr
}

// buildPayload creates the UDP payload with sequence information.
func (p *UDPProber) buildPayload(seq uint32) []byte {
	payload := make([]byte, p.config.PayloadSize)

	// Store identifier and sequence in payload for matching responses
	if len(payload) >= 8 {
		binary.BigEndian.PutUint16(payload[0:2], p.id)
		binary.BigEndian.PutUint16(payload[2:4], uint16(seq))
		binary.BigEndian.PutUint32(payload[4:8], uint32(time.Now().UnixNano()))
	}

	return payload
}

// receiveResponse waits for an ICMP response to our UDP probe.
func (p *UDPProber) receiveResponse(ctx context.Context, dest net.IP, destPort int, sendTime time.Time, seq uint32) (*Result, error) {
	buf := make([]byte, 1500)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, peer, err := p.icmpConn.ReadFrom(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				return nil, ErrTimeout
			}
			return nil, fmt.Errorf("read error: %w", err)
		}

		rtt := time.Since(sendTime)

		// Parse ICMP message
		var proto int
		if p.config.IPv6 {
			proto = 58 // ICMPv6
		} else {
			proto = 1 // ICMPv4
		}

		msg, err := icmp.ParseMessage(proto, buf[:n])
		if err != nil {
			continue // Ignore malformed packets
		}

		// Check if this response is for our probe
		result, ok := p.matchResponse(msg, dest, destPort, seq)
		if ok {
			result.RTT = rtt
			result.ResponseIP = parseIP(peer)
			return result, nil
		}
	}
}

// matchResponse checks if an ICMP message is a response to our UDP probe.
func (p *UDPProber) matchResponse(msg *icmp.Message, dest net.IP, destPort int, seq uint32) (*Result, bool) {
	result := &Result{}

	if p.config.IPv6 {
		return p.matchResponseIPv6(msg, dest, destPort, seq, result)
	}
	return p.matchResponseIPv4(msg, dest, destPort, seq, result)
}

// matchResponseIPv4 handles IPv4 ICMP response matching.
func (p *UDPProber) matchResponseIPv4(msg *icmp.Message, dest net.IP, destPort int, seq uint32, result *Result) (*Result, bool) {
	result.ICMPType = msg.Type.(ipv4.ICMPType).Protocol()
	result.ICMPCode = msg.Code

	switch msg.Type {
	case ipv4.ICMPTypeTimeExceeded:
		// TTL expired - intermediate hop
		if body, ok := msg.Body.(*icmp.TimeExceeded); ok {
			if p.matchOriginalUDP(body.Data, dest, destPort) {
				result.TTLExpired = true
				return result, true
			}
		}

	case ipv4.ICMPTypeDestinationUnreachable:
		// Destination reached (port unreachable)
		if body, ok := msg.Body.(*icmp.DstUnreach); ok {
			if p.matchOriginalUDP(body.Data, dest, destPort) {
				result.Reached = true
				return result, true
			}
		}
	}

	return nil, false
}

// matchResponseIPv6 handles IPv6 ICMPv6 response matching.
func (p *UDPProber) matchResponseIPv6(msg *icmp.Message, dest net.IP, destPort int, seq uint32, result *Result) (*Result, bool) {
	result.ICMPType = msg.Type.(ipv6.ICMPType).Protocol()
	result.ICMPCode = msg.Code

	switch msg.Type {
	case ipv6.ICMPTypeTimeExceeded:
		// TTL expired - intermediate hop
		if body, ok := msg.Body.(*icmp.TimeExceeded); ok {
			if p.matchOriginalUDP(body.Data, dest, destPort) {
				result.TTLExpired = true
				return result, true
			}
		}

	case ipv6.ICMPTypeDestinationUnreachable:
		// Destination reached (port unreachable)
		if body, ok := msg.Body.(*icmp.DstUnreach); ok {
			if p.matchOriginalUDP(body.Data, dest, destPort) {
				result.Reached = true
				return result, true
			}
		}
	}

	return nil, false
}

// matchOriginalUDP checks if the ICMP error contains our original UDP packet.
func (p *UDPProber) matchOriginalUDP(data []byte, dest net.IP, destPort int) bool {
	// The ICMP error should contain the original IP header + 8 bytes of UDP
	// IPv4 header is typically 20 bytes, UDP header is 8 bytes

	if len(data) < 28 { // Minimum: 20 (IP) + 8 (UDP)
		return false
	}

	// Skip IP header (variable length, check IHL)
	ihl := int(data[0]&0x0f) * 4
	if ihl < 20 || len(data) < ihl+8 {
		return false
	}

	udpHeader := data[ihl:]

	// Extract source and destination ports from UDP header
	// srcPort := binary.BigEndian.Uint16(udpHeader[0:2])
	dstPort := binary.BigEndian.Uint16(udpHeader[2:4])

	// Check if destination port matches
	if int(dstPort) != destPort {
		return false
	}

	// Check destination IP from IP header
	destIPInPacket := net.IP(data[16:20])
	if !destIPInPacket.Equal(dest) {
		return false
	}

	return true
}

// parseIP extracts net.IP from net.Addr.
func parseIP(addr net.Addr) net.IP {
	switch v := addr.(type) {
	case *net.IPAddr:
		return v.IP
	case *net.UDPAddr:
		return v.IP
	default:
		return nil
	}
}

// Name returns the probe method name.
func (p *UDPProber) Name() string {
	return "udp"
}

// RequiresRoot returns true as UDP probing requires raw sockets for ICMP.
func (p *UDPProber) RequiresRoot() bool {
	return true
}

// Close releases resources held by the prober.
func (p *UDPProber) Close() error {
	var errs []error

	if p.icmpConn != nil {
		if err := p.icmpConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if p.udpConn != nil {
		if err := p.udpConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
