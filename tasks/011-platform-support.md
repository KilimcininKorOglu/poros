# Feature 011: Cross-Platform Support (macOS & Windows)

**Feature ID:** F011
**Feature Name:** Cross-Platform Support (macOS & Windows)
**Priority:** P2 - HIGH
**Target Version:** v0.4.0
**Estimated Duration:** 2 weeks
**Status:** NOT_STARTED

## Overview

Extend Poros to work on macOS and Windows in addition to the primary Linux platform. This involves implementing platform-specific socket handling, privilege management, and testing on each platform.

Each platform has unique requirements for raw sockets: macOS uses BPF (Berkeley Packet Filter), Windows uses Winsock2 with administrator privileges, and Linux uses raw sockets with CAP_NET_RAW or root.

## Goals
- Implement macOS raw socket support using BPF
- Implement Windows raw socket support using Winsock2
- Handle platform-specific privilege requirements
- Ensure consistent behavior across all platforms

## Success Criteria
- [ ] All tasks completed (T067-T075)
- [ ] ICMP tracing works on macOS
- [ ] ICMP tracing works on Windows
- [ ] UDP tracing works on all platforms
- [ ] Error messages are platform-appropriate

## Tasks

### T067: Implement macOS ICMP Socket

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Implement ICMP socket handling for macOS using the system's ICMP socket type or BPF for raw access.

#### Technical Details
```go
// internal/network/socket_darwin.go
//go:build darwin

import (
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv4"
)

type DarwinICMPSocket struct {
    conn   *icmp.PacketConn
    ipv6   bool
}

func NewRawICMPSocket(ipv6 bool) (*RawSocket, error) {
    // macOS supports "udp4" for unprivileged ICMP (limited)
    // or "ip4:icmp" for raw ICMP (requires root)
    
    network := "ip4:icmp"
    if ipv6 {
        network = "ip6:ipv6-icmp"
    }
    
    conn, err := icmp.ListenPacket(network, "")
    if err != nil {
        // Try unprivileged mode
        if ipv6 {
            conn, err = icmp.ListenPacket("udp6", "")
        } else {
            conn, err = icmp.ListenPacket("udp4", "")
        }
        if err != nil {
            return nil, fmt.Errorf("ICMP socket requires root: %w", err)
        }
    }
    
    return &RawSocket{
        conn: conn,
        ipv6: ipv6,
    }, nil
}

func (s *RawSocket) SetTTL(ttl int) error {
    p := ipv4.NewPacketConn(s.conn)
    return p.SetTTL(ttl)
}

func (s *RawSocket) WriteTo(data []byte, dst net.IP) error {
    addr := &net.IPAddr{IP: dst}
    _, err := s.conn.WriteTo(data, addr)
    return err
}

func (s *RawSocket) ReadFrom(buf []byte) (int, net.Addr, error) {
    return s.conn.ReadFrom(buf)
}

func (s *RawSocket) SetReadDeadline(t time.Time) error {
    return s.conn.SetReadDeadline(t)
}
```

#### Files to Touch
- `internal/network/socket_darwin.go` (new)
- `internal/network/socket_darwin_test.go` (new)

#### Dependencies
- T008: Linux socket implementation (for reference)

#### Success Criteria
- [ ] ICMP socket works with sudo
- [ ] TTL manipulation works
- [ ] Proper errors without sudo
- [ ] IPv6 support

---

### T068: Implement macOS UDP Socket

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement UDP socket with TTL control for macOS traceroute.

#### Technical Details
```go
// internal/network/udp_socket_darwin.go
//go:build darwin

func NewUDPSocket(config UDPSocketConfig) (*UDPSocket, error) {
    // Standard UDP socket works on macOS
    laddr := &net.UDPAddr{
        IP:   config.SourceIP,
        Port: config.SourcePort,
    }
    
    conn, err := net.ListenUDP("udp4", laddr)
    if err != nil {
        return nil, err
    }
    
    return &UDPSocket{
        conn: conn,
    }, nil
}

func (s *UDPSocket) SetTTL(ttl int) error {
    rawConn, err := s.conn.SyscallConn()
    if err != nil {
        return err
    }
    
    var setErr error
    err = rawConn.Control(func(fd uintptr) {
        setErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, 
            unix.IP_TTL, ttl)
    })
    
    if err != nil {
        return err
    }
    return setErr
}
```

#### Files to Touch
- `internal/network/udp_socket_darwin.go` (new)

#### Dependencies
- T027: UDP socket implementation

#### Success Criteria
- [ ] UDP socket creation works
- [ ] TTL setting works
- [ ] Receives ICMP responses

---

