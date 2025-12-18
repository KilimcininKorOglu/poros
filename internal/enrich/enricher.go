package enrich

import (
	"context"
	"net"
	"sync"
)

// Enricher performs IP enrichment with rDNS, ASN, and GeoIP data.
type Enricher struct {
	config   EnricherConfig
	rdns     *RDNSResolver
	asn      ASNLookup
	geo      GeoLookup
	maxmind  *MaxMindDB // Optional MaxMind database for offline/faster lookups
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

// NewEnricherWithMaxMind creates a new enricher with MaxMind database support.
// If MaxMind is configured and databases are available, they are used for ASN/GeoIP.
// Otherwise, falls back to online APIs (Team Cymru, ip-api.com).
func NewEnricherWithMaxMind(config EnricherConfig, maxmindDB *MaxMindDB) *Enricher {
	e := &Enricher{
		config:  config,
		maxmind: maxmindDB,
	}

	if config.EnableRDNS {
		e.rdns = NewRDNSResolver(DefaultRDNSConfig())
	}

	// Only create API lookups if MaxMind doesn't have the data
	if config.EnableASN {
		if maxmindDB == nil || !maxmindDB.HasASN() {
			e.asn = NewTeamCymruASN(DefaultTeamCymruConfig())
		}
	}

	if config.EnableGeoIP {
		if maxmindDB == nil || !maxmindDB.HasGeo() {
			e.geo = NewIPAPIGeo(DefaultIPAPIConfig())
		}
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
	var mu sync.Mutex

	// Reverse DNS (always use system DNS)
	if e.config.EnableRDNS && e.rdns != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hostname, _ := e.rdns.Lookup(ctx, ip)
			mu.Lock()
			result.Hostname = hostname
			mu.Unlock()
		}()
	}

	// ASN - try MaxMind first, then fall back to API
	if e.config.EnableASN {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var asn *ASNInfo

			// Try MaxMind first
			if e.maxmind != nil && e.maxmind.HasASN() {
				asn, _ = e.maxmind.LookupASN(ip)
			}

			// Fall back to API if MaxMind didn't have data
			if asn == nil && e.asn != nil {
				asn, _ = e.asn.Lookup(ctx, ip)
			}

			mu.Lock()
			result.ASN = asn
			mu.Unlock()
		}()
	}

	// GeoIP - try MaxMind first, then fall back to API
	if e.config.EnableGeoIP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var geo *GeoInfo

			// Try MaxMind first
			if e.maxmind != nil && e.maxmind.HasGeo() {
				geo, _ = e.maxmind.LookupGeo(ip)
			}

			// Fall back to API if MaxMind didn't have data
			if geo == nil && e.geo != nil {
				geo, _ = e.geo.Lookup(ctx, ip)
			}

			mu.Lock()
			result.Geo = geo
			mu.Unlock()
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
	if e.maxmind != nil {
		e.maxmind.Close()
	}
	return nil
}
