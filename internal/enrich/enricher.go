package enrich

import (
	"context"
	"net"
	"sync"
)

// Enricher performs IP enrichment with rDNS, ASN, and GeoIP data.
type Enricher struct {
	config EnricherConfig
	rdns   *RDNSResolver
	asn    ASNLookup
	geo    GeoLookup
}

// EnricherConfig holds configuration for the enricher.
type EnricherConfig struct {
	EnableRDNS  bool
	EnableASN   bool
	EnableGeoIP bool

	// Timeouts
	RDNSTimeout  int // milliseconds
	ASNTimeout   int // milliseconds
	GeoIPTimeout int // milliseconds

	// Cache settings
	CacheSize int
}

// DefaultEnricherConfig returns default enricher configuration.
func DefaultEnricherConfig() EnricherConfig {
	return EnricherConfig{
		EnableRDNS:   true,
		EnableASN:    true,
		EnableGeoIP:  true,
		RDNSTimeout:  2000,
		ASNTimeout:   3000,
		GeoIPTimeout: 5000,
		CacheSize:    1000,
	}
}

// NewEnricher creates a new enricher with the given configuration.
func NewEnricher(config EnricherConfig) *Enricher {
	e := &Enricher{
		config: config,
	}

	if config.EnableRDNS {
		e.rdns = NewRDNSResolver(DefaultRDNSConfig())
	}

	if config.EnableASN {
		e.asn = NewTeamCymruASN(DefaultTeamCymruConfig())
	}

	if config.EnableGeoIP {
		e.geo = NewIPAPIGeo(DefaultIPAPIConfig())
	}

	return e
}

// EnrichmentResult contains the results of IP enrichment.
type EnrichmentResult struct {
	Hostname string
	ASN      *ASNInfo
	Geo      *GeoInfo
}

// EnrichIP enriches a single IP with additional information.
func (e *Enricher) EnrichIP(ctx context.Context, ip net.IP) *EnrichmentResult {
	if ip == nil {
		return nil
	}

	result := &EnrichmentResult{}
	var wg sync.WaitGroup

	// Reverse DNS
	if e.config.EnableRDNS && e.rdns != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hostname, _ := e.rdns.Lookup(ctx, ip)
			result.Hostname = hostname
		}()
	}

	// ASN
	if e.config.EnableASN && e.asn != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.ASN, _ = e.asn.Lookup(ctx, ip)
		}()
	}

	// GeoIP
	if e.config.EnableGeoIP && e.geo != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result.Geo, _ = e.geo.Lookup(ctx, ip)
		}()
	}

	wg.Wait()
	return result
}

// EnrichIPs enriches multiple IPs concurrently and returns a map of results.
func (e *Enricher) EnrichIPs(ctx context.Context, ips []net.IP) map[string]*EnrichmentResult {
	results := make(map[string]*EnrichmentResult)
	if len(ips) == 0 {
		return results
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // Limit concurrency

	// Deduplicate IPs
	seen := make(map[string]bool)
	uniqueIPs := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		if ip != nil {
			ipStr := ip.String()
			if !seen[ipStr] {
				seen[ipStr] = true
				uniqueIPs = append(uniqueIPs, ip)
			}
		}
	}

	for _, ip := range uniqueIPs {
		wg.Add(1)
		go func(ip net.IP) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			result := e.EnrichIP(ctx, ip)

			mu.Lock()
			results[ip.String()] = result
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	return results
}

// Close releases resources held by the enricher.
func (e *Enricher) Close() error {
	if e.rdns != nil {
		e.rdns.Close()
	}
	if e.asn != nil {
		e.asn.Close()
	}
	if e.geo != nil {
		e.geo.Close()
	}
	return nil
}
