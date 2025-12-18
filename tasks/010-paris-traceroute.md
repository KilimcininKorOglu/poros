# Feature 010: Paris Traceroute Algorithm

**Feature ID:** F010
**Feature Name:** Paris Traceroute Algorithm
**Priority:** P3 - MEDIUM
**Target Version:** v0.4.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Implement the Paris Traceroute algorithm that maintains consistent flow identifiers across all probes to a destination. This ensures probes follow the same path through load-balanced networks, providing accurate path discovery.

Traditional traceroute varies probe characteristics (ports, ICMP identifiers) which causes probes to be distributed across multiple paths in load-balanced environments, showing false "diamonds" or unstable paths.

## Goals
- Implement Paris algorithm for ICMP, UDP, and TCP probes
- Maintain consistent 5-tuple flow hash
- Detect multi-path load balancing
- Report per-flow paths correctly

## Success Criteria
- [ ] All tasks completed (T061-T066)
- [ ] Same flow hash for all TTL values
- [ ] Consistent paths through ECMP networks
- [ ] Multi-path detection works
- [ ] Algorithm documented for users

## Tasks

### T061: Implement Flow Hash Calculator

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create a flow hash calculator that determines consistent values for probe fields that affect load balancer routing decisions.

#### Technical Details
```go
// internal/probe/paris/flow.go
type FlowIdentifier struct {
    SrcIP     net.IP
    DstIP     net.IP
    Protocol  uint8
    SrcPort   uint16
    DstPort   uint16
    ICMPIdent uint16
}

func (f *FlowIdentifier) Hash() uint32 {
    h := fnv.New32a()
    h.Write(f.SrcIP)
    h.Write(f.DstIP)
    h.Write([]byte{f.Protocol})
    
    buf := make([]byte, 2)
    binary.BigEndian.PutUint16(buf, f.SrcPort)
    h.Write(buf)
    binary.BigEndian.PutUint16(buf, f.DstPort)
    h.Write(buf)
    
    return h.Sum32()
}

// ParisConfig holds Paris traceroute settings
type ParisConfig struct {
    // Number of flows to probe (for multipath detection)
    NumFlows int
    // Base flow identifier
    BaseFlow FlowIdentifier
}

// FlowGenerator creates consistent probe parameters
type FlowGenerator struct {
    srcIP    net.IP
    dstIP    net.IP
    protocol uint8
    flowNum  int
}

func NewFlowGenerator(srcIP, dstIP net.IP, protocol uint8, flowNum int) *FlowGenerator {
    return &FlowGenerator{
        srcIP:    srcIP,
        dstIP:    dstIP,
        protocol: protocol,
        flowNum:  flowNum,
    }
}

func (g *FlowGenerator) UDPPorts() (srcPort, dstPort uint16) {
    // Keep source port constant for flow, vary dest port based on flow number
    // This ensures same path while allowing multipath detection
    basePort := uint16(33434)
    srcPort = basePort + uint16(g.flowNum)
    dstPort = basePort // Constant for all TTLs
    return
}

func (g *FlowGenerator) ICMPIdentifier() uint16 {
    // ICMP identifier affects some load balancers
    // Keep constant within a flow
    return uint16(g.flowNum + 1)
}

func (g *FlowGenerator) TCPPorts() (srcPort, dstPort uint16) {
    srcPort = uint16(49152 + g.flowNum)
    dstPort = 80 // or user-specified
    return
}
```

#### Files to Touch
- `internal/probe/paris/flow.go` (new)
- `internal/probe/paris/flow_test.go` (new)

#### Dependencies
- T003: Prober interface

#### Success Criteria
- [ ] Consistent hash for same flow
- [ ] Different hash for different flows
- [ ] Works with all probe types

---

### T062: Implement Paris ICMP Probe

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement Paris-style ICMP probing that manipulates the ICMP checksum to maintain a constant identifier across varying TTLs.

