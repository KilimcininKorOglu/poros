package probe

import (
	"context"
	"encoding/binary"
	"net"
	"os"
	"sync/atomic"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// ICMPProber implements the Prober interface using ICMP Echo requests.
type ICMPProber struct {
	conn4      *icmp.PacketConn // IPv4 connection
	conn6      *icmp.PacketConn // IPv6 connection
	identifier uint16
	sequence   uint32
	timeout    time.Duration
	ipv6       bool
}

// ICMPProberConfig holds configuration for the ICMP prober.
type ICMPProberConfig struct {
	Timeout    time.Duration
	IPv6       bool
	Identifier uint16 // If 0, uses process ID
}

// NewICMPProber creates a new ICMP prober.
func NewICMPProber(config ICMPProberConfig) (*ICMPProber, error) {
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}

	identifier := config.Identifier
	if identifier == 0 {
		identifier = uint16(os.Getpid() & 0xffff)
	}

	p := &ICMPProber{
		identifier: identifier,
		timeout:    config.Timeout,
		ipv6:       config.IPv6,
	}

	var err error
	if config.IPv6 {
		p.conn6, err = icmp.ListenPacket("ip6:ipv6-icmp", "::")
		if err != nil {
			// Try unprivileged mode
			p.conn6, err = icmp.ListenPacket("udp6", "::")
		}
	} else {
		p.conn4, err = icmp.ListenPacket("ip4:icmp", "0.0.0.0")
		if err != nil {
			// Try unprivileged mode on non-Linux
			p.conn4, err = icmp.ListenPacket("udp4", "0.0.0.0")
		}
	}

	if err != nil {
		return nil, err
	}

	return p, nil
}

// Probe sends an ICMP Echo Request with the given TTL and waits for a response.
func (p *ICMPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*Result, error) {
	if ttl < 1 || ttl > 255 {
		return nil, ErrInvalidTTL
	}

	conn := p.conn4
	proto := 1 // ICMP protocol number
	var icmpType icmp.Type = ipv4.ICMPTypeEcho

	if p.ipv6 || dest.To4() == nil {
		conn = p.conn6
		proto = 58 // ICMPv6 protocol number
		icmpType = ipv6.ICMPTypeEchoRequest
	}

	if conn == nil {
		return nil, ErrSocketClosed
	}

	// Set TTL
	if err := p.setTTL(conn, ttl); err != nil {
		return nil, err
	}

	// Build ICMP message
	seq := uint16(atomic.AddUint32(&p.sequence, 1))
	payload := TimestampPayload(nil)

	msg := &icmp.Message{
		Type: icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(p.identifier),
			Seq:  int(seq),
			Data: payload,
		},
	}

	msgBytes, err := msg.Marshal(nil)
	if err != nil {
		return nil, err
	}

	// Set deadline
	deadline := time.Now().Add(p.timeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	conn.SetDeadline(deadline)

	// Send probe
	sendTime := time.Now()
	var dst net.Addr
	if p.ipv6 || dest.To4() == nil {
		dst = &net.IPAddr{IP: dest}
	} else {
		dst = &net.IPAddr{IP: dest}
	}

	if _, err := conn.WriteTo(msgBytes, dst); err != nil {
		return nil, err
	}

	// Wait for response
	return p.waitForResponse(ctx, conn, proto, dest, seq, sendTime)
}

// setTTL sets the TTL/Hop Limit for outgoing packets.
func (p *ICMPProber) setTTL(conn *icmp.PacketConn, ttl int) error {
	if p.ipv6 {
		return conn.IPv6PacketConn().SetHopLimit(ttl)
	}
	return conn.IPv4PacketConn().SetTTL(ttl)
}

// waitForResponse waits for an ICMP response matching our probe.
func (p *ICMPProber) waitForResponse(ctx context.Context, conn *icmp.PacketConn, proto int,
	dest net.IP, expectedSeq uint16, sendTime time.Time) (*Result, error) {

	buf := make([]byte, 1500)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			if isTimeoutError(err) {
				return nil, ErrTimeout
			}
			return nil, err
		}

		// Parse the response
		result, matched := p.parseResponse(buf[:n], peer, proto, dest, expectedSeq, sendTime)
		if matched {
			return result, nil
		}
		// Not our packet, continue waiting
	}
}

