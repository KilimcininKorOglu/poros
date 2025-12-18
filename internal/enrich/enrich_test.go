package enrich

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	cache := NewCache(3, time.Minute)

	// Test basic set/get
	cache.Set("key1", "value1")
	val, ok := cache.Get("key1")
	if !ok || val != "value1" {
		t.Errorf("Get(key1) = %v, %v; want value1, true", val, ok)
	}

	// Test missing key
	_, ok = cache.Get("missing")
	if ok {
		t.Error("Get(missing) should return false")
	}

	// Test eviction
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")
	cache.Set("key4", "value4") // Should evict key1

	if cache.Size() != 3 {
		t.Errorf("Size() = %d, want 3", cache.Size())
	}

	// Test clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Size() after Clear() = %d, want 0", cache.Size())
	}
}

func TestCacheExpiration(t *testing.T) {
	cache := NewCache(10, 50*time.Millisecond)

	cache.Set("key", "value")

	// Should exist immediately
	_, ok := cache.Get("key")
	if !ok {
		t.Error("Key should exist immediately after set")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("key")
	if ok {
		t.Error("Key should be expired")
	}
}

func TestRDNSResolver(t *testing.T) {
	config := DefaultRDNSConfig()
	config.Timeout = 5 * time.Second
	resolver := NewRDNSResolver(config)
	defer resolver.Close()

	ctx := context.Background()

	// Test localhost (should resolve)
	hostname, err := resolver.Lookup(ctx, net.ParseIP("127.0.0.1"))
	if err != nil {
		t.Logf("Localhost rDNS lookup returned error: %v", err)
	}
	t.Logf("127.0.0.1 -> %q", hostname)

	// Test nil IP
	hostname, err = resolver.Lookup(ctx, nil)
	if err != nil {
		t.Errorf("nil IP lookup should not error: %v", err)
	}
	if hostname != "" {
		t.Errorf("nil IP should return empty hostname, got %q", hostname)
	}

	// Test caching
	resolver.Lookup(ctx, net.ParseIP("127.0.0.1"))
	if resolver.cache.Size() == 0 {
		t.Error("Cache should have entries after lookup")
	}
}

func TestRDNSBatchLookup(t *testing.T) {
	config := DefaultRDNSConfig()
	resolver := NewRDNSResolver(config)
	defer resolver.Close()

	ctx := context.Background()
	ips := []net.IP{
		net.ParseIP("127.0.0.1"),
		net.ParseIP("127.0.0.1"), // Duplicate
		nil,                       // Nil should be skipped
	}

	results := resolver.LookupBatch(ctx, ips)

	if len(results) != 1 { // Only unique non-nil IPs
		t.Errorf("LookupBatch returned %d results, expected 1", len(results))
	}
}

func TestTeamCymruASN(t *testing.T) {
	config := DefaultTeamCymruConfig()
	asn := NewTeamCymruASN(config)
	defer asn.Close()

	ctx := context.Background()

	// Test private IP (should return nil)
	info, err := asn.Lookup(ctx, net.ParseIP("192.168.1.1"))
	if err != nil {
		t.Errorf("Private IP lookup should not error: %v", err)
	}
	if info != nil {
		t.Error("Private IP should return nil ASN info")
	}

	// Test nil IP
	info, err = asn.Lookup(ctx, nil)
	if err != nil {
		t.Errorf("nil IP lookup should not error: %v", err)
	}
	if info != nil {
		t.Error("nil IP should return nil ASN info")
	}
}

func TestTeamCymruASN_PublicIP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	config := DefaultTeamCymruConfig()
	asn := NewTeamCymruASN(config)
	defer asn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test Google DNS (8.8.8.8)
	info, err := asn.Lookup(ctx, net.ParseIP("8.8.8.8"))
	if err != nil {
		t.Logf("ASN lookup for 8.8.8.8 error: %v", err)
		return
	}

	if info != nil {
		t.Logf("8.8.8.8 -> AS%d %s (%s)", info.Number, info.Org, info.Country)
		if info.Number != 15169 {
			t.Logf("Expected AS15169 for Google, got AS%d", info.Number)
		}
	}
}

