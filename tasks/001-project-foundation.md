# Feature 001: Project Foundation

**Feature ID:** F001
**Feature Name:** Project Foundation & Core Infrastructure
**Priority:** P1 - CRITICAL
**Target Version:** v0.1.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Establish the foundational project structure for Poros, including Go module initialization, directory structure, core interfaces, and build system. This feature sets up the skeleton upon which all other features will be built.

The foundation includes creating the proper Go project layout with `cmd/`, `internal/`, and `pkg/` directories, defining core data structures and interfaces, setting up the build toolchain with Makefile, and configuring development dependencies like linters and testing frameworks.

## Goals
- Create a well-organized Go project structure following best practices
- Define core interfaces and data structures used throughout the codebase
- Set up build automation for cross-platform compilation
- Configure development tooling (linting, testing, formatting)

## Success Criteria
- [ ] All tasks completed (T001-T006)
- [ ] `go build ./cmd/poros` compiles successfully
- [ ] `go test ./...` runs without errors
- [ ] `golangci-lint run` passes with no issues
- [ ] Project structure matches PRD specifications

## Tasks

### T001: Initialize Go Module and Project Structure

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Create the Go module and establish the directory structure as defined in the PRD. This includes all major directories and placeholder files to establish the project skeleton.

#### Technical Details
```bash
go mod init github.com/user/poros
```

Directory structure to create:
```
poros/
├── cmd/poros/main.go
├── internal/
│   ├── probe/
│   ├── trace/
│   ├── enrich/
│   ├── network/
│   ├── output/
│   └── tui/
├── pkg/poros/
├── data/.gitkeep
├── scripts/
└── docs/
```

#### Files to Touch
- `go.mod` (new)
- `cmd/poros/main.go` (new)
- `internal/probe/.gitkeep` (new)
- `internal/trace/.gitkeep` (new)
- `internal/enrich/.gitkeep` (new)
- `internal/network/.gitkeep` (new)
- `internal/output/.gitkeep` (new)
- `internal/tui/.gitkeep` (new)
- `pkg/poros/.gitkeep` (new)
- `data/.gitkeep` (new)
- `scripts/.gitkeep` (new)
- `docs/.gitkeep` (new)

#### Dependencies
- None (first task)

#### Success Criteria
- [ ] `go mod init` completes successfully
- [ ] All directories exist as specified
- [ ] Basic main.go compiles

---

### T002: Define Core Data Structures

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement the core data structures defined in the PRD: `Hop`, `ASNInfo`, `GeoInfo`, `TraceResult`, `TracerConfig`, and `ProbeResult`. These structures form the foundation for all trace operations.

#### Technical Details
```go
// internal/trace/hop.go
type Hop struct {
    Number      int           `json:"hop"`
    IP          net.IP        `json:"ip"`
    Hostname    string        `json:"hostname,omitempty"`
    ASN         *ASNInfo      `json:"asn,omitempty"`
    Geo         *GeoInfo      `json:"geo,omitempty"`
    RTTs        []float64     `json:"rtts"`
    AvgRTT      float64       `json:"avg_rtt"`
    MinRTT      float64       `json:"min_rtt"`
    MaxRTT      float64       `json:"max_rtt"`
    Jitter      float64       `json:"jitter"`
    LossPercent float64       `json:"loss_percent"`
    Responded   bool          `json:"responded"`
}

// internal/trace/types.go
type TraceResult struct { ... }
type TracerConfig struct { ... }
```

#### Files to Touch
- `internal/trace/hop.go` (new)
- `internal/trace/types.go` (new)
- `internal/trace/config.go` (new)
- `internal/enrich/types.go` (new) - ASNInfo, GeoInfo

#### Dependencies
- T001: Project structure must exist

#### Success Criteria
- [ ] All data structures compile without errors
- [ ] JSON tags are correctly defined
- [ ] Structures can be instantiated in tests
- [ ] Unit tests for data structure methods

---

### T003: Define Prober Interface

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Define the `Prober` interface that all probe implementations (ICMP, UDP, TCP, Paris) will implement. This ensures consistent behavior across different probe methods.

#### Technical Details
```go
// internal/probe/probe.go
type Prober interface {
    Probe(ctx context.Context, dest net.IP, ttl int) (*ProbeResult, error)
    Name() string
    RequiresRoot() bool
    Close() error
}

type ProbeResult struct {
    ResponseIP  net.IP
    RTT         time.Duration
    ICMPType    int
    ICMPCode    int
    Reached     bool
}

type ProbeMethod int
const (
    ProbeICMP ProbeMethod = iota
    ProbeUDP
    ProbeTCP
    ProbeParis
)
```