#### Technical Details
```go
// internal/probe/paris/icmp.go
type ParisICMPProber struct {
    socket    *network.RawSocket
    flow      *FlowGenerator
    timeout   time.Duration
}

func NewParisICMPProber(config ParisICMPConfig) (*ParisICMPProber, error) {
    socket, err := network.NewRawICMPSocket(false)
    if err != nil {
        return nil, err
    }
    
    flow := NewFlowGenerator(config.SrcIP, config.DstIP, 
        syscall.IPPROTO_ICMP, config.FlowNum)
    
    return &ParisICMPProber{
        socket:  socket,
        flow:    flow,
        timeout: config.Timeout,
    }, nil
}

func (p *ParisICMPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error) {
    // Set TTL
    if err := p.socket.SetTTL(ttl); err != nil {
        return nil, err
    }
    
    // Build Paris ICMP packet
    // Key: Identifier is constant, but we vary payload to get desired checksum
    identifier := p.flow.ICMPIdentifier()
    sequence := uint16(ttl) // Use TTL as sequence for matching
    
    pkt := &ICMPPacket{
        Type:       ICMPv4EchoRequest,
        Code:       0,
        Identifier: identifier,
        Sequence:   sequence,
    }
    
    // Calculate payload to achieve constant checksum
    // This keeps the flow hash constant across TTLs
    pkt.Payload = p.calculateParisPayload(pkt, ttl)
    
    // ... send and receive as normal
}

func (p *ParisICMPProber) calculateParisPayload(pkt *ICMPPacket, ttl int) []byte {
    // Paris trick: manipulate payload so that checksum is constant
    // Checksum is part of flow ID for some load balancers
    
    targetChecksum := p.flow.ICMPIdentifier() // Use identifier as target
    
    // Start with timestamp payload
    payload := make([]byte, 16)
    binary.BigEndian.PutUint64(payload[0:8], uint64(time.Now().UnixNano()))
    binary.BigEndian.PutUint16(payload[8:10], uint16(ttl))
    
    // Calculate current checksum
    pkt.Payload = payload
    tempData, _ := pkt.MarshalWithoutChecksum()
    currentSum := partialChecksum(tempData)
    
    // Adjust last two bytes to achieve target
    adjustment := targetChecksum - currentSum
    binary.BigEndian.PutUint16(payload[14:16], adjustment)
    
    return payload
}
```

#### Files to Touch
- `internal/probe/paris/icmp.go` (new)
- `internal/probe/paris/icmp_test.go` (new)

#### Dependencies
- T061: Flow hash calculator
- T011: ICMP probe implementation

#### Success Criteria
- [ ] Constant checksum across TTLs
- [ ] Works through load balancers
- [ ] Probe matching still works

---

### T063: Implement Paris UDP Probe

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement Paris-style UDP probing with constant source port to maintain flow consistency.

#### Technical Details
```go
// internal/probe/paris/udp.go
type ParisUDPProber struct {
    socket  *network.UDPSocket
    flow    *FlowGenerator
    timeout time.Duration
}

func NewParisUDPProber(config ParisUDPConfig) (*ParisUDPProber, error) {
    // Get constant source port for this flow
    srcPort, _ := NewFlowGenerator(config.SrcIP, config.DstIP,
        syscall.IPPROTO_UDP, config.FlowNum).UDPPorts()
    
    socket, err := network.NewUDPSocketWithPort(network.UDPSocketConfig{
        SourceIP:   config.SrcIP,
        SourcePort: int(srcPort), // Fixed source port!
    })
    if err != nil {
        return nil, err
    }
    
    return &ParisUDPProber{
        socket:  socket,
        flow:    NewFlowGenerator(config.SrcIP, config.DstIP, 
            syscall.IPPROTO_UDP, config.FlowNum),
        timeout: config.Timeout,
    }, nil
}

func (p *ParisUDPProber) Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error) {
    // Set TTL
    if err := p.socket.SetTTL(ttl); err != nil {
        return nil, err
    }
    
    // Paris: destination port is CONSTANT (unlike classic traceroute)
    // Classic traceroute: dstPort = 33434 + ttl
    // Paris traceroute: dstPort = 33434 (constant)
    dstPort := uint16(33434)
    
    // Build probe payload with TTL for identification
    payload := makeParisUDPPayload(ttl, p.flow.flowNum)
    
    sendTime := time.Now()
    destAddr := &net.UDPAddr{IP: dest, Port: int(dstPort)}
    
    if _, err := p.socket.WriteTo(payload, destAddr); err != nil {
        return nil, err
    }
    
    // ... receive logic
}

func makeParisUDPPayload(ttl, flowNum int) []byte {
    payload := make([]byte, 16)
    binary.BigEndian.PutUint64(payload[0:8], uint64(time.Now().UnixNano()))
    binary.BigEndian.PutUint16(payload[8:10], uint16(ttl))
    binary.BigEndian.PutUint16(payload[10:12], uint16(flowNum))
    return payload
}
```

