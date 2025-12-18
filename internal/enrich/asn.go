package enrich

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// ASNInfo contains ASN information for an IP address.
type ASNInfo struct {
	Number  int
	Org     string
	Country string
}

// ASNLookup defines the interface for ASN lookups.
type ASNLookup interface {
	Lookup(ctx context.Context, ip net.IP) (*ASNInfo, error)
	Close() error
}

// TeamCymruASN implements ASN lookup using Team Cymru's DNS service.
// This is a free service that doesn't require any database files.
// See: https://www.team-cymru.com/ip-asn-mapping
type TeamCymruASN struct {
	timeout time.Duration
	cache   *Cache
}

// TeamCymruConfig holds configuration for Team Cymru ASN lookups.
type TeamCymruConfig struct {
	Timeout   time.Duration
	CacheSize int
	CacheTTL  time.Duration
}

// DefaultTeamCymruConfig returns default configuration.
func DefaultTeamCymruConfig() TeamCymruConfig {
	return TeamCymruConfig{
		Timeout:   3 * time.Second,
		CacheSize: 1000,
		CacheTTL:  1 * time.Hour, // ASN data changes infrequently
	}
}

// NewTeamCymruASN creates a new Team Cymru ASN resolver.
func NewTeamCymruASN(config TeamCymruConfig) *TeamCymruASN {
	if config.Timeout == 0 {
		config.Timeout = 3 * time.Second
	}

	var cache *Cache
	if config.CacheSize > 0 {
		cache = NewCache(config.CacheSize, config.CacheTTL)
	}

	return &TeamCymruASN{
		timeout: config.Timeout,
		cache:   cache,
	}
}

// Lookup performs an ASN lookup using Team Cymru's DNS service.
// For IPv4: query <reversed-ip>.origin.asn.cymru.com
// For IPv6: query <nibble-reversed>.origin6.asn.cymru.com
func (t *TeamCymruASN) Lookup(ctx context.Context, ip net.IP) (*ASNInfo, error) {
	if ip == nil {
		return nil, nil
	}

	// Skip private/special IPs
	if isPrivateIP(ip) {
		return nil, nil
	}

	ipStr := ip.String()

	// Check cache
	if t.cache != nil {
		if cached, ok := t.cache.Get(ipStr); ok {
			if cached == nil {
				return nil, nil
			}
			return cached.(*ASNInfo), nil
		}
	}

	// Build DNS query
	var query string
	if ip4 := ip.To4(); ip4 != nil {
		// IPv4: reverse octets
		query = fmt.Sprintf("%d.%d.%d.%d.origin.asn.cymru.com",
			ip4[3], ip4[2], ip4[1], ip4[0])
	} else {
		// IPv6: reverse nibbles
		query = reverseIPv6Nibbles(ip) + ".origin6.asn.cymru.com"
	}

	// Create context with timeout
	lookupCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// Query TXT record
	records, err := net.DefaultResolver.LookupTXT(lookupCtx, query)
	if err != nil {
		// Cache negative result
		if t.cache != nil {
			t.cache.Set(ipStr, nil)
		}
		return nil, nil
	}

	if len(records) == 0 {
		if t.cache != nil {
			t.cache.Set(ipStr, nil)
		}
		return nil, nil
	}

	// Parse response: "ASN | IP/Prefix | Country | Registry | Date"
	info := parseTeamCymruResponse(records[0])
	if info == nil {
		if t.cache != nil {
			t.cache.Set(ipStr, nil)
		}
		return nil, nil
	}

	// Get AS name if we have an ASN
	if info.Number > 0 {
		info.Org = t.lookupASName(ctx, info.Number)
	}

	// Cache result
	if t.cache != nil {
		t.cache.Set(ipStr, info)
	}

	return info, nil
}

// lookupASName queries Team Cymru for the AS name.
func (t *TeamCymruASN) lookupASName(ctx context.Context, asn int) string {
	query := fmt.Sprintf("AS%d.asn.cymru.com", asn)

	lookupCtx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	records, err := net.DefaultResolver.LookupTXT(lookupCtx, query)
	if err != nil || len(records) == 0 {
		return ""
	}

	// Parse: "ASN | Country | Registry | Date | Name"
	parts := strings.Split(records[0], "|")
	if len(parts) >= 5 {
		return strings.TrimSpace(parts[4])
	}

	return ""
}

// Close releases resources.
func (t *TeamCymruASN) Close() error {
	if t.cache != nil {
		t.cache.Clear()
	}
	return nil
}

// parseTeamCymruResponse parses the TXT record response.
func parseTeamCymruResponse(txt string) *ASNInfo {
	// Format: "ASN | IP/Prefix | Country | Registry | Date"
	parts := strings.Split(txt, "|")
	if len(parts) < 3 {
		return nil
	}

	asnStr := strings.TrimSpace(parts[0])
	country := strings.TrimSpace(parts[2])

	asn, err := strconv.Atoi(asnStr)
	if err != nil {
		return nil
	}

	return &ASNInfo{
		Number:  asn,
		Country: country,
	}
}

// reverseIPv6Nibbles reverses the nibbles of an IPv6 address for DNS lookup.
func reverseIPv6Nibbles(ip net.IP) string {
	// Expand to full 16 bytes
	ip16 := ip.To16()
	if ip16 == nil {
		return ""
	}

	var nibbles []string
	for i := len(ip16) - 1; i >= 0; i-- {
		b := ip16[i]
		nibbles = append(nibbles, fmt.Sprintf("%x", b&0x0f))
		nibbles = append(nibbles, fmt.Sprintf("%x", b>>4))
	}

	return strings.Join(nibbles, ".")
}

// isPrivateIP checks if an IP is private/reserved.
func isPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	// Check for loopback
	if ip.IsLoopback() {
		return true
	}

	// Check for private ranges
	if ip.IsPrivate() {
		return true
	}

	// Check for link-local
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}

	return false
}
