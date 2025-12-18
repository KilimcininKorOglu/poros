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

// TCPProberConfig holds configuration for the TCP prober.
type TCPProberConfig struct {
	// Timeout is the maximum time to wait for a response
	Timeout time.Duration

	// Port is the destination port (default: 80)
	Port int

	// IPv6 enables IPv6 mode
	IPv6 bool
}

// DefaultTCPProberConfig returns a default TCP prober configuration.
func DefaultTCPProberConfig() TCPProberConfig {
	return TCPProberConfig{
		Timeout: 3 * time.Second,
		Port:    80,
		IPv6:    false,
	}
}

// TCPProber implements the Prober interface using TCP SYN packets.
// It sends TCP SYN packets and listens for:
// - ICMP Time Exceeded (intermediate hops)
// - TCP SYN-ACK or RST (destination reached)
type TCPProber struct {
	config   TCPProberConfig
	icmpConn *icmp.PacketConn
	rawConn  net.PacketConn
	localIP  net.IP
	localPort uint16
	sequence uint32
}

// NewTCPProber creates a new TCP SYN prober.
func NewTCPProber(config TCPProberConfig) (*TCPProber, error) {
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}
	if config.Port == 0 {
		config.Port = 80
	}

	// Create ICMP listener for Time Exceeded messages
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

	// Create raw socket for TCP
	var rawConn net.PacketConn
	if config.IPv6 {
		rawConn, err = net.ListenPacket("ip6:tcp", "::")
	} else {
		rawConn, err = net.ListenPacket("ip4:tcp", "0.0.0.0")
	}
	if err != nil {
		icmpConn.Close()
		return nil, fmt.Errorf("failed to create TCP raw socket: %w", err)
	}

	// Get local IP for source address in packets
	localIP := getOutboundIP(config.IPv6)

	return &TCPProber{
		config:    config,
		icmpConn:  icmpConn,
		rawConn:   rawConn,
		localIP:   localIP,
		localPort: uint16(30000 + (time.Now().UnixNano() % 10000)),
		sequence:  0,
	}, nil
}

// Probe sends a TCP SYN probe with the specified TTL.
func (p *TCPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*Result, error) {
	if ttl < 1 || ttl > 255 {
		return nil, ErrInvalidTTL
	}

	// Set TTL on raw socket
	if err := p.setTTL(ttl); err != nil {
		return nil, fmt.Errorf("failed to set TTL: %w", err)
	}

	// Generate unique sequence number
	seq := atomic.AddUint32(&p.sequence, 1)
	srcPort := p.localPort + uint16(seq%1000)

	// Build TCP SYN packet
	packet := p.buildSYNPacket(p.localIP, dest, srcPort, uint16(p.config.Port), seq)

	// Set read deadline
	deadline := time.Now().Add(p.config.Timeout)
	if err := p.icmpConn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set ICMP deadline: %w", err)
	}
	if err := p.rawConn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("failed to set TCP deadline: %w", err)
	}

	// Record send time
	sendTime := time.Now()

	// Send TCP SYN packet
	var destAddr net.Addr
	if p.config.IPv6 {
		destAddr = &net.IPAddr{IP: dest}
	} else {
		destAddr = &net.IPAddr{IP: dest}
	}

	if _, err := p.rawConn.WriteTo(packet, destAddr); err != nil {
		return nil, fmt.Errorf("failed to send TCP SYN: %w", err)
	}

	// Wait for response (ICMP or TCP)
	return p.receiveResponse(ctx, dest, srcPort, sendTime)
}

