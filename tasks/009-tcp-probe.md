# Feature 009: TCP SYN Probe Implementation

**Feature ID:** F009
**Feature Name:** TCP SYN Probe Implementation
**Priority:** P2 - HIGH
**Target Version:** v0.4.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

Implement TCP SYN (half-open) probe method that sends TCP SYN packets to common ports like 80 (HTTP) and 443 (HTTPS). This method is often effective when ICMP and UDP are filtered, as web traffic ports are typically allowed through firewalls.

TCP SYN probing requires careful connection management to avoid completing the TCP handshake and to properly clean up with RST packets.

## Goals
- Implement TCP SYN packet construction using raw sockets
- Handle SYN-ACK and RST responses
- Detect ICMP Time Exceeded for intermediate hops
- Support configurable destination ports
- Implement proper connection cleanup

## Success Criteria
- [ ] All tasks completed (T054-T060)
- [ ] TCP probe reaches destinations on port 80/443
- [ ] Handles firewall filtered responses
- [ ] No hanging connections left behind
- [ ] Performance comparable to ICMP probe

## Tasks

### T054: Implement TCP Packet Builder

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create TCP packet construction utilities including SYN flag setting, sequence number management, and TCP checksum calculation with pseudo-header.

#### Technical Details
```go
// internal/probe/tcp_packet.go
type TCPPacket struct {
    SrcPort    uint16
    DstPort    uint16
    SeqNum     uint32
    AckNum     uint32
    DataOffset uint8  // Header length in 32-bit words
    Flags      uint8
    Window     uint16
    Checksum   uint16
    UrgentPtr  uint16
    Options    []byte
    Payload    []byte
}

// TCP Flags
const (
    TCPFlagFIN = 0x01
    TCPFlagSYN = 0x02
    TCPFlagRST = 0x04
    TCPFlagPSH = 0x08
    TCPFlagACK = 0x10
    TCPFlagURG = 0x20
)

func (p *TCPPacket) Marshal(srcIP, dstIP net.IP) ([]byte, error) {
    headerLen := 20 + len(p.Options)
    if headerLen%4 != 0 {
        // Pad options to 32-bit boundary
        padding := 4 - (headerLen % 4)
        p.Options = append(p.Options, make([]byte, padding)...)
        headerLen += padding
    }
    
    p.DataOffset = uint8(headerLen / 4)
    
    buf := make([]byte, headerLen+len(p.Payload))
    
    binary.BigEndian.PutUint16(buf[0:2], p.SrcPort)
    binary.BigEndian.PutUint16(buf[2:4], p.DstPort)
    binary.BigEndian.PutUint32(buf[4:8], p.SeqNum)
    binary.BigEndian.PutUint32(buf[8:12], p.AckNum)
    buf[12] = p.DataOffset << 4
    buf[13] = p.Flags
    binary.BigEndian.PutUint16(buf[14:16], p.Window)
    // Checksum at bytes 16-17 (set after calculation)
    binary.BigEndian.PutUint16(buf[18:20], p.UrgentPtr)
    
    copy(buf[20:], p.Options)
    copy(buf[headerLen:], p.Payload)
    
    // Calculate checksum with pseudo-header
    p.Checksum = tcpChecksum(buf, srcIP, dstIP)
    binary.BigEndian.PutUint16(buf[16:18], p.Checksum)
    
    return buf, nil
}

func tcpChecksum(tcpData []byte, srcIP, dstIP net.IP) uint16 {
    // Build pseudo-header
    src := srcIP.To4()
    dst := dstIP.To4()
    
    pseudoHeader := make([]byte, 12)
    copy(pseudoHeader[0:4], src)
    copy(pseudoHeader[4:8], dst)
    pseudoHeader[8] = 0
    pseudoHeader[9] = 6 // TCP protocol
    binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpData)))
    
    // Combine and calculate
    data := append(pseudoHeader, tcpData...)
    return internetChecksum(data)
}

func NewTCPSYNPacket(srcPort, dstPort uint16, seqNum uint32) *TCPPacket {
    return &TCPPacket{
        SrcPort:    srcPort,
        DstPort:    dstPort,
        SeqNum:     seqNum,
        Flags:      TCPFlagSYN,
        Window:     65535,
        DataOffset: 5,
    }
}
```

#### Files to Touch
- `internal/probe/tcp_packet.go` (new)
- `internal/probe/tcp_packet_test.go` (new)
- `internal/probe/checksum.go` (update - add internetChecksum)

#### Dependencies
- T007: Checksum utilities

