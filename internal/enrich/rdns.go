// Package enrich provides IP enrichment functionality including
// reverse DNS, ASN lookups, and GeoIP information.
package enrich

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// RDNSResolver performs reverse DNS lookups.
type RDNSResolver struct {
	timeout time.Duration
	cache   *Cache
	mu      sync.RWMutex
}

// RDNSConfig holds configuration for the rDNS resolver.
type RDNSConfig struct {
	Timeout    time.Duration
	CacheSize  int
	CacheTTL   time.Duration
	MaxRetries int
}

// DefaultRDNSConfig returns default rDNS configuration.
func DefaultRDNSConfig() RDNSConfig {
	return RDNSConfig{
		Timeout:    2 * time.Second,
		CacheSize:  1000,
		CacheTTL:   5 * time.Minute,
		MaxRetries: 1,
	}
}

// NewRDNSResolver creates a new reverse DNS resolver.
func NewRDNSResolver(config RDNSConfig) *RDNSResolver {
	if config.Timeout == 0 {
		config.Timeout = 2 * time.Second
	}

	var cache *Cache
	if config.CacheSize > 0 {
		cache = NewCache(config.CacheSize, config.CacheTTL)
	}

	return &RDNSResolver{
		timeout: config.Timeout,
		cache:   cache,
	}
}

// Lookup performs a reverse DNS lookup for the given IP address.
func (r *RDNSResolver) Lookup(ctx context.Context, ip net.IP) (string, error) {
	if ip == nil {
		return "", nil
	}

	ipStr := ip.String()

	// Check cache first
	if r.cache != nil {
		if cached, ok := r.cache.Get(ipStr); ok {
			return cached.(string), nil
		}
	}

	// Create context with timeout
	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	// Perform lookup
	names, err := net.DefaultResolver.LookupAddr(lookupCtx, ipStr)
	if err != nil {
		// Cache negative result briefly to avoid repeated failures
		if r.cache != nil {
			r.cache.Set(ipStr, "")
		}
		return "", nil // Return empty string, not error (DNS failures are common)
	}

	hostname := ""
	if len(names) > 0 {
		// Remove trailing dot from FQDN
		hostname = strings.TrimSuffix(names[0], ".")
	}

	// Cache result
	if r.cache != nil {
		r.cache.Set(ipStr, hostname)
	}

	return hostname, nil
}

// LookupBatch performs reverse DNS lookups for multiple IPs concurrently.
func (r *RDNSResolver) LookupBatch(ctx context.Context, ips []net.IP) map[string]string {
	results := make(map[string]string)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Limit concurrency
	sem := make(chan struct{}, 10)

	for _, ip := range ips {
		if ip == nil {
			continue
		}

		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			hostname, _ := r.Lookup(ctx, ip)

			mu.Lock()
			results[ip.String()] = hostname
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	return results
}

// Close releases resources held by the resolver.
func (r *RDNSResolver) Close() error {
	if r.cache != nil {
		r.cache.Clear()
	}
	return nil
}