### T069: Implement Windows ICMP Socket

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Implement ICMP handling for Windows using Winsock2 raw sockets or the IcmpSendEcho API.

#### Technical Details
```go
// internal/network/socket_windows.go
//go:build windows

import (
    "golang.org/x/sys/windows"
)

type WindowsICMPSocket struct {
    handle windows.Handle
    ipv6   bool
}

func NewRawICMPSocket(ipv6 bool) (*RawSocket, error) {
    // Windows requires Administrator for raw sockets
    // Alternative: Use IcmpCreateFile/IcmpSendEcho2 API
    
    var family, proto int
    if ipv6 {
        family = windows.AF_INET6
        proto = windows.IPPROTO_ICMPV6
    } else {
        family = windows.AF_INET
        proto = windows.IPPROTO_ICMP
    }
    
    // Try raw socket first (requires admin)
    fd, err := windows.Socket(int32(family), windows.SOCK_RAW, int32(proto))
    if err != nil {
        // Fallback to ICMP API
        return newICMPAPISocket(ipv6)
    }
    
    return &RawSocket{
        fd:   fd,
        ipv6: ipv6,
    }, nil
}

// Alternative using Windows ICMP API (works without admin for echo)
type ICMPAPISocket struct {
    handle windows.Handle
}

func newICMPAPISocket(ipv6 bool) (*RawSocket, error) {
    // IcmpCreateFile for IPv4, Icmp6CreateFile for IPv6
    handle, err := windows.IcmpCreateFile()
    if err != nil {
        return nil, fmt.Errorf("failed to create ICMP handle: %w", err)
    }
    
    return &RawSocket{
        icmpHandle: handle,
        useAPI:     true,
    }, nil
}

func (s *RawSocket) SetTTL(ttl int) error {
    if s.useAPI {
        // TTL is set per-request with ICMP API
        s.ttl = ttl
        return nil
    }
    
    return windows.SetsockoptInt(s.fd, windows.IPPROTO_IP, 
        windows.IP_TTL, ttl)
}
```

#### Files to Touch
- `internal/network/socket_windows.go` (new)
- `internal/network/icmp_api_windows.go` (new)
- `internal/network/socket_windows_test.go` (new)

#### Dependencies
- T008: Linux socket implementation

#### Success Criteria
- [ ] Works with Administrator
- [ ] Clear error without admin
- [ ] ICMP API fallback works
- [ ] IPv6 support

---

### T070: Implement Windows UDP Socket

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement UDP socket with TTL control for Windows.

#### Technical Details
```go
// internal/network/udp_socket_windows.go
//go:build windows

func NewUDPSocket(config UDPSocketConfig) (*UDPSocket, error) {
    laddr := &net.UDPAddr{
        IP:   config.SourceIP,
        Port: config.SourcePort,
    }
    
    conn, err := net.ListenUDP("udp4", laddr)
    if err != nil {
        return nil, err
    }
    
    return &UDPSocket{
        conn: conn,
    }, nil
}

func (s *UDPSocket) SetTTL(ttl int) error {
    rawConn, err := s.conn.SyscallConn()
    if err != nil {
        return err
    }
    
    var setErr error
    err = rawConn.Control(func(fd uintptr) {
        // Windows uses the same socket option
        setErr = windows.SetsockoptInt(windows.Handle(fd), 
            windows.IPPROTO_IP, windows.IP_TTL, ttl)
    })
    
    if err != nil {
        return err
    }
    return setErr
}
```

#### Files to Touch
- `internal/network/udp_socket_windows.go` (new)

#### Dependencies
- T069: Windows ICMP socket

#### Success Criteria
- [ ] UDP socket works
- [ ] TTL manipulation works
- [ ] Receives ICMP responses

---

### T071: Implement Platform-Specific Error Messages

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create platform-appropriate error messages that guide users on how to run with required privileges.

#### Technical Details
```go
// internal/network/errors.go
var ErrPermissionDenied = errors.New("permission denied")

func PermissionError() error {
    switch runtime.GOOS {
    case "linux":
        return fmt.Errorf(`%w

Raw socket requires elevated privileges.

Solutions:
  1. Run with sudo: sudo poros google.com
  2. Set capabilities: sudo setcap cap_net_raw+ep ./poros
  3. Use unprivileged mode: poros --unprivileged google.com`, 
            ErrPermissionDenied)
    
    case "darwin":
        return fmt.Errorf(`%w

Raw socket requires root privileges on macOS.

Solution:
  Run with sudo: sudo poros google.com`, 
            ErrPermissionDenied)
    
    case "windows":
        return fmt.Errorf(`%w

Raw socket requires Administrator privileges on Windows.

