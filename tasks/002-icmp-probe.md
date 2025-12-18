# Feature 002: ICMP Probe Implementation

**Feature ID:** F002
**Feature Name:** ICMP Probe Implementation
**Priority:** P1 - CRITICAL
**Target Version:** v0.1.0
**Estimated Duration:** 1.5 weeks
**Status:** NOT_STARTED

## Overview

Implement the ICMP Echo probe method, which is the default and most widely used probe method for traceroute. This involves creating raw sockets, building ICMP packets with proper checksums, sending probes with incrementing TTL values, and parsing ICMP Time Exceeded responses.

This feature is critical as ICMP probing is the foundation upon which other probe methods will be built. It requires platform-specific socket handling for Linux, macOS, and Windows.

## Goals
- Implement fully functional ICMP Echo Request probe
- Handle ICMP Time Exceeded (Type 11) responses correctly
- Calculate accurate RTT measurements
- Support both IPv4 and IPv6 ICMP
- Ensure cross-platform compatibility (Linux first, then macOS/Windows)

## Success Criteria
- [ ] All tasks completed (T007-T013)
- [ ] ICMP probe successfully traces to google.com
- [ ] RTT measurements are accurate (within 1ms of system traceroute)
- [ ] Handles timeout correctly
- [ ] Works on Linux with root/CAP_NET_RAW
- [ ] Unit tests for checksum calculation

## Tasks

### T007: Implement ICMP Checksum Calculation

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Implement the Internet Checksum algorithm (RFC 1071) required for ICMP packet construction. This is a fundamental building block for all ICMP operations.

#### Technical Details
```go
// internal/probe/checksum.go
func ICMPChecksum(data []byte) uint16 {
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
    return ^uint16(sum)
}
```

#### Files to Touch
- `internal/probe/checksum.go` (new)
- `internal/probe/checksum_test.go` (new)

#### Dependencies
- T003: Prober interface defined

#### Success Criteria
- [ ] Checksum matches known test vectors
- [ ] Handles odd-length data
- [ ] Unit tests pass
- [ ] Benchmark shows acceptable performance

---

### T008: Create Platform-Specific Raw Socket (Linux)

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement Linux-specific raw socket creation for ICMP using `syscall` or `golang.org/x/sys/unix`. Handle privilege requirements and socket options.

#### Technical Details
```go
// internal/network/socket_linux.go
//go:build linux

func NewRawICMPSocket(ipv6 bool) (*RawSocket, error) {
    var domain, proto int
    if ipv6 {
        domain = syscall.AF_INET6
        proto = syscall.IPPROTO_ICMPV6
    } else {
        domain = syscall.AF_INET
        proto = syscall.IPPROTO_ICMP
    }
    
    fd, err := syscall.Socket(domain, syscall.SOCK_RAW, proto)
    if err != nil {
        if errors.Is(err, syscall.EPERM) {
            return nil, ErrPermissionDenied
        }
        return nil, err
    }
    
    // Set socket options
    syscall.SetsockoptInt(fd, syscall.IPPROTO_IP, syscall.IP_HDRINCL, 0)
    
    return &RawSocket{fd: fd, ipv6: ipv6}, nil
}
```

#### Files to Touch
- `internal/network/socket.go` (new) - interface definition
- `internal/network/socket_linux.go` (new)
- `internal/network/errors.go` (new)

#### Dependencies
- T001: Project structure
- T006: golang.org/x/sys dependency

#### Success Criteria
- [ ] Can create ICMP raw socket with root
- [ ] Proper error on permission denied
- [ ] IPv4 and IPv6 support
- [ ] Socket can be closed properly

---

### T009: Implement ICMP Packet Builder

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create ICMP Echo Request packet builder that constructs properly formatted ICMP packets with configurable type, code, identifier, sequence number, and payload.

