# Feature 005: UDP Probe Implementation

**Feature ID:** F005
**Feature Name:** UDP Probe Implementation
**Priority:** P2 - HIGH
**Target Version:** v0.2.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Implement the UDP probe method, which is the default method for Unix traceroute. UDP probes send packets to high-numbered ports (typically starting at 33434) and rely on ICMP Port Unreachable responses to detect the destination and ICMP Time Exceeded for intermediate hops.

UDP probing can be useful when ICMP is filtered, though UDP itself may also be filtered. The port incrementing behavior helps identify the destination even when multiple responses arrive.

## Goals
- Implement fully functional UDP probe method
- Support configurable port ranges (default 33434-33534)
- Handle ICMP responses for UDP probes
- Support both IPv4 and IPv6
- Implement high-port fallback option

## Success Criteria
- [ ] All tasks completed (T026-T031)
- [ ] UDP probe successfully traces to targets
- [ ] Port incrementing works correctly
- [ ] Handles ICMP Port Unreachable for destination detection
- [ ] Works when ICMP Echo is blocked but UDP responses work

## Tasks

### T026: Implement UDP Packet Builder

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Create UDP packet construction utilities. Unlike ICMP, UDP packets need destination port handling and may require specific payload for identification.

#### Technical Details
```go
// internal/probe/udp_packet.go
type UDPPacket struct {
    SrcPort  uint16
    DstPort  uint16
    Length   uint16
    Checksum uint16
    Payload  []byte
}

func (p *UDPPacket) Marshal() []byte {
    buf := make([]byte, 8+len(p.Payload))
    binary.BigEndian.PutUint16(buf[0:2], p.SrcPort)
    binary.BigEndian.PutUint16(buf[2:4], p.DstPort)
    p.Length = uint16(8 + len(p.Payload))
    binary.BigEndian.PutUint16(buf[4:6], p.Length)
    // Checksum calculated with pseudo-header
    binary.BigEndian.PutUint16(buf[6:8], p.Checksum)
    copy(buf[8:], p.Payload)
    return buf
}

// UDPProbePayload creates a traceroute-identifiable payload
func UDPProbePayload(identifier uint16, sequence uint16) []byte {
    payload := make([]byte, 12)
    // Include timestamp for RTT calculation
    binary.BigEndian.PutUint64(payload[0:8], uint64(time.Now().UnixNano()))
    binary.BigEndian.PutUint16(payload[8:10], identifier)
    binary.BigEndian.PutUint16(payload[10:12], sequence)
    return payload
}
```

#### Files to Touch
- `internal/probe/udp_packet.go` (new)
- `internal/probe/udp_packet_test.go` (new)

#### Dependencies
- T003: Prober interface

#### Success Criteria
- [ ] UDP packets marshal correctly
- [ ] Payload contains identification info
- [ ] Checksum calculation works

---

### T027: Implement UDP Socket Handling

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create UDP socket handling for sending probes. This requires managing both the outgoing UDP socket and listening for incoming ICMP responses.

#### Technical Details
```go
// internal/network/udp_socket.go
type UDPSocket struct {
    conn     *net.UDPConn
    icmpConn *network.RawSocket  // For receiving ICMP responses
    localIP  net.IP
    localPort uint16
}

func NewUDPSocket(config UDPSocketConfig) (*UDPSocket, error) {
    // Create UDP socket for sending
    laddr := &net.UDPAddr{
        IP:   config.SourceIP,
        Port: config.SourcePort,
    }
    
    conn, err := net.ListenUDP("udp4", laddr)
    if err != nil {
        return nil, err
    }
    
    // Create ICMP raw socket for receiving responses
    icmpConn, err := NewRawICMPSocket(false)
    if err != nil {
        conn.Close()
        return nil, err
    }
    
    return &UDPSocket{
        conn:      conn,
        icmpConn:  icmpConn,
        localIP:   laddr.IP,
        localPort: uint16(laddr.Port),
    }, nil
}

func (s *UDPSocket) SetTTL(ttl int) error {
    // Set TTL on the UDP socket
    rawConn, err := s.conn.SyscallConn()
    if err != nil {
        return err
    }
    
    return rawConn.Control(func(fd uintptr) {
        syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, 
            syscall.IP_TTL, ttl)
    })
}
```

#### Files to Touch
- `internal/network/udp_socket.go` (new)
- `internal/network/udp_socket_linux.go` (new - platform-specific)
- `internal/network/udp_socket_test.go` (new)

#### Dependencies
- T008: Raw socket implementation (for ICMP receive)

#### Success Criteria
- [ ] Can create UDP socket
- [ ] Can set TTL on UDP packets
- [ ] Can receive ICMP responses
- [ ] Proper cleanup on close

---

