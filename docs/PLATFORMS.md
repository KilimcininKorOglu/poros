# Platform Support

Poros supports Linux, macOS, and Windows with platform-specific optimizations for raw socket access.

## Quick Reference

| Platform | ICMP | UDP | TCP | Paris | Privileges Required |
|----------|------|-----|-----|-------|---------------------|
| Linux | ✅ | ✅ | ✅ | ✅ | root or CAP_NET_RAW |
| macOS | ✅ | ✅ | ✅ | ✅ | root (sudo) |
| Windows | ✅ | ✅ | ✅ | ✅ | Administrator |

## Linux

### Running with Privileges

**Option 1: Using sudo**
```bash
sudo poros google.com
```

**Option 2: Set capabilities (recommended for regular use)**
```bash
# Set capability once
sudo setcap cap_net_raw+ep /path/to/poros

# Now run without sudo
poros google.com
```

**Option 3: UDP high-port mode (no privileges needed)**
```bash
poros -U google.com
```

### Socket Implementation
- ICMP: Raw socket with `SOCK_RAW, IPPROTO_ICMP`
- UDP: Standard UDP socket with `IP_TTL` setsockopt
- TCP: Raw socket with `SOCK_RAW, IPPROTO_TCP`

### Known Issues
- Some container environments may restrict raw sockets
- WSL1 has limited raw socket support (WSL2 works)

## macOS

### Running with Privileges

```bash
sudo poros google.com
```

### Socket Implementation
- Uses `golang.org/x/net/icmp` package
- ICMP: "ip4:icmp" or "ip6:ipv6-icmp" network
- UDP/TCP: Standard sockets with TTL manipulation

### Known Issues
- Requires root for all ICMP operations
- TCP SYN probes may be filtered by firewall
- Big Sur and later have stricter raw socket policies

### Firewall Configuration
If probes are being blocked:
```bash
# Check firewall status
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --getglobalstate

# Temporarily disable (for testing)
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --setglobalstate off
```

## Windows

### Running with Privileges

1. Right-click Command Prompt or PowerShell
2. Select "Run as Administrator"
3. Run poros:
```powershell
.\poros.exe google.com
```

### Socket Implementation
- Uses `golang.org/x/net/icmp` with Windows adaptations
- ICMP: Raw socket or Windows ICMP API fallback
- UDP: Winsock2 UDP socket with TTL option
- TCP: Raw socket (requires Administrator)

### Known Issues
- Windows Defender may flag raw socket creation
- Some antivirus software may interfere with probes
- IPv6 support depends on Windows version

### Windows Defender
If Windows Defender is blocking:
1. Open Windows Security
2. Go to "Virus & threat protection"
3. Under "Exclusions", add poros.exe

## Cross-Platform Building

### Build for All Platforms
```bash
make build-all
```

This creates:
- `bin/poros-linux-amd64`
- `bin/poros-linux-arm64`
- `bin/poros-darwin-amd64`
- `bin/poros-darwin-arm64`
- `bin/poros-windows-amd64.exe`

### Build for Specific Platform
```bash
# Linux AMD64
GOOS=linux GOARCH=amd64 go build -o poros-linux ./cmd/poros

# macOS ARM64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o poros-macos ./cmd/poros

# Windows
GOOS=windows GOARCH=amd64 go build -o poros.exe ./cmd/poros
```

## Privilege Detection

Poros automatically detects if it has sufficient privileges:

```
$ poros google.com
Error: permission denied

Raw socket requires elevated privileges.

Solutions:
  1. Run with sudo: sudo poros google.com
  2. Set capabilities: sudo setcap cap_net_raw+ep ./poros
```

## IPv6 Support

IPv6 is supported on all platforms:

```bash
poros -6 ipv6.google.com
```

Note: IPv6 requires:
- Linux: IPv6 enabled in kernel
- macOS: IPv6 enabled in network settings
- Windows: IPv6 protocol installed and enabled

## Performance Notes

| Platform | Cold Start | 30-hop Trace |
|----------|------------|--------------|
| Linux | ~50ms | ~3s |
| macOS | ~75ms | ~4s |
| Windows | ~100ms | ~5s |

Performance may vary based on:
- Network conditions
- Firewall configuration
- System load
- Antivirus software (Windows)

## Troubleshooting

### "Permission denied" errors
- Ensure you have appropriate privileges (see above)
- Check if raw sockets are allowed in your environment

### "Network unreachable" errors
- Check network connectivity: `ping google.com`
- Verify DNS resolution: `nslookup google.com`
- Check firewall settings

### Timeouts on all hops
- Some networks block ICMP; try UDP: `poros -U target`
- Some networks block UDP; try TCP: `poros -T -p 80 target`
- Check if outbound traffic is allowed

### Inconsistent results
- Use Paris mode for load-balanced paths: `poros --paris target`
- Increase probe count: `poros -q 5 target`
- Use sequential mode: `poros --sequential target`