#### Files to Touch
- `internal/probe/probe.go` (new)
- `internal/probe/errors.go` (new)
- `internal/probe/result.go` (new)

#### Dependencies
- T001: Project structure must exist

#### Success Criteria
- [ ] Interface is well-documented
- [ ] ProbeMethod enum is defined
- [ ] Error types are defined
- [ ] Compiles successfully

---

### T004: Setup Build System (Makefile Enhancement)

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Enhance the existing Makefile to support Go build commands, cross-platform compilation, testing, linting, and common development tasks.

#### Technical Details
```makefile
.PHONY: build test lint clean

VERSION ?= $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X main.version=${VERSION}"

build:
	go build ${LDFLAGS} -o bin/poros ./cmd/poros

build-all:
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o bin/poros-linux-amd64 ./cmd/poros
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o bin/poros-darwin-amd64 ./cmd/poros
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o bin/poros-windows-amd64.exe ./cmd/poros

test:
	go test -v -race ./...

lint:
	golangci-lint run

clean:
	rm -rf bin/
```

#### Files to Touch
- `Makefile` (update)
- `build.bat` (update - Windows build script)
- `.golangci.yml` (new)

#### Dependencies
- T001: Project structure must exist

#### Success Criteria
- [ ] `make build` produces binary
- [ ] `make test` runs tests
- [ ] `make lint` checks code
- [ ] Cross-platform targets work

---

### T005: Setup CLI Framework (Cobra)

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Initialize Cobra CLI framework with root command, basic flags, and help text. Set up the foundation for all command-line argument parsing.

#### Technical Details
```go
// cmd/poros/main.go
var rootCmd = &cobra.Command{
    Use:   "poros [flags] <target>",
    Short: "Modern network path tracer",
    Long: `Poros (Πόρος) - A modern, cross-platform network path tracer
    
Features:
  - Multiple probe methods (ICMP, UDP, TCP, Paris)
  - Concurrent and sequential tracing modes
  - ASN and GeoIP enrichment
  - Interactive TUI mode`,
    RunE: runTrace,
}

// Basic flags
rootCmd.Flags().BoolP("icmp", "I", false, "Use ICMP probes")
rootCmd.Flags().BoolP("udp", "U", false, "Use UDP probes")
rootCmd.Flags().BoolP("tcp", "T", false, "Use TCP probes")
rootCmd.Flags().IntP("max-hops", "m", 30, "Maximum number of hops")
```

#### Files to Touch
- `cmd/poros/main.go` (update)
- `cmd/poros/root.go` (new)
- `cmd/poros/flags.go` (new)
- `cmd/poros/version.go` (new)

#### Dependencies
- T001: Project structure must exist
- T002: Core data structures for TracerConfig

#### Success Criteria
- [ ] `poros --help` shows usage
- [ ] `poros --version` shows version
- [ ] All PRD flags are defined
- [ ] Flags parse correctly

---

### T006: Add Core Dependencies

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Add all required Go dependencies to go.mod as specified in the PRD. Ensure dependencies are properly versioned and compatible.

#### Technical Details
```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get golang.org/x/net@latest
go get golang.org/x/sys@latest
go get github.com/oschwald/maxminddb-golang@latest
go get github.com/miekg/dns@latest
go get github.com/olekukonko/tablewriter@latest
go get github.com/fatih/color@latest
```

TUI dependencies (can be added later):
```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
```

#### Files to Touch
- `go.mod` (update)
- `go.sum` (generated)

#### Dependencies
- T001: Go module must be initialized

#### Success Criteria
- [ ] All dependencies resolve
- [ ] `go mod tidy` succeeds
- [ ] No version conflicts
- [ ] Dependencies are properly imported

---

## Performance Targets
- Project compiles in < 5 seconds
- Test suite runs in < 10 seconds (with no actual tests yet)
- Linting completes in < 30 seconds

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Dependency conflicts | Low | Medium | Use specific versions, test compatibility |
| Structure changes needed later | Medium | Low | Design for extensibility |
| Build system issues on Windows | Medium | Medium | Test build.bat early |

## Notes
- This is the foundational phase - all other features depend on this
- Keep the structure flexible for future additions
- Follow Go best practices for package naming and organization
- Consider adding GitHub Actions CI config in this phase
