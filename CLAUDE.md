# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Poros** (Greek: Πόρος - "path, passage") is a modern, cross-platform network path tracer written in Go. It's designed as a feature-rich alternative to traditional traceroute/tracert tools with concurrent probing, ASN/GeoIP enrichment, and TUI support.

**Status:** In development - PRD complete, implementation pending.

## Target Architecture

```
poros/
├── cmd/poros/main.go           # CLI entry point (cobra)
├── internal/
│   ├── probe/                  # Probe implementations (ICMP, UDP, TCP, Paris)
│   ├── trace/                  # Tracer logic (sequential, concurrent, adaptive)
│   ├── enrich/                 # Enrichment (rDNS, ASN, GeoIP, cache)
│   ├── network/                # Platform-specific socket handling
│   ├── output/                 # Formatters (text, JSON, CSV, HTML)
│   └── tui/                    # Bubble Tea TUI
├── pkg/poros/                  # Public API for library usage
├── data/                       # GeoIP databases (gitignored)
├── scripts/                    # Build, install, download scripts
└── docs/                       # Documentation
```

## Build Commands

| Command | Description |
|---------|-------------|
| `go build ./cmd/poros` | Build binary |
| `go test ./...` | Run all tests |
| `go test -v ./internal/probe/...` | Test specific package |
| `go test -race ./...` | Run tests with race detector |
| `go test -bench=. ./...` | Run benchmarks |
| `golangci-lint run` | Lint code |
| `go mod tidy` | Clean up dependencies |

## Key Dependencies (Planned)

```
github.com/spf13/cobra          # CLI framework
github.com/charmbracelet/bubbletea  # TUI framework
golang.org/x/net                # Network utilities
golang.org/x/sys                # System calls (raw sockets)
github.com/oschwald/maxminddb-golang  # GeoIP database
github.com/miekg/dns            # DNS lookups
github.com/olekukonko/tablewriter    # Table output
github.com/fatih/color          # Colored output
```

## Core Interfaces

### Prober Interface
```go
type Prober interface {
    Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error)
    Name() string
    RequiresRoot() bool
    Close() error
}
```

### Key Data Structures
- `Hop` - Single hop with IP, hostname, ASN, GeoIP, RTT stats
- `TraceResult` - Complete trace with hops and summary
- `TracerConfig` - Configuration for probe method, timeouts, enrichment

## Platform-Specific Notes

| Platform | Socket Type | Notes |
|----------|-------------|-------|
| Linux | Raw socket | Full ICMP/TCP SYN support, requires root/CAP_NET_RAW |
| macOS | BPF socket | Some restrictions, requires root |
| Windows | Winsock2 | Requires Administrator |

File naming convention for platform code:
- `socket_linux.go`
- `socket_darwin.go`
- `socket_windows.go`

## Probe Methods

1. **ICMP Echo** (default) - Raw socket, Type 8 echo request
2. **UDP** - Ports 33434-33534, high-port fallback
3. **TCP SYN** - Half-open connections, ports 80/443
4. **Paris** - Flow-identifier consistent for load balancers

## Development Phases

| Phase | Version | Focus |
|-------|---------|-------|
| MVP | v0.1.0 | ICMP probe, sequential trace, Linux |
| Core | v0.2.0 | UDP probe, concurrent mode, macOS |
| Enrichment | v0.3.0 | ASN, GeoIP, caching |
| Advanced | v0.4.0 | TCP SYN, Paris, IPv6, Windows |
| TUI | v0.5.0 | Bubble Tea TUI, HTML export |
| Release | v1.0.0 | Testing, optimization, packaging |

## CLI Design

```bash
# Basic usage
poros google.com
poros -I/-U/-T google.com    # ICMP/UDP/TCP probe
poros -4/-6 google.com       # Force IPv4/IPv6
poros --paris google.com     # Paris mode

# Parameters
poros -m 30 -q 3 -w 3s google.com  # max-hops, queries, timeout
poros -i eth0 -s 192.168.1.1 google.com  # interface, source

# Output
poros -v/--verbose google.com    # Detailed table
poros --json/--csv google.com    # Structured output
poros --tui google.com           # Interactive TUI
```

## Testing Strategy

- Unit tests: Mock raw sockets, test checksum calculations
- Integration tests: Shell scripts for probe methods
- Platform matrix: Linux/macOS (amd64/arm64), Windows
- Coverage target: >70%

## Code Style

- Use `golangci-lint` for linting
- Conventional commits for version control
- Error types in `internal/*/errors.go`
- User-friendly error messages with solutions

## Network Permissions

The tool requires elevated privileges for raw sockets:
```bash
# Linux: root or CAP_NET_RAW
sudo poros google.com
sudo setcap cap_net_raw+ep ./poros

# macOS: root
sudo poros google.com

# Windows: Run as Administrator
```

## External Data Sources

| Data | Primary Source | Fallback |
|------|----------------|----------|
| ASN | Local MaxMind DB | Team Cymru DNS |
| GeoIP | Local MaxMind DB | ip-api.com |
| rDNS | System resolver | - |

## Performance Targets

- Cold start: <100ms
- 30 hop trace (concurrent): <5s
- Memory: <50MB
- Binary size: <15MB (stripped)