#### Success Criteria
- [ ] TCP packets marshal correctly
- [ ] Checksum calculation is correct
- [ ] SYN flag is set properly
- [ ] Packet captures show valid TCP

---

### T055: Create TCP Raw Socket

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement raw socket handling for TCP with IP_HDRINCL option to allow full packet control including TTL manipulation.

#### Technical Details
```go
// internal/network/tcp_socket.go
type TCPRawSocket struct {
    fd       int
    srcIP    net.IP
    srcPort  uint16
    icmpConn *RawSocket
}

func NewTCPRawSocket(config TCPSocketConfig) (*TCPRawSocket, error) {
    // Create raw TCP socket with IP_HDRINCL
    fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_RAW, syscall.IPPROTO_TCP)
    if err != nil {
        if errors.Is(err, syscall.EPERM) {
            return nil, ErrPermissionDenied
        }
        return nil, err
    }
    
    // Include IP header (we'll build it ourselves)
    if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 1); err != nil {
        syscall.Close(fd)
        return nil, err
    }
    
    // Create ICMP socket for receiving Time Exceeded
    icmpConn, err := NewRawICMPSocket(false)
    if err != nil {
        syscall.Close(fd)
        return nil, err
    }
    
    // Allocate ephemeral source port
    srcPort := allocateEphemeralPort()
    
    return &TCPRawSocket{
        fd:       fd,
        srcIP:    config.SourceIP,
        srcPort:  srcPort,
        icmpConn: icmpConn,
    }, nil
}

func (s *TCPRawSocket) SendSYN(dstIP net.IP, dstPort uint16, ttl int, seqNum uint32) error {
    // Build TCP packet
    tcp := NewTCPSYNPacket(s.srcPort, dstPort, seqNum)
    tcpData, err := tcp.Marshal(s.srcIP, dstIP)
    if err != nil {
        return err
    }
    
    // Build IP header
    ip := NewIPv4Header(s.srcIP, dstIP, syscall.IPPROTO_TCP, ttl, tcpData)
    packet := ip.Marshal()
    packet = append(packet, tcpData...)
    
    // Send
    addr := syscall.SockaddrInet4{Port: int(dstPort)}
    copy(addr.Addr[:], dstIP.To4())
    
    return syscall.Sendto(s.fd, packet, 0, &addr)
}

func allocateEphemeralPort() uint16 {
    // Use random high port (49152-65535)
    return uint16(49152 + rand.Intn(16383))
}
```

#### Files to Touch
- `internal/network/tcp_socket.go` (new)
- `internal/network/tcp_socket_linux.go` (new)
- `internal/network/ip_header.go` (new)

#### Dependencies
- T008: Raw socket base implementation

#### Success Criteria
- [ ] Can create TCP raw socket
- [ ] IP_HDRINCL works correctly
- [ ] TTL manipulation in IP header
- [ ] Source port allocation works

---

### T056: Implement IP Header Builder

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Create IP header construction for use with IP_HDRINCL sockets where we need to build the full IP packet.

#### Technical Details
```go
// internal/network/ip_header.go
type IPv4Header struct {
    Version    uint8
    IHL        uint8
    TOS        uint8
    TotalLen   uint16
    ID         uint16
    Flags      uint8
    FragOffset uint16
    TTL        uint8
    Protocol   uint8
    Checksum   uint16
    SrcIP      net.IP
    DstIP      net.IP
}

func NewIPv4Header(srcIP, dstIP net.IP, protocol uint8, ttl int, payload []byte) *IPv4Header {
    return &IPv4Header{
        Version:  4,
        IHL:      5, // No options = 20 bytes
        TOS:      0,
        TotalLen: uint16(20 + len(payload)),
        ID:       uint16(rand.Uint32()),
        Flags:    0x40, // Don't fragment
        TTL:      uint8(ttl),
        Protocol: protocol,
        SrcIP:    srcIP,
        DstIP:    dstIP,
    }
}

func (h *IPv4Header) Marshal() []byte {
    buf := make([]byte, 20)
    
    buf[0] = (h.Version << 4) | h.IHL
    buf[1] = h.TOS
    binary.BigEndian.PutUint16(buf[2:4], h.TotalLen)
    binary.BigEndian.PutUint16(buf[4:6], h.ID)
    binary.BigEndian.PutUint16(buf[6:8], (uint16(h.Flags)<<13)|h.FragOffset)
    buf[8] = h.TTL
    buf[9] = h.Protocol
    // Checksum at bytes 10-11
    copy(buf[12:16], h.SrcIP.To4())
    copy(buf[16:20], h.DstIP.To4())
    
    // Calculate header checksum
    h.Checksum = internetChecksum(buf)
    binary.BigEndian.PutUint16(buf[10:12], h.Checksum)
    
    return buf
}
```

