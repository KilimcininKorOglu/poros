# Poros

**Poros** (Greek: Πόρος - "path, passage") is a modern, cross-platform network path tracer written in Go.

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Features

- **Multiple Probe Methods**: ICMP (default), UDP, TCP SYN, Paris traceroute
- **Concurrent Probing**: Fast parallel probing for quick results
- **Rich Enrichment**: Reverse DNS, ASN lookup, GeoIP information
- **Multiple Output Formats**: Text, verbose table, JSON, CSV, HTML reports
- **Interactive TUI**: Beautiful terminal UI with real-time updates
- **Cross-Platform**: Linux, macOS, Windows support

## Installation

### Using Go

```bash
go install github.com/KilimcininKorOglu/poros/cmd/poros@latest
```

### From Source

```bash
git clone https://github.com/KilimcininKorOglu/poros.git
cd poros
make build
```

### Using Install Script (Linux/macOS)

```bash
curl -sSL https://raw.githubusercontent.com/KilimcininKorOglu/poros/master/scripts/install.sh | bash
```

## Quick Start

```bash
# Basic trace using ICMP
poros google.com

# Use UDP probes
poros -U google.com

# Use TCP SYN probes to port 443
poros -T -p 443 google.com

# Verbose table output
poros -v google.com

# JSON output
poros --json google.com

# Interactive TUI mode
poros --tui google.com

# Generate HTML report
poros --html report.html google.com

# Paris traceroute (load-balancer friendly)
poros --paris google.com
```

## Command Line Options

```
Usage:
  poros [flags] <target>

Probe Methods:
  -I, --icmp           Use ICMP Echo probes (default)
  -U, --udp            Use UDP probes
  -T, --tcp            Use TCP SYN probes
      --paris          Use Paris traceroute algorithm

Trace Parameters:
  -m, --max-hops int   Maximum number of hops (default 30)
  -q, --queries int    Number of probes per hop (default 3)
  -w, --timeout duration  Probe timeout (default 3s)
  -f, --first-hop int  Start from specified hop (default 1)
      --sequential     Use sequential mode (slower but reliable)

Network Settings:
  -4, --ipv4           Use IPv4 only
  -6, --ipv6           Use IPv6 only
  -p, --port int       Destination port for UDP/TCP (default 33434)
  -i, --interface string  Network interface to use
  -s, --source string  Source IP address

Output Formats:
  -v, --verbose        Show detailed table output
  -j, --json           Output in JSON format
      --csv            Output in CSV format
      --html string    Generate HTML report to file
  -t, --tui            Interactive TUI mode
      --no-color       Disable colored output

Enrichment:
      --no-enrich      Disable all enrichment
      --no-rdns        Disable reverse DNS lookups
      --no-asn         Disable ASN lookups
      --no-geoip       Disable GeoIP lookups
```

## Output Examples

### Classic Text Output
```
traceroute to google.com (142.250.185.238), 30 hops max

  1  router.local (192.168.1.1)  1.234 ms  1.456 ms  1.123 ms
  2  10.0.0.1  5.678 ms  5.432 ms  5.555 ms  [AS15169 Google]
  3  * * *
  4  dns.google (8.8.8.8)  12.345 ms  12.123 ms  12.456 ms

Trace complete. 4 hops, 12.31 ms total
```

### JSON Output
```json
{
  "target": "google.com",
  "resolved_ip": "142.250.185.238",
  "probe_method": "icmp",
  "completed": true,
  "hops": [
    {
      "hop": 1,
      "ip": "192.168.1.1",
      "hostname": "router.local",
      "avg_rtt_ms": 1.271,
      "loss_percent": 0
    }
  ],
  "summary": {
    "total_hops": 4,
    "total_time_ms": 12.31
  }
}
```

## Requirements

- **Go 1.21+** (for building from source)
- **Root/Administrator privileges** for raw socket access

### Platform Notes

| Platform | Privilege Required | Notes |
|----------|-------------------|-------|
| Linux | `sudo` or `CAP_NET_RAW` | Use `setcap cap_net_raw+ep ./poros` |
| macOS | `sudo` | Required for ICMP |
| Windows | Run as Administrator | Required for raw sockets |

## Development

```bash
# Build
make build

# Run tests
make test

# Run with coverage
make test-coverage

# Build for all platforms
make build-all

# Lint code
make lint

# Format code
make fmt
```

## Documentation

For detailed documentation, see the [docs](docs/) directory.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by traditional `traceroute` and `mtr`
- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- TUI powered by [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- ASN data from [Team Cymru](https://www.team-cymru.com/)
- GeoIP from [ip-api.com](https://ip-api.com/)