// setTTL sets the TTL on the raw TCP socket.
func (p *TCPProber) setTTL(ttl int) error {
	// Get the underlying file descriptor
	if conn, ok := p.rawConn.(*net.IPConn); ok {
		rawConn, err := conn.SyscallConn()
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
	return fmt.Errorf("unsupported connection type")
}

// buildSYNPacket creates a TCP SYN packet.
func (p *TCPProber) buildSYNPacket(src, dst net.IP, srcPort, dstPort uint16, seq uint32) []byte {
	// TCP header (20 bytes minimum)
	tcp := make([]byte, 20)

	// Source port
	binary.BigEndian.PutUint16(tcp[0:2], srcPort)
	// Destination port
	binary.BigEndian.PutUint16(tcp[2:4], dstPort)
	// Sequence number
	binary.BigEndian.PutUint32(tcp[4:8], seq)
	// Acknowledgment number (0 for SYN)
	binary.BigEndian.PutUint32(tcp[8:12], 0)
	// Data offset (5 = 20 bytes) + reserved + flags
	// Data offset: 5 (20 bytes / 4), SYN flag: 0x02
	tcp[12] = 0x50 // Data offset = 5
	tcp[13] = 0x02 // SYN flag
	// Window size
	binary.BigEndian.PutUint16(tcp[14:16], 65535)
	// Checksum (calculated below)
	binary.BigEndian.PutUint16(tcp[16:18], 0)
	// Urgent pointer
	binary.BigEndian.PutUint16(tcp[18:20], 0)

	// Calculate TCP checksum
	checksum := p.tcpChecksum(src, dst, tcp)
	binary.BigEndian.PutUint16(tcp[16:18], checksum)

	return tcp
}

// tcpChecksum calculates the TCP checksum including pseudo-header.
func (p *TCPProber) tcpChecksum(src, dst net.IP, tcpHeader []byte) uint16 {
	// Build pseudo-header
	var pseudoHeader []byte

	if p.config.IPv6 {
		// IPv6 pseudo-header
		pseudoHeader = make([]byte, 40)
		copy(pseudoHeader[0:16], src.To16())
		copy(pseudoHeader[16:32], dst.To16())
		binary.BigEndian.PutUint32(pseudoHeader[32:36], uint32(len(tcpHeader)))
		pseudoHeader[39] = 6 // TCP protocol
	} else {
		// IPv4 pseudo-header
		pseudoHeader = make([]byte, 12)
		copy(pseudoHeader[0:4], src.To4())
		copy(pseudoHeader[4:8], dst.To4())
		pseudoHeader[8] = 0
		pseudoHeader[9] = 6 // TCP protocol
		binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpHeader)))
	}

	// Combine pseudo-header and TCP header
	data := append(pseudoHeader, tcpHeader...)

	return Checksum(data)
}

// receiveResponse waits for ICMP or TCP response.
func (p *TCPProber) receiveResponse(ctx context.Context, dest net.IP, srcPort uint16, sendTime time.Time) (*Result, error) {
	icmpBuf := make([]byte, 1500)
	tcpBuf := make([]byte, 1500)

	// Create channels for responses
	icmpChan := make(chan *Result, 1)
	tcpChan := make(chan *Result, 1)
	errChan := make(chan error, 2)

	// Listen for ICMP responses
	go func() {
		for {
			n, peer, err := p.icmpConn.ReadFrom(icmpBuf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					errChan <- ErrTimeout
					return
				}
				errChan <- err
				return
			}

			rtt := time.Since(sendTime)
			result, ok := p.parseICMPResponse(icmpBuf[:n], dest, srcPort)
			if ok {
				result.RTT = rtt
				result.ResponseIP = parseIP(peer)
				icmpChan <- result
				return
			}
		}
	}()

	// Listen for TCP responses
	go func() {
		for {
			n, peer, err := p.rawConn.ReadFrom(tcpBuf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					return
				}
				return
			}

			rtt := time.Since(sendTime)
			result, ok := p.parseTCPResponse(tcpBuf[:n], dest, srcPort)
			if ok {
				result.RTT = rtt
				result.ResponseIP = parseIP(peer)
				tcpChan <- result
				return
			}
		}
	}()

	// Wait for first valid response
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-icmpChan:
		return result, nil
	case result := <-tcpChan:
		return result, nil
	case err := <-errChan:
		return nil, err
	}
}