#### Files to Touch
- `internal/network/ip_header.go` (new)
- `internal/network/ip_header_test.go` (new)

#### Dependencies
- T007: Checksum calculation

#### Success Criteria
- [ ] Valid IPv4 headers generated
- [ ] TTL is correctly set
- [ ] Checksum is correct
- [ ] Works with Wireshark capture

---

### T057: Implement TCP Probe Send/Receive

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Implement the core TCP probe logic: send SYN, receive responses (SYN-ACK, RST, or ICMP Time Exceeded), calculate RTT.

#### Technical Details
```go
// internal/probe/tcp.go
type TCPProber struct {
    socket   *network.TCPRawSocket
    dstPort  uint16
    timeout  time.Duration
    sequence uint32
}

type TCPProberConfig struct {
    Timeout  time.Duration
    DstPort  uint16 // Default 80 or 443
    SourceIP net.IP
}

func NewTCPProber(config TCPProberConfig) (*TCPProber, error) {
    if config.DstPort == 0 {
        config.DstPort = 80
    }
    
    socket, err := network.NewTCPRawSocket(network.TCPSocketConfig{
        SourceIP: config.SourceIP,
    })
    if err != nil {
        return nil, err
    }
    
    return &TCPProber{
        socket:   socket,
        dstPort:  config.DstPort,
        timeout:  config.Timeout,
        sequence: rand.Uint32(),
    }, nil
}

func (p *TCPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error) {
    seqNum := atomic.AddUint32(&p.sequence, 1)
    
    sendTime := time.Now()
    
    // Send TCP SYN
    if err := p.socket.SendSYN(dest, p.dstPort, ttl, seqNum); err != nil {
        return nil, err
    }
    
    // Wait for response
    return p.waitForResponse(ctx, sendTime, dest, ttl, seqNum)
}

func (p *TCPProber) waitForResponse(ctx context.Context, sendTime time.Time,
    dest net.IP, ttl int, seqNum uint32) (*ProbeResult, error) {
    
    deadline := sendTime.Add(p.timeout)
    
    // Listen for both TCP responses and ICMP
    tcpCh := make(chan *ProbeResult, 1)
    icmpCh := make(chan *ProbeResult, 1)
    
    go p.listenTCP(ctx, deadline, dest, seqNum, sendTime, tcpCh)
    go p.listenICMP(ctx, deadline, ttl, seqNum, sendTime, icmpCh)
    
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case result := <-tcpCh:
        return result, nil
    case result := <-icmpCh:
        return result, nil
    case <-time.After(p.timeout):
        return &ProbeResult{RTT: p.timeout}, ErrTimeout
    }
}

func (p *TCPProber) listenTCP(ctx context.Context, deadline time.Time, 
    dest net.IP, seqNum uint32, sendTime time.Time, ch chan<- *ProbeResult) {
    
    // Listen for SYN-ACK or RST from destination
    // ...
    // On SYN-ACK: destination reached, send RST to clean up
    // On RST: destination reached (port closed)
}

func (p *TCPProber) listenICMP(ctx context.Context, deadline time.Time,
    ttl int, seqNum uint32, sendTime time.Time, ch chan<- *ProbeResult) {
    
    // Listen for ICMP Time Exceeded
    // Parse embedded TCP header to match our probe
}
```

#### Files to Touch
- `internal/probe/tcp.go` (new)
- `internal/probe/tcp_test.go` (new)

#### Dependencies
- T054: TCP packet builder
- T055: TCP raw socket
- T056: IP header builder

#### Success Criteria
- [ ] Sends valid TCP SYN
- [ ] Detects intermediate hops via ICMP
- [ ] Detects destination via SYN-ACK/RST
- [ ] Accurate RTT measurement

---

### T058: Implement Connection Cleanup

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Ensure TCP probes don't leave half-open connections by sending RST when SYN-ACK is received.

#### Technical Details
```go
// internal/probe/tcp.go (update)
func (p *TCPProber) sendRST(dstIP net.IP, dstPort uint16, ackNum uint32) error {
    rst := &TCPPacket{
        SrcPort: p.socket.SrcPort(),
        DstPort: dstPort,
        SeqNum:  ackNum, // Use their ACK as our SEQ
        AckNum:  0,
        Flags:   TCPFlagRST,
        Window:  0,
    }
    
    return p.socket.SendPacket(rst, dstIP)
}

func (p *TCPProber) handleSYNACK(packet []byte, from net.IP, sendTime time.Time) *ProbeResult {
    // Parse SYN-ACK
    seqNum := binary.BigEndian.Uint32(packet[4:8])
    ackNum := binary.BigEndian.Uint32(packet[8:12])
    
    // Send RST to clean up
    p.sendRST(from, p.dstPort, ackNum)
    
    return &ProbeResult{
        ResponseIP: from,
        RTT:        time.Since(sendTime),
        Reached:    true,
        TCPFlags:   TCPFlagSYN | TCPFlagACK,
    }
}
```