Solution:
  Run Command Prompt or PowerShell as Administrator`, 
            ErrPermissionDenied)
    
    default:
        return ErrPermissionDenied
    }
}

// internal/network/privileges.go
func CheckPrivileges() error {
    switch runtime.GOOS {
    case "linux", "darwin":
        if os.Getuid() != 0 {
            // Check for CAP_NET_RAW on Linux
            if runtime.GOOS == "linux" && hasCapNetRaw() {
                return nil
            }
            return PermissionError()
        }
    case "windows":
        if !isWindowsAdmin() {
            return PermissionError()
        }
    }
    return nil
}

func isWindowsAdmin() bool {
    // Windows admin check
    _, err := os.Open("\\\\.\\PHYSICALDRIVE0")
    return err == nil
}
```

#### Files to Touch
- `internal/network/errors.go` (update)
- `internal/network/privileges.go` (new)
- `internal/network/privileges_linux.go` (new)
- `internal/network/privileges_darwin.go` (new)
- `internal/network/privileges_windows.go` (new)

#### Dependencies
- T067-T070: Platform socket implementations

#### Success Criteria
- [ ] Clear error messages per platform
- [ ] CAP_NET_RAW detection on Linux
- [ ] Admin detection on Windows

---

### T072: Add Interface Enumeration Per Platform

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
Implement network interface listing and selection for each platform.

#### Technical Details
```go
// internal/network/interface.go
type NetworkInterface struct {
    Name       string
    Index      int
    HWAddr     string
    Addresses  []net.IP
    IsUp       bool
    IsLoopback bool
}

func ListInterfaces() ([]NetworkInterface, error) {
    ifaces, err := net.Interfaces()
    if err != nil {
        return nil, err
    }
    
    result := make([]NetworkInterface, 0, len(ifaces))
    for _, iface := range ifaces {
        ni := NetworkInterface{
            Name:       iface.Name,
            Index:      iface.Index,
            HWAddr:     iface.HardwareAddr.String(),
            IsUp:       iface.Flags&net.FlagUp != 0,
            IsLoopback: iface.Flags&net.FlagLoopback != 0,
        }
        
        addrs, err := iface.Addrs()
        if err == nil {
            for _, addr := range addrs {
                if ipnet, ok := addr.(*net.IPNet); ok {
                    ni.Addresses = append(ni.Addresses, ipnet.IP)
                }
            }
        }
        
        result = append(result, ni)
    }
    
    return result, nil
}

func GetInterfaceByName(name string) (*NetworkInterface, error) {
    ifaces, err := ListInterfaces()
    if err != nil {
        return nil, err
    }
    
    for _, iface := range ifaces {
        if iface.Name == name {
            return &iface, nil
        }
    }
    
    return nil, fmt.Errorf("interface %q not found", name)
}

// cmd/poros/interfaces.go
var listInterfacesCmd = &cobra.Command{
    Use:     "interfaces",
    Aliases: []string{"if", "list-if"},
    Short:   "List network interfaces",
    RunE: func(cmd *cobra.Command, args []string) error {
        ifaces, err := network.ListInterfaces()
        if err != nil {
            return err
        }
        
        table := tablewriter.NewWriter(os.Stdout)
        table.SetHeader([]string{"Name", "Index", "MAC", "IP Addresses", "Status"})
        
        for _, iface := range ifaces {
            if iface.IsLoopback {
                continue
            }
            
            status := "DOWN"
            if iface.IsUp {
                status = "UP"
            }
            
            ips := make([]string, len(iface.Addresses))
            for i, ip := range iface.Addresses {
                ips[i] = ip.String()
            }
            
            table.Append([]string{
                iface.Name,
                fmt.Sprintf("%d", iface.Index),
                iface.HWAddr,
                strings.Join(ips, ", "),
                status,
            })
        }
        
        table.Render()
        return nil
    },
}
```

#### Files to Touch
- `internal/network/interface.go` (new)
- `cmd/poros/interfaces.go` (new)

#### Dependencies
- None (uses standard library)

#### Success Criteria
- [ ] Lists interfaces on all platforms
- [ ] Shows IP addresses
- [ ] Handles interface by name or index

---

### T073: Build Cross-Platform Binaries

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Set up build scripts for cross-compilation to all supported platforms.

