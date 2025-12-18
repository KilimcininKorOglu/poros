# Feature 014: Release and Packaging

**Feature ID:** F014
**Feature Name:** Release and Packaging
**Priority:** P2 - HIGH
**Target Version:** v1.0.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Set up release infrastructure including GoReleaser configuration, GitHub Actions for CI/CD, package manager recipes (Homebrew, AUR), documentation finalization, and binary distribution.

## Goals
- Configure GoReleaser for automated releases
- Create GitHub Actions CI/CD pipeline
- Create Homebrew formula
- Create AUR package
- Finalize documentation

## Success Criteria
- [ ] All tasks completed (T090-T097)
- [ ] GitHub releases created automatically
- [ ] Homebrew installation works
- [ ] Documentation is complete
- [ ] All platforms have binaries

## Tasks

### T090: Configure GoReleaser

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Set up GoReleaser for automated cross-platform binary releases.

#### Technical Details
```yaml
# .goreleaser.yml
version: 1

before:
  hooks:
    - go mod tidy
    - go generate ./...

builds:
  - id: poros
    main: ./cmd/poros
    binary: poros
    env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - id: default
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}
    format_overrides:
      - goos: windows
        format: zip
    files:
      - LICENSE
      - README.md
      - docs/*

checksum:
  name_template: 'checksums.txt'

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
      - Merge pull request
      - Merge branch

brews:
  - name: poros
    repository:
      owner: user
      name: homebrew-tap
    homepage: https://github.com/user/poros
    description: Modern network path tracer
    license: MIT
    install: |
      bin.install "poros"
    test: |
      system "#{bin}/poros", "--version"

nfpms:
  - id: poros
    package_name: poros
    vendor: Poros
    homepage: https://github.com/user/poros
    maintainer: User <user@example.com>
    description: Modern network path tracer
    license: MIT
    formats:
      - deb
      - rpm
      - apk
    bindir: /usr/bin

snapcrafts:
  - name: poros
    publish: true
    summary: Modern network path tracer
    description: |
      Poros is a modern, cross-platform network path tracer with
      concurrent probing, ASN/GeoIP enrichment, and TUI support.
    grade: stable
    confinement: classic
    license: MIT
    apps:
      poros:
        command: poros
```

#### Files to Touch
- `.goreleaser.yml` (new)

#### Dependencies
- T073: Cross-platform builds working

#### Success Criteria
- [ ] `goreleaser check` passes
- [ ] Local release works
- [ ] All platforms included

---

### T091: Set Up GitHub Actions CI

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create GitHub Actions workflows for testing, linting, and releases.

#### Technical Details
```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main, master]
  pull_request:
    branches: [main, master]

jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ['1.21', '1.22']
    runs-on: ${{ matrix.os }}
    
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: ${{ matrix.go }}
      
      - name: Download dependencies
        run: go mod download
      
      - name: Run tests
        run: go test -v -race -coverprofile=coverage.txt ./...
      
      - name: Upload coverage
        uses: codecov/codecov-action@v4
        if: matrix.os == 'ubuntu-latest' && matrix.go == '1.22'

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  build:
    needs: [test, lint]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Build
        run: go build -v ./cmd/poros

# .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      
      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

#### Files to Touch
- `.github/workflows/ci.yml` (new)
- `.github/workflows/release.yml` (new)

#### Dependencies
- T090: GoReleaser config

#### Success Criteria
- [ ] CI runs on PRs
- [ ] Tests run on all platforms
- [ ] Release creates binaries

---

### T092: Create Homebrew Formula

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create and publish Homebrew formula for macOS/Linux installation.

#### Technical Details
```ruby
# Formula/poros.rb
class Poros < Formula
  desc "Modern network path tracer"
  homepage "https://github.com/user/poros"
  version "1.0.0"
  license "MIT"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/user/poros/releases/download/v#{version}/poros_#{version}_darwin_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    else
      url "https://github.com/user/poros/releases/download/v#{version}/poros_#{version}_darwin_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/user/poros/releases/download/v#{version}/poros_#{version}_linux_arm64.tar.gz"
      sha256 "PLACEHOLDER"
    else
      url "https://github.com/user/poros/releases/download/v#{version}/poros_#{version}_linux_amd64.tar.gz"
      sha256 "PLACEHOLDER"
    end
  end

  def install
    bin.install "poros"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/poros --version")
  end