func TestIPAPIGeo(t *testing.T) {
	config := DefaultIPAPIConfig()
	geo := NewIPAPIGeo(config)
	defer geo.Close()

	ctx := context.Background()

	// Test private IP (should return nil)
	info, err := geo.Lookup(ctx, net.ParseIP("192.168.1.1"))
	if err != nil {
		t.Errorf("Private IP lookup should not error: %v", err)
	}
	if info != nil {
		t.Error("Private IP should return nil GeoIP info")
	}
}

func TestIPAPIGeo_PublicIP(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network test in short mode")
	}

	config := DefaultIPAPIConfig()
	geo := NewIPAPIGeo(config)
	defer geo.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test Google DNS (8.8.8.8)
	info, err := geo.Lookup(ctx, net.ParseIP("8.8.8.8"))
	if err != nil {
		t.Logf("GeoIP lookup for 8.8.8.8 error: %v", err)
		return
	}

	if info != nil {
		t.Logf("8.8.8.8 -> %s, %s (%s)", info.City, info.Country, info.CountryCode)
	}
}

func TestEnricher(t *testing.T) {
	config := DefaultEnricherConfig()
	enricher := NewEnricher(config)
	defer enricher.Close()

	ctx := context.Background()

	// Test with private IP (should not error, but no enrichment)
	result := enricher.EnrichIP(ctx, net.ParseIP("192.168.1.1"))

	// Private IP shouldn't have ASN or GeoIP
	if result.ASN != nil {
		t.Error("Private IP should not have ASN info")
	}
	if result.Geo != nil {
		t.Error("Private IP should not have GeoIP info")
	}
}

func TestEnricherDisabled(t *testing.T) {
	config := EnricherConfig{
		EnableRDNS:  false,
		EnableASN:   false,
		EnableGeoIP: false,
	}
	enricher := NewEnricher(config)
	defer enricher.Close()

	ctx := context.Background()

	result := enricher.EnrichIP(ctx, net.ParseIP("8.8.8.8"))

	// Nothing should be enriched
	if result.Hostname != "" {
		t.Error("rDNS should be disabled")
	}
	if result.ASN != nil {
		t.Error("ASN should be disabled")
	}
	if result.Geo != nil {
		t.Error("GeoIP should be disabled")
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip      string
		private bool
	}{
		{"127.0.0.1", true},
		{"192.168.1.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"169.254.1.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"", true}, // nil IP
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			var ip net.IP
			if tt.ip != "" {
				ip = net.ParseIP(tt.ip)
			}

			result := isPrivateIP(ip)
			if result != tt.private {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.private)
			}
		})
	}
}

func TestReverseIPv6Nibbles(t *testing.T) {
	ip := net.ParseIP("2001:db8::1")
	result := reverseIPv6Nibbles(ip)

	// Should be nibble-reversed
	if len(result) == 0 {
		t.Error("reverseIPv6Nibbles returned empty string")
	}

	// First nibble of reversed should be '1' (last nibble of original)
	if result[0] != '1' {
		t.Errorf("First nibble should be '1', got %c", result[0])
	}
}

func TestParseTeamCymruResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected *ASNInfo
	}{
		{
			"15169 | 8.8.8.0/24 | US | arin | 2014-03-14",
			&ASNInfo{Number: 15169, Country: "US"},
		},
		{
			"invalid",
			nil,
		},
		{
			"",
			nil,
		},
	}

	for _, tt := range tests {
		result := parseTeamCymruResponse(tt.input)

		if tt.expected == nil {
			if result != nil {
				t.Errorf("parseTeamCymruResponse(%q) = %v, want nil", tt.input, result)
			}
			continue
		}

		if result == nil {
			t.Errorf("parseTeamCymruResponse(%q) = nil, want %v", tt.input, tt.expected)
			continue
		}

		if result.Number != tt.expected.Number {
			t.Errorf("ASN Number = %d, want %d", result.Number, tt.expected.Number)
		}
		if result.Country != tt.expected.Country {
			t.Errorf("Country = %q, want %q", result.Country, tt.expected.Country)
		}
	}
}