#### Technical Details
```makefile
# Makefile (update)
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build-all
build-all: $(PLATFORMS)

$(PLATFORMS):
	$(eval OS := $(word 1,$(subst /, ,$@)))
	$(eval ARCH := $(word 2,$(subst /, ,$@)))
	$(eval EXT := $(if $(filter windows,$(OS)),.exe,))
	GOOS=$(OS) GOARCH=$(ARCH) go build -ldflags "$(LDFLAGS)" \
		-o bin/poros-$(OS)-$(ARCH)$(EXT) ./cmd/poros

.PHONY: release
release: clean build-all
	cd bin && \
	for f in poros-linux-* poros-darwin-*; do \
		tar -czf $$f.tar.gz $$f; \
	done && \
	for f in poros-windows-*; do \
		zip $$f.zip $$f; \
	done
```

```powershell
# build.bat (Windows build script)
@echo off
setlocal

set VERSION=dev
for /f %%i in ('git describe --tags --always --dirty 2^>nul') do set VERSION=%%i

set LDFLAGS=-ldflags "-X main.version=%VERSION%"

echo Building poros for Windows...
go build %LDFLAGS% -o bin\poros.exe .\cmd\poros

if %ERRORLEVEL% EQU 0 (
    echo Build successful: bin\poros.exe
) else (
    echo Build failed!
    exit /b 1
)
```

#### Files to Touch
- `Makefile` (update)
- `build.bat` (update)
- `scripts/build-all.sh` (new)

#### Dependencies
- T001: Project structure

#### Success Criteria
- [ ] Linux builds work
- [ ] macOS builds work
- [ ] Windows builds work
- [ ] ARM64 builds work

---

### T074: Add Platform Integration Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
Create integration tests that run on each platform in CI.

#### Technical Details
```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  test-linux:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v ./...
      - name: Run integration tests
        run: sudo go test -v -tags=integration ./...

  test-macos:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v ./...
      - name: Run integration tests
        run: sudo go test -v -tags=integration ./...

  test-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - name: Run tests
        run: go test -v ./...
      # Note: Integration tests on Windows require elevated privileges
```

```go
// internal/network/socket_test.go
func TestRawSocket_Platform(t *testing.T) {
    switch runtime.GOOS {
    case "linux":
        t.Run("Linux", testLinuxSocket)
    case "darwin":
        t.Run("macOS", testDarwinSocket)
    case "windows":
        t.Run("Windows", testWindowsSocket)
    }
}
```

#### Files to Touch
- `.github/workflows/test.yml` (new)
- `internal/network/socket_test.go` (new)

#### Dependencies
- T067-T070: Platform implementations

#### Success Criteria
- [ ] CI runs on all platforms
- [ ] Tests pass on Linux
- [ ] Tests pass on macOS
- [ ] Tests pass on Windows

---

### T075: Document Platform-Specific Behavior

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Document platform differences, privilege requirements, and known limitations.

#### Technical Details
```markdown
<!-- docs/PLATFORMS.md -->
# Platform Support

## Linux

### Privileges
- **Root**: Full access to all probe methods
- **CAP_NET_RAW**: ICMP and UDP probes without root
  ```bash
  sudo setcap cap_net_raw+ep ./poros
  ```
- **Unprivileged**: UDP high-port only

### Socket Types
- Raw ICMP: `socket(AF_INET, SOCK_RAW, IPPROTO_ICMP)`
- UDP: Standard UDP socket with `IP_TTL` option

## macOS

### Privileges
- **Root required** for ICMP raw sockets
- UDP traceroute works without root

### Socket Types
- ICMP: `golang.org/x/net/icmp` with "ip4:icmp"
- UDP: Standard UDP socket

### Known Limitations
- Some BPF restrictions in newer macOS versions
- TCP raw sockets require kernel extension

## Windows

### Privileges
- **Administrator required** for raw sockets
- ICMP API works without admin (limited)

### Socket Types
- Raw ICMP: Winsock2 `SOCK_RAW`
- ICMP API: `IcmpSendEcho2` (fallback)
- UDP: Standard Winsock

### Known Limitations
- No TCP SYN probe without admin
- ICMP API doesn't support TTL < 1
```

#### Files to Touch
- `docs/PLATFORMS.md` (new)
- `README.md` (update with platform notes)

#### Dependencies
- T067-T074: Platform implementations complete

#### Success Criteria
- [ ] All platforms documented
- [ ] Privilege requirements clear
- [ ] Known issues listed

---

## Performance Targets
- Platform detection: < 1ms
- Socket creation: < 10ms
- Cross-compile time: < 2 minutes

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Windows API differences | High | High | Thorough testing, fallbacks |
| macOS security restrictions | Medium | Medium | Document workarounds |
| Inconsistent behavior | Medium | High | Extensive cross-platform testing |

## Notes
- Windows is often the most challenging platform
- macOS may require code signing for distribution
- Consider using cgo sparingly for better cross-compilation
- Test on actual hardware, not just CI VMs