### T028: Implement UDP Probe Send/Receive

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description
Implement the core UDP probe logic: send UDP packet to target port, receive ICMP Time Exceeded or Port Unreachable, calculate RTT.

#### Technical Details
```go
// internal/probe/udp.go
type UDPProber struct {
    socket     *network.UDPSocket
    identifier uint16
    sequence   uint32
    basePort   int
    timeout    time.Duration
}

type UDPProberConfig struct {
    Timeout   time.Duration
    BasePort  int  // Starting destination port (default 33434)
    SourceIP  net.IP
    IPv6      bool
}

func NewUDPProber(config UDPProberConfig) (*UDPProber, error) {
    if config.BasePort == 0 {
        config.BasePort = 33434
    }
    
    socket, err := network.NewUDPSocket(network.UDPSocketConfig{
        SourceIP: config.SourceIP,
    })
    if err != nil {
        return nil, err
    }
    
    return &UDPProber{
        socket:     socket,
        identifier: uint16(os.Getpid() & 0xffff),
        basePort:   config.BasePort,
        timeout:    config.Timeout,
    }, nil
}

func (p *UDPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error) {
    // Set TTL
    if err := p.socket.SetTTL(ttl); err != nil {
        return nil, err
    }
    
    // Calculate destination port (base + ttl to help identify responses)
    seq := atomic.AddUint32(&p.sequence, 1)
    dstPort := p.basePort + int(ttl)
    
    // Build and send UDP packet
    payload := UDPProbePayload(p.identifier, uint16(seq))
    
    sendTime := time.Now()
    destAddr := &net.UDPAddr{IP: dest, Port: dstPort}
    
    if _, err := p.socket.WriteTo(payload, destAddr); err != nil {
        return nil, err
    }
    
    // Wait for ICMP response
    return p.waitForResponse(ctx, sendTime, dest, ttl, uint16(seq))
}

func (p *UDPProber) waitForResponse(ctx context.Context, sendTime time.Time, 
    dest net.IP, ttl int, seq uint16) (*ProbeResult, error) {
    
    deadline := sendTime.Add(p.timeout)
    p.socket.SetReadDeadline(deadline)
    
    buf := make([]byte, 1500)
    for {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
        }
        
        n, from, err := p.socket.ReadICMP(buf)
        if err != nil {
            if isTimeout(err) {
                return &ProbeResult{RTT: p.timeout}, ErrTimeout
            }
            return nil, err
        }
        
        result, ok := p.parseICMPResponse(buf[:n], from, sendTime, ttl, seq)
        if ok {
            return result, nil
        }
    }
}
```

#### Files to Touch
- `internal/probe/udp.go` (new)
- `internal/probe/udp_test.go` (new)

#### Dependencies
- T026: UDP packet builder
- T027: UDP socket handling

#### Success Criteria
- [ ] Sends UDP probe correctly
- [ ] Receives ICMP Time Exceeded
- [ ] Receives ICMP Port Unreachable
- [ ] RTT calculation is accurate

---

### T029: Implement UDP Response Parser

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Parse ICMP responses to UDP probes. The ICMP message contains the original UDP header in its payload, which we use to match responses to sent probes.

#### Technical Details
```go
// internal/probe/udp.go
func (p *UDPProber) parseICMPResponse(data []byte, from net.Addr, 
    sendTime time.Time, expectedTTL int, expectedSeq uint16) (*ProbeResult, bool) {
    
    // Parse IP header
    ipHeaderLen := int(data[0]&0x0f) * 4
    icmpData := data[ipHeaderLen:]
    
    icmpType := icmpData[0]
    icmpCode := icmpData[1]
    
    rtt := time.Since(sendTime)
    fromIP := netAddrToIP(from)
    
    switch icmpType {
    case ICMPv4TimeExceeded:
        // Extract original IP + UDP header from ICMP payload
        // ICMP header is 8 bytes, then original IP header + 8 bytes of UDP
        origIPHeader := icmpData[8:]
        origIPHeaderLen := int(origIPHeader[0]&0x0f) * 4
        origUDP := origIPHeader[origIPHeaderLen:]
        
        // Check if this is our UDP packet
        origSrcPort := binary.BigEndian.Uint16(origUDP[0:2])
        origDstPort := binary.BigEndian.Uint16(origUDP[2:4])
        
        expectedDstPort := uint16(p.basePort + expectedTTL)
        
        if origSrcPort == p.socket.LocalPort() && origDstPort == expectedDstPort {
            return &ProbeResult{
                ResponseIP: fromIP,
                RTT:        rtt,
                ICMPType:   int(icmpType),
                ICMPCode:   int(icmpCode),
                Reached:    false,
            }, true
        }
        
    case ICMPv4Unreachable:
        if icmpCode == 3 { // Port Unreachable = destination reached
            // Same extraction logic
            // ...
            return &ProbeResult{
                ResponseIP: fromIP,
                RTT:        rtt,
                ICMPType:   int(icmpType),
                ICMPCode:   int(icmpCode),
                Reached:    true,  // Port unreachable means we reached dest
            }, true
        }
    }
    
    return nil, false
}
```