#### Technical Details
```go
// internal/probe/icmp_packet.go
type ICMPPacket struct {
    Type       uint8
    Code       uint8
    Checksum   uint16
    Identifier uint16
    Sequence   uint16
    Payload    []byte
}

func (p *ICMPPacket) Marshal() ([]byte, error) {
    buf := make([]byte, 8+len(p.Payload))
    buf[0] = p.Type
    buf[1] = p.Code
    // Checksum at bytes 2-3 (initially 0)
    binary.BigEndian.PutUint16(buf[4:], p.Identifier)
    binary.BigEndian.PutUint16(buf[6:], p.Sequence)
    copy(buf[8:], p.Payload)
    
    // Calculate and set checksum
    p.Checksum = ICMPChecksum(buf)
    binary.BigEndian.PutUint16(buf[2:], p.Checksum)
    
    return buf, nil
}

// ICMP Types
const (
    ICMPv4EchoRequest   = 8
    ICMPv4EchoReply     = 0
    ICMPv4TimeExceeded  = 11
    ICMPv4Unreachable   = 3
)
```

#### Files to Touch
- `internal/probe/icmp_packet.go` (new)
- `internal/probe/icmp_packet_test.go` (new)
- `internal/probe/icmp_types.go` (new)

#### Dependencies
- T007: Checksum implementation

#### Success Criteria
- [ ] Packets marshal correctly
- [ ] Checksum is valid
- [ ] Can unmarshal received packets
- [ ] ICMPv6 packets supported

---

### T010: Implement TTL Manipulation

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Implement the ability to set the IP TTL (Time To Live) field on outgoing packets. This is essential for traceroute functionality - each hop is discovered by incrementing the TTL.

#### Technical Details
```go
// internal/network/socket_linux.go
func (s *RawSocket) SetTTL(ttl int) error {
    if s.ipv6 {
        return syscall.SetsockoptInt(s.fd, syscall.IPPROTO_IPV6, 
            syscall.IPV6_UNICAST_HOPS, ttl)
    }
    return syscall.SetsockoptInt(s.fd, syscall.IPPROTO_IP, 
        syscall.IP_TTL, ttl)
}

// Alternative using golang.org/x/net/ipv4
func (s *RawSocket) SetTTLv4(conn *ipv4.PacketConn, ttl int) error {
    return conn.SetTTL(ttl)
}
```

#### Files to Touch
- `internal/network/socket_linux.go` (update)
- `internal/network/socket.go` (update interface)

#### Dependencies
- T008: Raw socket implementation

#### Success Criteria
- [ ] Can set TTL from 1-255
- [ ] TTL change affects outgoing packets
- [ ] Works for both IPv4 and IPv6

---

### T011: Implement ICMP Probe Send/Receive

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Implement the core ICMP probe logic: send an Echo Request with specific TTL, receive the ICMP response (Time Exceeded or Echo Reply), and calculate RTT.

#### Technical Details
```go
// internal/probe/icmp.go
type ICMPProber struct {
    socket    *network.RawSocket
    identifier uint16
    sequence   uint32
    timeout    time.Duration
    ipv6       bool
}

func (p *ICMPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error) {
    // 1. Set TTL
    if err := p.socket.SetTTL(ttl); err != nil {
        return nil, err
    }
    
    // 2. Build ICMP packet
    seq := atomic.AddUint32(&p.sequence, 1)
    pkt := &ICMPPacket{
        Type:       ICMPv4EchoRequest,
        Code:       0,
        Identifier: p.identifier,
        Sequence:   uint16(seq),
        Payload:    makeTimestampPayload(),
    }
    
    // 3. Send packet
    sendTime := time.Now()
    if err := p.socket.WriteTo(pkt.Marshal(), dest); err != nil {
        return nil, err
    }
    
    // 4. Wait for response with timeout
    deadline := sendTime.Add(p.timeout)
    p.socket.SetReadDeadline(deadline)
    
    buf := make([]byte, 1500)
    for {
        n, from, err := p.socket.ReadFrom(buf)
        if err != nil {
            if isTimeout(err) {
                return &ProbeResult{RTT: p.timeout}, ErrTimeout
            }
            return nil, err
        }
        
        // 5. Parse response
        result, ok := p.parseResponse(buf[:n], from, seq, sendTime)
        if ok {
            return result, nil
        }
        // Not our packet, continue waiting
    }
}
```

#### Files to Touch
- `internal/probe/icmp.go` (new)
- `internal/probe/icmp_test.go` (new)

#### Dependencies
- T008: Raw socket
- T009: Packet builder
- T010: TTL manipulation

#### Success Criteria
- [ ] Can send ICMP Echo Request
- [ ] Receives Time Exceeded from intermediate hops
- [ ] Receives Echo Reply from destination
- [ ] RTT calculation is accurate
- [ ] Handles multiple concurrent probes