// parseResponse parses an ICMP response and checks if it matches our probe.
func (p *ICMPProber) parseResponse(data []byte, peer net.Addr, proto int,
	dest net.IP, expectedSeq uint16, sendTime time.Time) (*Result, bool) {

	msg, err := icmp.ParseMessage(proto, data)
	if err != nil {
		return nil, false
	}

	rtt := time.Since(sendTime)
	peerIP := extractIP(peer)

	switch msg.Type {
	case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
		// Echo Reply - destination reached
		echo, ok := msg.Body.(*icmp.Echo)
		if !ok {
			return nil, false
		}
		if uint16(echo.ID) != p.identifier || uint16(echo.Seq) != expectedSeq {
			return nil, false
		}
		return &Result{
			ResponseIP: peerIP,
			RTT:        rtt,
			ICMPType:   int(msg.Type.(ipv4.ICMPType)),
			ICMPCode:   int(msg.Code),
			Reached:    true,
			TTLExpired: false,
		}, true

	case ipv4.ICMPTypeTimeExceeded, ipv6.ICMPTypeTimeExceeded:
		// Time Exceeded - intermediate hop
		return p.parseTimeExceeded(msg, peerIP, rtt, expectedSeq)

	case ipv4.ICMPTypeDestinationUnreachable, ipv6.ICMPTypeDestinationUnreachable:
		// Destination Unreachable
		return p.parseUnreachable(msg, peerIP, rtt, expectedSeq)
	}

	return nil, false
}

// parseTimeExceeded parses a Time Exceeded message.
func (p *ICMPProber) parseTimeExceeded(msg *icmp.Message, peerIP net.IP, rtt time.Duration, expectedSeq uint16) (*Result, bool) {
	// Time Exceeded contains the original IP header + first 8 bytes of original packet
	body, ok := msg.Body.(*icmp.TimeExceeded)
	if !ok {
		return nil, false
	}

	// Extract original ICMP header from the payload
	// IP header is typically 20 bytes, then ICMP header
	origData := body.Data
	if len(origData) < 28 { // 20 (IP) + 8 (ICMP header)
		return nil, false
	}

	// Find the ICMP header in the original packet
	// IPv4 header length is in the first byte (lower 4 bits * 4)
	ipHeaderLen := int(origData[0]&0x0f) * 4
	if len(origData) < ipHeaderLen+8 {
		return nil, false
	}

	icmpHeader := origData[ipHeaderLen:]

	// Check if this is our ICMP Echo Request
	if icmpHeader[0] != 8 { // ICMP Echo Request type
		return nil, false
	}

	// Extract ID and Sequence from original ICMP header
	origID := binary.BigEndian.Uint16(icmpHeader[4:6])
	origSeq := binary.BigEndian.Uint16(icmpHeader[6:8])

	if origID != p.identifier || origSeq != expectedSeq {
		return nil, false
	}

	return &Result{
		ResponseIP: peerIP,
		RTT:        rtt,
		ICMPType:   int(msg.Type.(ipv4.ICMPType)),
		ICMPCode:   int(msg.Code),
		Reached:    false,
		TTLExpired: true,
	}, true
}

// parseUnreachable parses a Destination Unreachable message.
func (p *ICMPProber) parseUnreachable(msg *icmp.Message, peerIP net.IP, rtt time.Duration, expectedSeq uint16) (*Result, bool) {
	body, ok := msg.Body.(*icmp.DstUnreach)
	if !ok {
		return nil, false
	}

	origData := body.Data
	if len(origData) < 28 {
		return nil, false
	}

	ipHeaderLen := int(origData[0]&0x0f) * 4
	if len(origData) < ipHeaderLen+8 {
		return nil, false
	}

	icmpHeader := origData[ipHeaderLen:]
	if icmpHeader[0] != 8 {
		return nil, false
	}

	origID := binary.BigEndian.Uint16(icmpHeader[4:6])
	origSeq := binary.BigEndian.Uint16(icmpHeader[6:8])

	if origID != p.identifier || origSeq != expectedSeq {
		return nil, false
	}

	return &Result{
		ResponseIP: peerIP,
		RTT:        rtt,
		ICMPType:   int(msg.Type.(ipv4.ICMPType)),
		ICMPCode:   int(msg.Code),
		Reached:    true, // We reached the destination but it's unreachable
		TTLExpired: false,
	}, true
}

// Name returns the probe method name.
func (p *ICMPProber) Name() string {
	if p.ipv6 {
		return "icmp6"
	}
	return "icmp"
}

// RequiresRoot returns true as ICMP raw sockets typically require elevated privileges.
func (p *ICMPProber) RequiresRoot() bool {
	return true
}

// Close releases resources held by the prober.
func (p *ICMPProber) Close() error {
	var err error
	if p.conn4 != nil {
		err = p.conn4.Close()
		p.conn4 = nil
	}
	if p.conn6 != nil {
		if e := p.conn6.Close(); e != nil && err == nil {
			err = e
		}
		p.conn6 = nil
	}
	return err
}

// Helper functions

func extractIP(addr net.Addr) net.IP {
	switch a := addr.(type) {
	case *net.IPAddr:
		return a.IP
	case *net.UDPAddr:
		return a.IP
	default:
		return nil
	}
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}