#### Files to Touch
- `internal/probe/udp.go` (update)
- `internal/probe/udp_parser.go` (new - if splitting)

#### Dependencies
- T028: UDP probe send/receive

#### Success Criteria
- [ ] Correctly identifies Time Exceeded
- [ ] Correctly identifies Port Unreachable as destination
- [ ] Matches response to correct probe
- [ ] Handles various ICMP codes

---

### T030: Register UDP Prober with Tracer

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Integrate the UDP prober with the tracer system. Add UDP as a selectable probe method with appropriate CLI flags.

#### Technical Details
```go
// internal/trace/tracer.go (update)
func NewTracer(config *TracerConfig) (*Tracer, error) {
    var prober probe.Prober
    var err error
    
    switch config.ProbeMethod {
    case probe.ProbeICMP:
        prober, err = probe.NewICMPProber(probe.ICMPProberConfig{
            Timeout: config.Timeout,
            IPv6:    config.IPv6,
        })
    case probe.ProbeUDP:
        prober, err = probe.NewUDPProber(probe.UDPProberConfig{
            Timeout:  config.Timeout,
            BasePort: config.DestPort,
            SourceIP: config.SourceIP,
            IPv6:     config.IPv6,
        })
    default:
        return nil, fmt.Errorf("unsupported probe method: %v", config.ProbeMethod)
    }
    
    if err != nil {
        return nil, err
    }
    
    return &Tracer{
        config: config,
        prober: prober,
    }, nil
}

// cmd/poros/root.go (update)
func init() {
    rootCmd.Flags().BoolP("udp", "U", false, "Use UDP probes (default port 33434)")
    rootCmd.Flags().IntP("port", "p", 33434, "Destination port for UDP/TCP probes")
}
```

#### Files to Touch
- `internal/trace/tracer.go` (update)
- `cmd/poros/root.go` (update)
- `cmd/poros/flags.go` (update)

#### Dependencies
- T028: UDP prober implementation
- T005: CLI framework

#### Success Criteria
- [ ] `-U` flag selects UDP probe
- [ ] `--port` configures base port
- [ ] UDP tracer works end-to-end
- [ ] Error handling for permission issues

---

### T031: Add UDP Probe Integration Test

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create integration tests for UDP probing. Test against real targets and verify correct behavior.

#### Technical Details
```go
// internal/probe/udp_integration_test.go
//go:build integration

func TestUDPProbe_Basic(t *testing.T) {
    if os.Getuid() != 0 {
        t.Skip("requires root")
    }
    
    prober, err := NewUDPProber(UDPProberConfig{
        Timeout:  2 * time.Second,
        BasePort: 33434,
    })
    require.NoError(t, err)
    defer prober.Close()
    
    // Test against a known target
    result, err := prober.Probe(context.Background(), 
        net.ParseIP("8.8.8.8"), 30)
    require.NoError(t, err)
    
    assert.True(t, result.Reached || result.RTT > 0)
}

func TestUDPTrace_Complete(t *testing.T) {
    // Test full UDP trace
    tracer, err := NewTracer(&TracerConfig{
        ProbeMethod: probe.ProbeUDP,
        MaxHops:     30,
        ProbeCount:  3,
        Timeout:     3 * time.Second,
    })
    require.NoError(t, err)
    defer tracer.Close()
    
    result, err := tracer.Trace(context.Background(), "google.com")
    require.NoError(t, err)
    
    assert.True(t, len(result.Hops) > 0)
}
```

#### Files to Touch
- `internal/probe/udp_integration_test.go` (new)
- `scripts/test-udp.sh` (new)

#### Dependencies
- T030: UDP prober registered

#### Success Criteria
- [ ] Tests pass with root
- [ ] Tests skip without root
- [ ] Compares well with system traceroute -U

---

## Performance Targets
- UDP probe creation: < 1ms
- Single UDP probe: < 100μs overhead
- Response matching: < 10μs

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| UDP filtering | High | Medium | Document as fallback, suggest ICMP |
| Port conflicts | Low | Low | Use random source port |
| Response matching errors | Medium | Medium | Thorough testing of edge cases |

## Notes
- UDP probing is the Unix default but ICMP often works better
- High-numbered source port can help with some NAT scenarios
- Consider adding --high-port option for firewall bypass
- Paris traceroute will build on this with fixed source port