#### Files to Touch
- `internal/probe/paris/udp.go` (new)
- `internal/probe/paris/udp_test.go` (new)

#### Dependencies
- T061: Flow hash calculator
- T028: UDP probe implementation

#### Success Criteria
- [ ] Constant source/dest port per flow
- [ ] TTL encoded in payload
- [ ] Works through ECMP routers

---

### T064: Implement Multi-Path Detection

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
Implement multi-path detection by sending probes with different flow identifiers and comparing resulting paths.

#### Technical Details
```go
// internal/probe/paris/multipath.go
type MultiPathDetector struct {
    proberFactory func(flowNum int) Prober
    numFlows      int
    timeout       time.Duration
}

type MultiPathResult struct {
    Hop        int
    Paths      []PathInfo
    IsMultiPath bool
}

type PathInfo struct {
    FlowNum int
    IP      net.IP
    RTT     time.Duration
}

func (d *MultiPathDetector) DetectAtHop(ctx context.Context, dest net.IP, ttl int) (*MultiPathResult, error) {
    result := &MultiPathResult{
        Hop:   ttl,
        Paths: make([]PathInfo, 0, d.numFlows),
    }
    
    // Send probes with different flow IDs
    var wg sync.WaitGroup
    var mu sync.Mutex
    
    for flowNum := 0; flowNum < d.numFlows; flowNum++ {
        wg.Add(1)
        go func(fn int) {
            defer wg.Done()
            
            prober := d.proberFactory(fn)
            defer prober.Close()
            
            probeResult, err := prober.Probe(ctx, dest, ttl)
            if err != nil {
                return
            }
            
            mu.Lock()
            result.Paths = append(result.Paths, PathInfo{
                FlowNum: fn,
                IP:      probeResult.ResponseIP,
                RTT:     probeResult.RTT,
            })
            mu.Unlock()
        }(flowNum)
    }
    
    wg.Wait()
    
    // Detect multipath
    result.IsMultiPath = d.detectMultiPath(result.Paths)
    
    return result, nil
}

func (d *MultiPathDetector) detectMultiPath(paths []PathInfo) bool {
    if len(paths) < 2 {
        return false
    }
    
    // Check if different flows got different IPs
    ips := make(map[string]bool)
    for _, p := range paths {
        if p.IP != nil {
            ips[p.IP.String()] = true
        }
    }
    
    return len(ips) > 1
}
```

#### Files to Touch
- `internal/probe/paris/multipath.go` (new)
- `internal/probe/paris/multipath_test.go` (new)

#### Dependencies
- T062: Paris ICMP probe
- T063: Paris UDP probe

#### Success Criteria
- [ ] Detects ECMP load balancing
- [ ] Reports per-flow paths
- [ ] Works with concurrent tracing

---

### T065: Register Paris Mode with CLI

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Add Paris traceroute mode to CLI and integrate with tracer system.