---

### T012: Implement ICMP Response Parser

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Parse incoming ICMP packets to extract response type, source IP, and match responses to sent probes using identifier and sequence number.

#### Technical Details
```go
// internal/probe/icmp.go
func (p *ICMPProber) parseResponse(data []byte, from net.Addr, expectedSeq uint16, sendTime time.Time) (*ProbeResult, bool) {
    // Skip IP header (usually 20 bytes for IPv4)
    ipHeaderLen := int(data[0]&0x0f) * 4
    icmpData := data[ipHeaderLen:]
    
    icmpType := icmpData[0]
    icmpCode := icmpData[1]
    
    rtt := time.Since(sendTime)
    fromIP := netAddrToIP(from)
    
    switch icmpType {
    case ICMPv4TimeExceeded:
        // Extract original ICMP header from payload
        // Payload starts at offset 8 in Time Exceeded message
        // Contains original IP header + first 8 bytes of original ICMP
        origICMP := icmpData[8+ipHeaderLen:]
        origID := binary.BigEndian.Uint16(origICMP[4:6])
        origSeq := binary.BigEndian.Uint16(origICMP[6:8])
        
        if origID == p.identifier && origSeq == expectedSeq {
            return &ProbeResult{
                ResponseIP: fromIP,
                RTT:        rtt,
                ICMPType:   int(icmpType),
                ICMPCode:   int(icmpCode),
                Reached:    false,
            }, true
        }
        
    case ICMPv4EchoReply:
        respID := binary.BigEndian.Uint16(icmpData[4:6])
        respSeq := binary.BigEndian.Uint16(icmpData[6:8])
        
        if respID == p.identifier && respSeq == expectedSeq {
            return &ProbeResult{
                ResponseIP: fromIP,
                RTT:        rtt,
                ICMPType:   int(icmpType),
                ICMPCode:   int(icmpCode),
                Reached:    true,
            }, true
        }
    }
    
    return nil, false
}
```

#### Files to Touch
- `internal/probe/icmp.go` (update)
- `internal/probe/icmp_parser.go` (new - if splitting)

#### Dependencies
- T011: Send/Receive implementation

#### Success Criteria
- [ ] Correctly identifies Time Exceeded
- [ ] Correctly identifies Echo Reply
- [ ] Matches response to correct probe
- [ ] Handles ICMPv6 responses

---

### T013: Add ICMP Probe Integration Test

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create integration tests that verify ICMP probing works end-to-end against real targets (localhost, local gateway, and optionally external hosts).

#### Technical Details
```go
// internal/probe/icmp_integration_test.go
//go:build integration

func TestICMPProbe_Localhost(t *testing.T) {
    if os.Getuid() != 0 {
        t.Skip("requires root")
    }
    
    prober, err := NewICMPProber(ICMPProberConfig{
        Timeout: 2 * time.Second,
    })
    require.NoError(t, err)
    defer prober.Close()
    
    result, err := prober.Probe(context.Background(), 
        net.ParseIP("127.0.0.1"), 64)
    require.NoError(t, err)
    
    assert.True(t, result.Reached)
    assert.Less(t, result.RTT, time.Millisecond)
}

func TestICMPProbe_Gateway(t *testing.T) {
    // Test against default gateway with TTL=1
}
```

#### Files to Touch
- `internal/probe/icmp_integration_test.go` (new)
- `scripts/test-icmp.sh` (new)

#### Dependencies
- T011: ICMP probe implementation
- T012: Response parser

#### Success Criteria
- [ ] Tests pass on Linux with root
- [ ] Tests properly skip without privileges
- [ ] CI integration (with sudo)

---

## Performance Targets
- Single probe RTT overhead: < 100μs
- Probe packet creation: < 10μs
- Response parsing: < 5μs
- Memory per probe: < 1KB

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Permission issues | High | High | Clear error messages, document CAP_NET_RAW |
| Firewall blocking | Medium | Medium | Document firewall requirements |
| IPv6 differences | Medium | Medium | Test both protocol versions early |
| Packet loss | Low | Low | Implement retry logic in tracer |

## Notes
- ICMP is the foundational probe method - quality here affects all tracing
- Consider using golang.org/x/net/icmp package as alternative to raw sockets
- Document the need for root/CAP_NET_RAW clearly in README
- Platform-specific code should be minimal and well-isolated