end
```

#### Files to Touch
- `Formula/poros.rb` (new - in separate homebrew-tap repo)

#### Dependencies
- T091: GitHub releases working

#### Success Criteria
- [ ] `brew install user/tap/poros` works
- [ ] Version matches release
- [ ] Both architectures supported

---

### T093: Create AUR Package

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 0.5 days

#### Description
Create Arch User Repository (AUR) package for Arch Linux users.

#### Technical Details
```bash
# PKGBUILD
pkgname=poros
pkgver=1.0.0
pkgrel=1
pkgdesc='Modern network path tracer'
arch=('x86_64' 'aarch64')
url='https://github.com/user/poros'
license=('MIT')
makedepends=('go')
source=("${pkgname}-${pkgver}.tar.gz::https://github.com/user/poros/archive/v${pkgver}.tar.gz")
sha256sums=('PLACEHOLDER')

build() {
    cd "${pkgname}-${pkgver}"
    export CGO_ENABLED=0
    go build -ldflags "-s -w -X main.version=${pkgver}" -o poros ./cmd/poros
}

package() {
    cd "${pkgname}-${pkgver}"
    install -Dm755 poros "${pkgdir}/usr/bin/poros"
    install -Dm644 LICENSE "${pkgdir}/usr/share/licenses/${pkgname}/LICENSE"
    install -Dm644 README.md "${pkgdir}/usr/share/doc/${pkgname}/README.md"
}
```

```bash
# .SRCINFO
pkgbase = poros
	pkgdesc = Modern network path tracer
	pkgver = 1.0.0
	pkgrel = 1
	url = https://github.com/user/poros
	arch = x86_64
	arch = aarch64
	license = MIT
	makedepends = go
	source = poros-1.0.0.tar.gz::https://github.com/user/poros/archive/v1.0.0.tar.gz
	sha256sums = PLACEHOLDER

pkgname = poros
```

#### Files to Touch
- `aur/PKGBUILD` (new)
- `aur/.SRCINFO` (new)

#### Dependencies
- T091: GitHub releases working

#### Success Criteria
- [ ] `yay -S poros` works
- [ ] Package builds correctly
- [ ] Version updates work

---

### T094: Create Docker Image

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 0.5 days

#### Description
Create Docker image for containerized usage.

#### Technical Details
```dockerfile
# Dockerfile
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /poros ./cmd/poros

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /poros /usr/local/bin/poros

ENTRYPOINT ["poros"]
```

```yaml
# .github/workflows/docker.yml
name: Docker

on:
  push:
    tags:
      - 'v*'

jobs:
  docker:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3
      
      - name: Login to DockerHub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}
      
      - name: Build and push
        uses: docker/build-push-action@v5
        with:
          context: .
          push: true
          tags: |
            user/poros:latest
            user/poros:${{ github.ref_name }}
          platforms: linux/amd64,linux/arm64
```

#### Files to Touch
- `Dockerfile` (new)
- `.github/workflows/docker.yml` (new)
- `.dockerignore` (new)

#### Dependencies
- T091: CI/CD setup

#### Success Criteria
- [ ] Docker image builds
- [ ] Multi-arch support
- [ ] `docker run poros google.com` works

---

### T095: Finalize README

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Complete README.md with all features, installation instructions, and examples.

#### Technical Details
```markdown
# 🔱 Poros

Modern, cross-platform network path tracer with concurrent probing, 
ASN/GeoIP enrichment, and TUI support.

