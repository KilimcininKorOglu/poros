package enrich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// GeoInfo contains geographic information for an IP address.
type GeoInfo struct {
	Country     string
	CountryCode string
	City        string
	Region      string
	Latitude    float64
	Longitude   float64
	Timezone    string
}

// GeoLookup defines the interface for GeoIP lookups.
type GeoLookup interface {
	Lookup(ctx context.Context, ip net.IP) (*GeoInfo, error)
	Close() error
}

// IPAPIGeo implements GeoIP lookup using the free ip-api.com service.
// Rate limit: 45 requests per minute (free tier).
type IPAPIGeo struct {
	client  *http.Client
	timeout time.Duration
	cache   *Cache
}

// IPAPIConfig holds configuration for ip-api.com lookups.
type IPAPIConfig struct {
	Timeout   time.Duration
	CacheSize int
	CacheTTL  time.Duration
}

// DefaultIPAPIConfig returns default configuration.
func DefaultIPAPIConfig() IPAPIConfig {
	return IPAPIConfig{
		Timeout:   5 * time.Second,
		CacheSize: 1000,
		CacheTTL:  24 * time.Hour, // GeoIP data is relatively stable
	}
}

// NewIPAPIGeo creates a new ip-api.com GeoIP resolver.
func NewIPAPIGeo(config IPAPIConfig) *IPAPIGeo {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	var cache *Cache
	if config.CacheSize > 0 {
		cache = NewCache(config.CacheSize, config.CacheTTL)
	}

	return &IPAPIGeo{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		timeout: config.Timeout,
		cache:   cache,
	}
}

// ipAPIResponse represents the JSON response from ip-api.com.
type ipAPIResponse struct {
	Status      string  `json:"status"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Timezone    string  `json:"timezone"`
	Message     string  `json:"message"`
}

// Lookup performs a GeoIP lookup using ip-api.com.
func (g *IPAPIGeo) Lookup(ctx context.Context, ip net.IP) (*GeoInfo, error) {
	if ip == nil {
		return nil, nil
	}

	// Skip private IPs
	if isPrivateIP(ip) {
		return nil, nil
	}

	ipStr := ip.String()

	// Check cache
	if g.cache != nil {
		if cached, ok := g.cache.Get(ipStr); ok {
			if cached == nil {
				return nil, nil
			}
			return cached.(*GeoInfo), nil
		}
	}

	// Build request
	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,message,country,countryCode,region,regionName,city,lat,lon,timezone", ipStr)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Make request
	resp, err := g.client.Do(req)
	if err != nil {
		// Cache negative result briefly
		if g.cache != nil {
			g.cache.SetWithTTL(ipStr, nil, 5*time.Minute)
		}
		return nil, nil
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil
	}

	// Parse JSON
	var apiResp ipAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil
	}

	if apiResp.Status != "success" {
		// Cache negative result
		if g.cache != nil {
			g.cache.SetWithTTL(ipStr, nil, 5*time.Minute)
		}
		return nil, nil
	}

	info := &GeoInfo{
		Country:     apiResp.Country,
		CountryCode: apiResp.CountryCode,
		City:        apiResp.City,
		Region:      apiResp.RegionName,
		Latitude:    apiResp.Lat,
		Longitude:   apiResp.Lon,
		Timezone:    apiResp.Timezone,
	}

	// Cache result
	if g.cache != nil {
		g.cache.Set(ipStr, info)
	}

	return info, nil
}

// Close releases resources.
func (g *IPAPIGeo) Close() error {
	if g.cache != nil {
		g.cache.Clear()
	}
	return nil
}

// BatchGeoLookup performs GeoIP lookups for multiple IPs.
// Note: ip-api.com has a batch endpoint but requires rate limiting.
func BatchGeoLookup(ctx context.Context, geo GeoLookup, ips []net.IP) map[string]*GeoInfo {
	results := make(map[string]*GeoInfo)

	for _, ip := range ips {
		if ip == nil {
			continue
		}

		info, _ := geo.Lookup(ctx, ip)
		if info != nil {
			results[ip.String()] = info
		}
	}

	return results
}