// parseICMPResponse parses an ICMP response for our TCP probe.
func (p *TCPProber) parseICMPResponse(data []byte, dest net.IP, srcPort uint16) (*Result, bool) {
	var proto int
	if p.config.IPv6 {
		proto = 58
	} else {
		proto = 1
	}

	msg, err := icmp.ParseMessage(proto, data)
	if err != nil {
		return nil, false
	}

	result := &Result{}

	if p.config.IPv6 {
		switch msg.Type {
		case ipv6.ICMPTypeTimeExceeded:
			if body, ok := msg.Body.(*icmp.TimeExceeded); ok {
				if p.matchOriginalTCP(body.Data, dest, srcPort) {
					result.TTLExpired = true
					result.ICMPType = msg.Type.(ipv6.ICMPType).Protocol()
					result.ICMPCode = msg.Code
					return result, true
				}
			}
		case ipv6.ICMPTypeDestinationUnreachable:
			if body, ok := msg.Body.(*icmp.DstUnreach); ok {
				if p.matchOriginalTCP(body.Data, dest, srcPort) {
					result.Reached = true
					result.ICMPType = msg.Type.(ipv6.ICMPType).Protocol()
					result.ICMPCode = msg.Code
					return result, true
				}
			}
		}
	} else {
		switch msg.Type {
		case ipv4.ICMPTypeTimeExceeded:
			if body, ok := msg.Body.(*icmp.TimeExceeded); ok {
				if p.matchOriginalTCP(body.Data, dest, srcPort) {
					result.TTLExpired = true
					result.ICMPType = msg.Type.(ipv4.ICMPType).Protocol()
					result.ICMPCode = msg.Code
					return result, true
				}
			}
		case ipv4.ICMPTypeDestinationUnreachable:
			if body, ok := msg.Body.(*icmp.DstUnreach); ok {
				if p.matchOriginalTCP(body.Data, dest, srcPort) {
					result.Reached = true
					result.ICMPType = msg.Type.(ipv4.ICMPType).Protocol()
					result.ICMPCode = msg.Code
					return result, true
				}
			}
		}
	}

	return nil, false
}

// matchOriginalTCP checks if ICMP error contains our original TCP packet.
func (p *TCPProber) matchOriginalTCP(data []byte, dest net.IP, srcPort uint16) bool {
	if len(data) < 28 { // IP header + TCP header
		return false
	}

	// Skip IP header
	ihl := int(data[0]&0x0f) * 4
	if ihl < 20 || len(data) < ihl+8 {
		return false
	}

	tcpHeader := data[ihl:]

	// Check source port
	pktSrcPort := binary.BigEndian.Uint16(tcpHeader[0:2])
	if pktSrcPort != srcPort {
		return false
	}

	// Check destination port
	pktDstPort := binary.BigEndian.Uint16(tcpHeader[2:4])
	if int(pktDstPort) != p.config.Port {
		return false
	}

	// Check destination IP
	destIPInPacket := net.IP(data[16:20])
	if !destIPInPacket.Equal(dest) {
		return false
	}

	return true
}

// parseTCPResponse parses a TCP response (SYN-ACK or RST).
func (p *TCPProber) parseTCPResponse(data []byte, dest net.IP, srcPort uint16) (*Result, bool) {
	if len(data) < 20 {
		return nil, false
	}

	// TCP header fields
	pktSrcPort := binary.BigEndian.Uint16(data[0:2])
	pktDstPort := binary.BigEndian.Uint16(data[2:4])
	flags := data[13]

	// Check if this is a response to our probe
	if int(pktSrcPort) != p.config.Port || pktDstPort != srcPort {
		return nil, false
	}

	result := &Result{
		Reached: true,
	}

	// Check flags
	synAck := (flags & 0x12) == 0x12 // SYN + ACK
	rst := (flags & 0x04) == 0x04    // RST

	if synAck || rst {
		return result, true
	}

	return nil, false
}

// Name returns the probe method name.
func (p *TCPProber) Name() string {
	return "tcp"
}

// RequiresRoot returns true as TCP raw sockets require elevated privileges.
func (p *TCPProber) RequiresRoot() bool {
	return true
}

// Close releases resources held by the prober.
func (p *TCPProber) Close() error {
	var errs []error

	if p.icmpConn != nil {
		if err := p.icmpConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if p.rawConn != nil {
		if err := p.rawConn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// getOutboundIP gets the preferred outbound IP address.
func getOutboundIP(ipv6 bool) net.IP {
	var network, address string
	if ipv6 {
		network = "udp6"
		address = "[2001:4860:4860::8888]:80"
	} else {
		network = "udp4"
		address = "8.8.8.8:80"
	}

	conn, err := net.Dial(network, address)
	if err != nil {
		if ipv6 {
			return net.ParseIP("::")
		}
		return net.ParseIP("0.0.0.0")
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP
}