[![CI](https://github.com/user/poros/workflows/CI/badge.svg)](https://github.com/user/poros/actions)
[![Release](https://img.shields.io/github/v/release/user/poros)](https://github.com/user/poros/releases)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## Features

- 🚀 **Fast**: Concurrent probing completes 30-hop traces in <5 seconds
- 🌐 **Multi-protocol**: ICMP, UDP, TCP SYN, and Paris traceroute
- 📊 **Rich data**: ASN, GeoIP, and reverse DNS enrichment
- 🖥️ **Interactive TUI**: Real-time visualization with latency graphs
- 📄 **Multiple outputs**: Text, JSON, CSV, and HTML reports
- 💻 **Cross-platform**: Linux, macOS, and Windows

## Installation

### Homebrew (macOS/Linux)
\`\`\`bash
brew install user/tap/poros
\`\`\`

### Go Install
\`\`\`bash
go install github.com/user/poros@latest
\`\`\`

### Binary Download
Download from [Releases](https://github.com/user/poros/releases)

## Quick Start

\`\`\`bash
# Basic trace
sudo poros google.com

# Verbose table output
sudo poros -v google.com

# JSON output
sudo poros --json google.com

# Interactive TUI
sudo poros --tui google.com

# TCP probe on port 443
sudo poros -T --port 443 google.com
\`\`\`

## Usage

\`\`\`
poros [flags] <target>

Flags:
  -I, --icmp           Use ICMP probes (default)
  -U, --udp            Use UDP probes
  -T, --tcp            Use TCP SYN probes
      --paris          Use Paris traceroute algorithm
  -m, --max-hops int   Maximum number of hops (default 30)
  -q, --queries int    Number of probes per hop (default 3)
  -w, --timeout duration   Per-probe timeout (default 3s)
  -v, --verbose        Show detailed table output
  -j, --json           Output in JSON format
      --csv            Output in CSV format
      --html file      Generate HTML report
  -t, --tui            Interactive TUI mode
      --no-enrich      Disable enrichment
  -h, --help           Show help
\`\`\`

## Requirements

- Linux: root or CAP_NET_RAW capability
- macOS: root (sudo)
- Windows: Administrator

## License

MIT License - see [LICENSE](LICENSE) for details.
```

#### Files to Touch
- `README.md` (update)

#### Dependencies
- All features implemented

#### Success Criteria
- [ ] Installation instructions complete
- [ ] All features documented
- [ ] Examples work

---

### T096: Create Man Page

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 0.5 days

#### Description
Create Unix man page for poros.

#### Technical Details
```man
.TH POROS 1 "December 2025" "poros 1.0.0" "User Commands"
.SH NAME
poros \- modern network path tracer
.SH SYNOPSIS
.B poros
[\fIOPTIONS\fR] \fItarget\fR
.SH DESCRIPTION
.B poros
is a modern, cross-platform network path tracer with concurrent 
probing, ASN/GeoIP enrichment, and TUI support.
.SH OPTIONS
.TP
.BR \-I ", " \-\-icmp
Use ICMP Echo probes (default)
.TP
.BR \-U ", " \-\-udp
Use UDP probes
.TP
.BR \-T ", " \-\-tcp
Use TCP SYN probes
.TP
.BR \-\-paris
Use Paris traceroute algorithm
.TP
.BR \-m ", " \-\-max\-hops =\fIN\fR
Set maximum number of hops (default: 30)
.TP
.BR \-q ", " \-\-queries =\fIN\fR
Set number of probes per hop (default: 3)
.TP
.BR \-w ", " \-\-timeout =\fIDURATION\fR
Set per-probe timeout (default: 3s)
.TP
.BR \-v ", " \-\-verbose
Show detailed table output
.TP
.BR \-j ", " \-\-json
Output in JSON format
.TP
.BR \-\-csv
Output in CSV format
.TP
.BR \-t ", " \-\-tui
Run in interactive TUI mode
.SH EXAMPLES
.TP
Basic trace:
.B sudo poros google.com
.TP
TCP probe to HTTPS:
.B sudo poros \-T \-\-port 443 google.com
.TP
JSON output:
.B sudo poros \-\-json google.com | jq
.SH AUTHOR
Written by User.
.SH SEE ALSO
.BR traceroute (1),
.BR mtr (1)
```

#### Files to Touch
- `docs/poros.1` (new)

#### Dependencies
- T095: README complete

#### Success Criteria
- [ ] Man page renders correctly
- [ ] All options documented
- [ ] Examples included

---

### T097: Final Testing and Quality Assurance

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Comprehensive testing across all platforms and features before v1.0.0 release.

#### Technical Details
```markdown
# Release Checklist

## Functionality Testing
- [ ] ICMP trace works on Linux
- [ ] ICMP trace works on macOS
- [ ] ICMP trace works on Windows
- [ ] UDP trace works
- [ ] TCP trace works
- [ ] Paris mode works
- [ ] Concurrent mode is faster than sequential
- [ ] Enrichment (rDNS, ASN, GeoIP) works
- [ ] TUI displays correctly
- [ ] JSON output is valid
- [ ] CSV output is valid
- [ ] HTML report renders

## Platform Testing
- [ ] Linux amd64
- [ ] Linux arm64
- [ ] macOS amd64 (Intel)
- [ ] macOS arm64 (Apple Silicon)
- [ ] Windows amd64

## Installation Testing
- [ ] Homebrew installation
- [ ] Go install
- [ ] Binary download
- [ ] Docker run

## Documentation Review
- [ ] README is accurate
- [ ] CLI help is complete
- [ ] Man page is correct
- [ ] PLATFORMS.md is accurate

## Performance Verification
- [ ] 30-hop trace < 5s (concurrent)
- [ ] Memory usage < 50MB
- [ ] Binary size < 15MB
```

#### Files to Touch
- `docs/RELEASE_CHECKLIST.md` (new)

#### Dependencies
- All features complete

#### Success Criteria
- [ ] All checklist items pass
- [ ] No critical bugs
- [ ] Performance targets met

---

## Performance Targets
- Release build time: < 5 minutes
- Docker build time: < 3 minutes
- Download size: < 5MB compressed

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Build failures | Low | High | Test locally first |
| Package manager issues | Medium | Medium | Test installation |
| Documentation gaps | Medium | Low | Review checklist |

## Notes
- Tag releases with semantic versioning (v1.0.0)
- Create GitHub release notes from changelog
- Consider scoop package for Windows
- Consider snap package for Linux