#### Files to Touch
- `internal/probe/tcp.go` (update)

#### Dependencies
- T057: TCP probe implementation

#### Success Criteria
- [ ] RST sent after SYN-ACK
- [ ] No lingering connections
- [ ] Verified with netstat

---

### T059: Register TCP Prober with Tracer

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Integrate TCP prober with the tracer system and add CLI flags.

#### Technical Details
```go
// internal/trace/tracer.go (update)
func NewTracer(config *TracerConfig) (*Tracer, error) {
    var prober probe.Prober
    var err error
    
    switch config.ProbeMethod {
    case probe.ProbeICMP:
        prober, err = probe.NewICMPProber(...)
    case probe.ProbeUDP:
        prober, err = probe.NewUDPProber(...)
    case probe.ProbeTCP:
        prober, err = probe.NewTCPProber(probe.TCPProberConfig{
            Timeout:  config.Timeout,
            DstPort:  uint16(config.DestPort),
            SourceIP: config.SourceIP,
        })
    }
    
    // ...
}

// cmd/poros/root.go (update)
func init() {
    rootCmd.Flags().BoolP("tcp", "T", false, "Use TCP SYN probes (default port 80)")
    rootCmd.Flags().Int("tcp-port", 80, "TCP destination port (80, 443, etc.)")
}
```

#### Files to Touch
- `internal/trace/tracer.go` (update)
- `cmd/poros/root.go` (update)

#### Dependencies
- T057: TCP prober complete

#### Success Criteria
- [ ] `-T` selects TCP probe
- [ ] `--tcp-port` sets destination
- [ ] Works with concurrent tracer

---

### T060: Add TCP Probe Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create unit and integration tests for TCP probing.

#### Technical Details
```go
// internal/probe/tcp_test.go
func TestTCPPacket_Marshal(t *testing.T) {
    pkt := NewTCPSYNPacket(12345, 80, 100)
    data, err := pkt.Marshal(
        net.ParseIP("192.168.1.100"),
        net.ParseIP("1.2.3.4"),
    )
    require.NoError(t, err)
    
    // Verify SYN flag
    flags := data[13]
    assert.Equal(t, uint8(TCPFlagSYN), flags)
    
    // Verify ports
    srcPort := binary.BigEndian.Uint16(data[0:2])
    dstPort := binary.BigEndian.Uint16(data[2:4])
    assert.Equal(t, uint16(12345), srcPort)
    assert.Equal(t, uint16(80), dstPort)
}

// internal/probe/tcp_integration_test.go
//go:build integration

func TestTCPProbe_Web(t *testing.T) {
    if os.Getuid() != 0 {
        t.Skip("requires root")
    }
    
    prober, err := NewTCPProber(TCPProberConfig{
        Timeout: 5 * time.Second,
        DstPort: 80,
    })
    require.NoError(t, err)
    defer prober.Close()
    
    // Test against a web server
    result, err := prober.Probe(context.Background(),
        net.ParseIP("1.1.1.1"), 64)
    require.NoError(t, err)
    
    assert.True(t, result.Reached)
}
```

#### Files to Touch
- `internal/probe/tcp_test.go` (new)
- `internal/probe/tcp_integration_test.go` (new)

#### Dependencies
- T059: TCP prober integrated

#### Success Criteria
- [ ] Unit tests pass
- [ ] Integration tests work with root
- [ ] Packet capture shows valid SYN/RST

---

## Performance Targets
- TCP probe overhead: < 200μs
- Connection cleanup: < 1ms
- Memory per probe: < 2KB

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Firewall blocking SYN | Medium | Medium | Offer multiple port options |
| IDS/IPS alerts | Medium | Low | Document as diagnostic tool |
| Connection table exhaustion | Low | Medium | Proper RST cleanup |
| Platform differences | High | Medium | Careful testing on all platforms |

## Notes
- TCP SYN scanning can trigger security alerts
- Port 443 often more reliable than 80 (HTTPS)
- Consider adding --tcp-ports flag for multiple ports
- Windows implementation may differ significantly