#### Technical Details
```go
// cmd/poros/root.go (update)
func init() {
    rootCmd.Flags().Bool("paris", false, 
        "Use Paris traceroute algorithm for consistent paths")
    rootCmd.Flags().Int("flows", 1, 
        "Number of flows for multipath detection (Paris mode)")
}

// internal/trace/tracer.go (update)
func NewTracer(config *TracerConfig) (*Tracer, error) {
    var prober probe.Prober
    
    if config.Paris {
        switch config.ProbeMethod {
        case probe.ProbeICMP:
            prober, err = paris.NewParisICMPProber(paris.ParisICMPConfig{
                Timeout: config.Timeout,
                SrcIP:   config.SourceIP,
                DstIP:   config.DestIP,
                FlowNum: 0,
            })
        case probe.ProbeUDP:
            prober, err = paris.NewParisUDPProber(...)
        }
    } else {
        // Regular probers
    }
    
    // ...
}
```

#### Files to Touch
- `cmd/poros/root.go` (update)
- `internal/trace/tracer.go` (update)
- `internal/trace/config.go` (update)

#### Dependencies
- T062-T064: Paris probers

#### Success Criteria
- [ ] `--paris` enables Paris mode
- [ ] `--flows` sets flow count
- [ ] Help text explains Paris mode

---

### T066: Add Paris Mode Documentation and Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Document Paris traceroute algorithm and add comprehensive tests.

#### Technical Details
```markdown
<!-- docs/PARIS.md -->
# Paris Traceroute Mode

## What is Paris Traceroute?

Traditional traceroute varies packet characteristics (ports, ICMP IDs) for 
each probe. In networks with load balancing (ECMP), this causes probes to 
follow different paths, showing misleading results.

Paris traceroute keeps the "flow identifier" constant across all probes,
ensuring they follow the same path through load balancers.

## How Poros Implements Paris

### ICMP Mode
- Keeps ICMP identifier constant
- Manipulates payload to maintain constant checksum
- Uses sequence number for TTL identification

### UDP Mode  
- Fixes source AND destination port (unlike classic traceroute)
- Encodes TTL in payload for probe matching

### TCP Mode
- Fixes source port
- Uses sequence numbers for probe matching

## Usage

```bash
# Basic Paris trace
poros --paris google.com

# With multipath detection
poros --paris --flows 8 google.com
```
```

```go
// internal/probe/paris/paris_test.go
func TestParisFlow_Consistency(t *testing.T) {
    flow := NewFlowGenerator(
        net.ParseIP("192.168.1.1"),
        net.ParseIP("8.8.8.8"),
        syscall.IPPROTO_UDP,
        0,
    )
    
    srcPort1, dstPort1 := flow.UDPPorts()
    srcPort2, dstPort2 := flow.UDPPorts()
    
    assert.Equal(t, srcPort1, srcPort2, "Source port should be constant")
    assert.Equal(t, dstPort1, dstPort2, "Dest port should be constant")
}
```

#### Files to Touch
- `docs/PARIS.md` (new)
- `internal/probe/paris/paris_test.go` (new)
- `internal/probe/paris/multipath_integration_test.go` (new)

#### Dependencies
- T065: Paris mode integrated

#### Success Criteria
- [ ] Documentation is clear
- [ ] Tests verify consistency
- [ ] Integration tests in ECMP networks (if available)

---

## Performance Targets
- Paris overhead vs regular: < 5%
- Multipath detection: < 10s for 8 flows
- Memory: No significant increase

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Algorithm complexity | Medium | Medium | Clear documentation |
| Limited testing environments | High | Medium | Document expected behavior |
| Some load balancers behave differently | Medium | Low | Support multiple flow IDs |

## Notes
- Paris traceroute is based on academic research paper
- Not all load balancers use the same hash fields
- Multipath detection requires sending multiple flows
- Some networks may show multipath at some hops but not others
