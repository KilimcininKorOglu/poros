# Feature 007: Enrichment System (rDNS, ASN, GeoIP)

**Feature ID:** F007
**Feature Name:** Enrichment System (rDNS, ASN, GeoIP)
**Priority:** P2 - HIGH
**Target Version:** v0.3.0
**Estimated Duration:** 2 weeks
**Status:** NOT_STARTED

## Overview

Implement the enrichment layer that adds valuable context to each hop: reverse DNS lookups (rDNS), Autonomous System Number (ASN) information, and geographic IP (GeoIP) data. This transforms raw IP addresses into meaningful information about network paths.

The enrichment system uses local databases (MaxMind) for speed and privacy, with optional fallback to external APIs. Results are cached to minimize repeated lookups and improve performance.

## Goals
- Implement parallel reverse DNS lookups
- Implement ASN lookup using MaxMind GeoLite2-ASN database
- Implement GeoIP lookup using MaxMind GeoLite2-City database
- Create LRU cache for all enrichment data
- Support fallback to external APIs (Team Cymru, ip-api.com)

## Success Criteria
- [ ] All tasks completed (T039-T048)
- [ ] rDNS lookups complete within 2 seconds per trace
- [ ] ASN data shows for public IPs
- [ ] GeoIP shows country/city for public IPs
- [ ] Cache improves repeated lookup performance by 10x
- [ ] External API fallbacks work when local DB missing

## Tasks

### T039: Implement Reverse DNS Lookup

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement parallel reverse DNS lookups for all hop IPs. Use Go's standard library with configurable timeout and concurrency.

#### Technical Details
```go
// internal/enrich/rdns.go
type RDNSResolver struct {
    timeout    time.Duration
    maxWorkers int
}

func NewRDNSResolver(timeout time.Duration) *RDNSResolver {
    return &RDNSResolver{
        timeout:    timeout,
        maxWorkers: 10,
    }
}

func (r *RDNSResolver) Resolve(ctx context.Context, ip net.IP) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, r.timeout)
    defer cancel()
    
    names, err := net.DefaultResolver.LookupAddr(ctx, ip.String())
    if err != nil {
        return "", err
    }
    
    if len(names) > 0 {
        // Remove trailing dot from FQDN
        hostname := strings.TrimSuffix(names[0], ".")
        return hostname, nil
    }
    
    return "", nil
}

func (r *RDNSResolver) ResolveMany(ctx context.Context, ips []net.IP) map[string]string {
    results := make(map[string]string)
    var mu sync.Mutex
    var wg sync.WaitGroup
    
    // Use semaphore for concurrency control
    sem := make(chan struct{}, r.maxWorkers)
    
    for _, ip := range ips {
        if ip == nil || isPrivateIP(ip) {
            continue
        }
        
        wg.Add(1)
        go func(ip net.IP) {
            defer wg.Done()
            
            sem <- struct{}{}
            defer func() { <-sem }()
            
            hostname, err := r.Resolve(ctx, ip)
            if err == nil && hostname != "" {
                mu.Lock()
                results[ip.String()] = hostname
                mu.Unlock()
            }
        }(ip)
    }
    
    wg.Wait()
    return results
}

func isPrivateIP(ip net.IP) bool {
    private := []string{
        "10.0.0.0/8",
        "172.16.0.0/12",
        "192.168.0.0/16",
        "127.0.0.0/8",
    }
    
    for _, cidr := range private {
        _, network, _ := net.ParseCIDR(cidr)
        if network.Contains(ip) {
            return true
        }
    }
    return false
}
```

#### Files to Touch
- `internal/enrich/rdns.go` (new)
- `internal/enrich/rdns_test.go` (new)

#### Dependencies
- T002: Core data structures (Hop)

#### Success Criteria
- [ ] Resolves PTR records correctly
- [ ] Handles timeout gracefully
- [ ] Parallel resolution works
- [ ] Skips private IPs

---

### T040: Implement LRU Cache

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create a generic LRU cache for storing enrichment data with TTL support. This improves performance for repeated lookups and reduces external API calls.

#### Technical Details
```go
// internal/enrich/cache.go
type CacheEntry struct {
    Value     interface{}
    ExpiresAt time.Time
}

type LRUCache struct {
    maxSize   int
    ttl       time.Duration
    items     map[string]*list.Element
    evictList *list.List
    mu        sync.RWMutex
}

type cacheItem struct {
    key   string
    entry CacheEntry
}

func NewLRUCache(maxSize int, ttl time.Duration) *LRUCache {
    return &LRUCache{
        maxSize:   maxSize,
        ttl:       ttl,
        items:     make(map[string]*list.Element),
        evictList: list.New(),
    }
}

func (c *LRUCache) Get(key string) (interface{}, bool) {
    c.mu.RLock()
    elem, ok := c.items[key]
    c.mu.RUnlock()
    
    if !ok {
        return nil, false
    }
    
    item := elem.Value.(*cacheItem)
    
    // Check expiration
    if time.Now().After(item.entry.ExpiresAt) {
        c.Delete(key)
        return nil, false
    }
    
    // Move to front (most recently used)
    c.mu.Lock()
    c.evictList.MoveToFront(elem)
    c.mu.Unlock()
    
    return item.entry.Value, true
}

func (c *LRUCache) Set(key string, value interface{}) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Check if already exists
    if elem, ok := c.items[key]; ok {
        c.evictList.MoveToFront(elem)
        item := elem.Value.(*cacheItem)
        item.entry.Value = value
        item.entry.ExpiresAt = time.Now().Add(c.ttl)
        return
    }
    
    // Evict if full
    if c.evictList.Len() >= c.maxSize {
        c.evictOldest()
    }
    
    // Add new item
    entry := CacheEntry{
        Value:     value,
        ExpiresAt: time.Now().Add(c.ttl),
    }
    item := &cacheItem{key: key, entry: entry}
    elem := c.evictList.PushFront(item)
    c.items[key] = elem
}

func (c *LRUCache) evictOldest() {
    elem := c.evictList.Back()
    if elem != nil {
        c.evictList.Remove(elem)
        item := elem.Value.(*cacheItem)
        delete(c.items, item.key)
    }
}

func (c *LRUCache) Delete(key string) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if elem, ok := c.items[key]; ok {
        c.evictList.Remove(elem)
        delete(c.items, key)
    }
}
```

#### Files to Touch
- `internal/enrich/cache.go` (new)
- `internal/enrich/cache_test.go` (new)

#### Dependencies
- None

#### Success Criteria
- [ ] LRU eviction works correctly
- [ ] TTL expiration works
- [ ] Thread-safe operations
- [ ] Performance benchmarks pass

---

### T041: Implement MaxMind Database Loader

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Create utilities for loading and managing MaxMind GeoLite2 database files (ASN and City databases).

#### Technical Details
```go
// internal/enrich/maxmind.go
import "github.com/oschwald/maxminddb-golang"

type MaxMindDB struct {
    asnDB  *maxminddb.Reader
    geoDB  *maxminddb.Reader
}

type MaxMindConfig struct {
    ASNDBPath  string // Path to GeoLite2-ASN.mmdb
    GeoDBPath  string // Path to GeoLite2-City.mmdb
}

func NewMaxMindDB(config MaxMindConfig) (*MaxMindDB, error) {
    db := &MaxMindDB{}
    
    // Load ASN database
    if config.ASNDBPath != "" {
        asnDB, err := maxminddb.Open(config.ASNDBPath)
        if err != nil {
            return nil, fmt.Errorf("failed to open ASN database: %w", err)
        }
        db.asnDB = asnDB
    }
    
    // Load GeoIP database
    if config.GeoDBPath != "" {
        geoDB, err := maxminddb.Open(config.GeoDBPath)
        if err != nil {
            if db.asnDB != nil {
                db.asnDB.Close()
            }
            return nil, fmt.Errorf("failed to open GeoIP database: %w", err)
        }
        db.geoDB = geoDB
    }
    
    return db, nil
}

func (db *MaxMindDB) Close() error {
    var errs []error
    if db.asnDB != nil {
        if err := db.asnDB.Close(); err != nil {
            errs = append(errs, err)
        }
    }
    if db.geoDB != nil {
        if err := db.geoDB.Close(); err != nil {
            errs = append(errs, err)
        }
    }
    if len(errs) > 0 {
        return errors.Join(errs...)
    }
    return nil
}

func (db *MaxMindDB) HasASN() bool {
    return db.asnDB != nil
}

func (db *MaxMindDB) HasGeo() bool {
    return db.geoDB != nil
}
```

#### Files to Touch
- `internal/enrich/maxmind.go` (new)
- `internal/enrich/maxmind_test.go` (new)

#### Dependencies
- T006: maxminddb-golang dependency

#### Success Criteria
- [ ] Loads .mmdb files correctly
- [ ] Handles missing files gracefully
- [ ] Proper resource cleanup
- [ ] Works with GeoLite2 free databases

---

### T042: Implement ASN Lookup

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement ASN lookup using MaxMind GeoLite2-ASN database with fallback to Team Cymru DNS-based lookup.

#### Technical Details
```go
// internal/enrich/asn.go
type ASNLookup struct {
    db    *MaxMindDB
    cache *LRUCache
}

type ASNRecord struct {
    Number int    `maxminddb:"autonomous_system_number"`
    Org    string `maxminddb:"autonomous_system_organization"`
}

func NewASNLookup(db *MaxMindDB, cache *LRUCache) *ASNLookup {
    return &ASNLookup{
        db:    db,
        cache: cache,
    }
}

func (l *ASNLookup) Lookup(ctx context.Context, ip net.IP) (*ASNInfo, error) {
    if ip == nil || isPrivateIP(ip) {
        return nil, nil
    }
    
    key := "asn:" + ip.String()
    
    // Check cache
    if cached, ok := l.cache.Get(key); ok {
        return cached.(*ASNInfo), nil
    }
    
    var info *ASNInfo
    var err error
    
    // Try local MaxMind database first
    if l.db != nil && l.db.HasASN() {
        info, err = l.lookupMaxMind(ip)
    }
    
    // Fallback to Team Cymru DNS
    if info == nil {
        info, err = l.lookupCymru(ctx, ip)
    }
    
    if err != nil {
        return nil, err
    }
    
    // Cache result
    if info != nil {
        l.cache.Set(key, info)
    }
    
    return info, nil
}

func (l *ASNLookup) lookupMaxMind(ip net.IP) (*ASNInfo, error) {
    var record ASNRecord
    err := l.db.asnDB.Lookup(ip, &record)
    if err != nil {
        return nil, err
    }
    
    if record.Number == 0 {
        return nil, nil
    }
    
    return &ASNInfo{
        Number: record.Number,
        Org:    record.Org,
    }, nil
}

func (l *ASNLookup) lookupCymru(ctx context.Context, ip net.IP) (*ASNInfo, error) {
    // Reverse IP for DNS query
    // Example: 1.2.3.4 -> 4.3.2.1.origin.asn.cymru.com
    reversed := reverseIP(ip)
    query := reversed + ".origin.asn.cymru.com"
    
    resolver := net.DefaultResolver
    records, err := resolver.LookupTXT(ctx, query)
    if err != nil {
        return nil, err
    }
    
    if len(records) == 0 {
        return nil, nil
    }
    
    // Parse: "ASN | IP | Country | RIR | Date"
    parts := strings.Split(records[0], "|")
    if len(parts) < 2 {
        return nil, nil
    }
    
    asn, err := strconv.Atoi(strings.TrimSpace(parts[0]))
    if err != nil {
        return nil, err
    }
    
    // Lookup AS name
    nameQuery := fmt.Sprintf("AS%d.asn.cymru.com", asn)
    nameRecords, _ := resolver.LookupTXT(ctx, nameQuery)
    
    org := ""
    if len(nameRecords) > 0 {
        nameParts := strings.Split(nameRecords[0], "|")
        if len(nameParts) >= 5 {
            org = strings.TrimSpace(nameParts[4])
        }
    }
    
    return &ASNInfo{
        Number: asn,
        Org:    org,
    }, nil
}

func reverseIP(ip net.IP) string {
    ip4 := ip.To4()
    if ip4 != nil {
        return fmt.Sprintf("%d.%d.%d.%d", ip4[3], ip4[2], ip4[1], ip4[0])
    }
    // IPv6 reverse (expand and reverse nibbles)
    // ...
    return ""
}
```

#### Files to Touch
- `internal/enrich/asn.go` (new)
- `internal/enrich/asn_test.go` (new)

#### Dependencies
- T040: LRU cache
- T041: MaxMind database loader

#### Success Criteria
- [ ] MaxMind lookup works
- [ ] Team Cymru fallback works
- [ ] Cache improves performance
- [ ] Handles missing data gracefully

---

### T043: Implement GeoIP Lookup

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement geographic IP lookup using MaxMind GeoLite2-City database with fallback to ip-api.com API.

#### Technical Details
```go
// internal/enrich/geoip.go
type GeoIPLookup struct {
    db    *MaxMindDB
    cache *LRUCache
}

type GeoRecord struct {
    Country struct {
        ISOCode string `maxminddb:"iso_code"`
        Names   struct {
            En string `maxminddb:"en"`
        } `maxminddb:"names"`
    } `maxminddb:"country"`
    City struct {
        Names struct {
            En string `maxminddb:"en"`
        } `maxminddb:"names"`
    } `maxminddb:"city"`
    Location struct {
        Latitude  float64 `maxminddb:"latitude"`
        Longitude float64 `maxminddb:"longitude"`
    } `maxminddb:"location"`
}

func NewGeoIPLookup(db *MaxMindDB, cache *LRUCache) *GeoIPLookup {
    return &GeoIPLookup{
        db:    db,
        cache: cache,
    }
}

func (l *GeoIPLookup) Lookup(ctx context.Context, ip net.IP) (*GeoInfo, error) {
    if ip == nil || isPrivateIP(ip) {
        return nil, nil
    }
    
    key := "geo:" + ip.String()
    
    // Check cache
    if cached, ok := l.cache.Get(key); ok {
        return cached.(*GeoInfo), nil
    }
    
    var info *GeoInfo
    var err error
    
    // Try local MaxMind database first
    if l.db != nil && l.db.HasGeo() {
        info, err = l.lookupMaxMind(ip)
    }
    
    // Fallback to ip-api.com
    if info == nil {
        info, err = l.lookupIPAPI(ctx, ip)
    }
    
    if err != nil {
        return nil, err
    }
    
    // Cache result
    if info != nil {
        l.cache.Set(key, info)
    }
    
    return info, nil
}

func (l *GeoIPLookup) lookupMaxMind(ip net.IP) (*GeoInfo, error) {
    var record GeoRecord
    err := l.db.geoDB.Lookup(ip, &record)
    if err != nil {
        return nil, err
    }
    
    return &GeoInfo{
        Country:     record.Country.Names.En,
        CountryCode: record.Country.ISOCode,
        City:        record.City.Names.En,
        Latitude:    record.Location.Latitude,
        Longitude:   record.Location.Longitude,
    }, nil
}

func (l *GeoIPLookup) lookupIPAPI(ctx context.Context, ip net.IP) (*GeoInfo, error) {
    url := fmt.Sprintf("http://ip-api.com/json/%s?fields=country,countryCode,city,lat,lon", ip)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return nil, err
    }
    
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result struct {
        Country     string  `json:"country"`
        CountryCode string  `json:"countryCode"`
        City        string  `json:"city"`
        Lat         float64 `json:"lat"`
        Lon         float64 `json:"lon"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    
    return &GeoInfo{
        Country:     result.Country,
        CountryCode: result.CountryCode,
        City:        result.City,
        Latitude:    result.Lat,
        Longitude:   result.Lon,
    }, nil
}
```

#### Files to Touch
- `internal/enrich/geoip.go` (new)
- `internal/enrich/geoip_test.go` (new)

#### Dependencies
- T040: LRU cache
- T041: MaxMind database loader

#### Success Criteria
- [ ] MaxMind lookup works
- [ ] ip-api.com fallback works
- [ ] Cache improves performance
- [ ] Rate limiting for external API

---

### T044: Implement Enricher Orchestrator

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create the main Enricher that coordinates all enrichment lookups (rDNS, ASN, GeoIP) and applies them to trace results.

#### Technical Details
```go
// internal/enrich/enricher.go
type Enricher struct {
    rdns    *RDNSResolver
    asn     *ASNLookup
    geoip   *GeoIPLookup
    cache   *LRUCache
    config  EnricherConfig
}

type EnricherConfig struct {
    EnableRDNS  bool
    EnableASN   bool
    EnableGeoIP bool
    RDNSTimeout time.Duration
    CacheSize   int
    CacheTTL    time.Duration
}

func NewEnricher(config EnricherConfig, dbConfig MaxMindConfig) (*Enricher, error) {
    cache := NewLRUCache(config.CacheSize, config.CacheTTL)
    
    var db *MaxMindDB
    var err error
    
    if config.EnableASN || config.EnableGeoIP {
        db, err = NewMaxMindDB(dbConfig)
        if err != nil {
            // Log warning but continue without local DB
            log.Printf("Warning: MaxMind DB not available: %v", err)
        }
    }
    
    e := &Enricher{
        cache:  cache,
        config: config,
    }
    
    if config.EnableRDNS {
        e.rdns = NewRDNSResolver(config.RDNSTimeout)
    }
    
    if config.EnableASN {
        e.asn = NewASNLookup(db, cache)
    }
    
    if config.EnableGeoIP {
        e.geoip = NewGeoIPLookup(db, cache)
    }
    
    return e, nil
}

func (e *Enricher) EnrichHops(ctx context.Context, hops []trace.Hop) {
    // Collect all unique IPs
    ips := make([]net.IP, 0, len(hops))
    for _, hop := range hops {
        if hop.IP != nil && hop.Responded {
            ips = append(ips, hop.IP)
        }
    }
    
    // Parallel enrichment
    var wg sync.WaitGroup
    
    // rDNS (already parallel internally)
    var hostnames map[string]string
    if e.rdns != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            hostnames = e.rdns.ResolveMany(ctx, ips)
        }()
    }
    
    // ASN and GeoIP (can be parallel for different IPs)
    asnResults := make(map[string]*ASNInfo)
    geoResults := make(map[string]*GeoInfo)
    var asnMu, geoMu sync.Mutex
    
    for _, ip := range ips {
        if e.asn != nil {
            wg.Add(1)
            go func(ip net.IP) {
                defer wg.Done()
                info, _ := e.asn.Lookup(ctx, ip)
                if info != nil {
                    asnMu.Lock()
                    asnResults[ip.String()] = info
                    asnMu.Unlock()
                }
            }(ip)
        }
        
        if e.geoip != nil {
            wg.Add(1)
            go func(ip net.IP) {
                defer wg.Done()
                info, _ := e.geoip.Lookup(ctx, ip)
                if info != nil {
                    geoMu.Lock()
                    geoResults[ip.String()] = info
                    geoMu.Unlock()
                }
            }(ip)
        }
    }
    
    wg.Wait()
    
    // Apply results to hops
    for i := range hops {
        if hops[i].IP == nil {
            continue
        }
        
        key := hops[i].IP.String()
        
        if hostname, ok := hostnames[key]; ok {
            hops[i].Hostname = hostname
        }
        if asn, ok := asnResults[key]; ok {
            hops[i].ASN = asn
        }
        if geo, ok := geoResults[key]; ok {
            hops[i].Geo = geo
        }
    }
}
```

#### Files to Touch
- `internal/enrich/enricher.go` (new)
- `internal/enrich/enricher_test.go` (new)

#### Dependencies
- T039: rDNS
- T042: ASN
- T043: GeoIP

#### Success Criteria
- [ ] Coordinates all enrichment types
- [ ] Parallel execution is efficient
- [ ] Handles partial failures gracefully
- [ ] Respects config disable flags

---

### T045: Integrate Enricher with Tracer

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Connect the enrichment system to the tracer so that trace results are automatically enriched when enabled.

#### Technical Details
```go
// internal/trace/tracer.go (update)
func NewTracer(config *TracerConfig) (*Tracer, error) {
    // ... prober creation ...
    
    var enricher *enrich.Enricher
    if config.EnableEnrichment {
        enrichConfig := enrich.EnricherConfig{
            EnableRDNS:  config.EnableRDNS,
            EnableASN:   config.EnableASN,
            EnableGeoIP: config.EnableGeoIP,
            RDNSTimeout: 2 * time.Second,
            CacheSize:   1000,
            CacheTTL:    10 * time.Minute,
        }
        
        dbConfig := enrich.MaxMindConfig{
            ASNDBPath: config.ASNDBPath,
            GeoDBPath: config.GeoDBPath,
        }
        
        var err error
        enricher, err = enrich.NewEnricher(enrichConfig, dbConfig)
        if err != nil {
            // Log warning but continue
            log.Printf("Enrichment unavailable: %v", err)
        }
    }
    
    return &Tracer{
        config:   config,
        prober:   prober,
        enricher: enricher,
    }, nil
}

// TracerConfig additions
type TracerConfig struct {
    // ... existing fields ...
    
    // Enrichment
    EnableEnrichment bool
    EnableRDNS       bool
    EnableASN        bool
    EnableGeoIP      bool
    ASNDBPath        string
    GeoDBPath        string
}
```

#### Files to Touch
- `internal/trace/tracer.go` (update)
- `internal/trace/config.go` (update)

#### Dependencies
- T044: Enricher orchestrator
- T014: Tracer core

#### Success Criteria
- [ ] Enrichment runs after tracing
- [ ] Can be disabled via config
- [ ] Works with both sequential and concurrent modes

---

### T046: Add CLI Flags for Enrichment

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Add CLI flags to control enrichment features and specify database paths.

#### Technical Details
```go
// cmd/poros/root.go (update)
func init() {
    // Enrichment flags
    rootCmd.Flags().Bool("no-enrich", false, 
        "Disable all enrichment (rDNS, ASN, GeoIP)")
    rootCmd.Flags().Bool("no-rdns", false, 
        "Disable reverse DNS lookups")
    rootCmd.Flags().Bool("no-asn", false, 
        "Disable ASN lookups")
    rootCmd.Flags().Bool("no-geoip", false, 
        "Disable GeoIP lookups")
    
    // Database paths
    rootCmd.Flags().String("asn-db", "", 
        "Path to MaxMind GeoLite2-ASN.mmdb")
    rootCmd.Flags().String("geo-db", "", 
        "Path to MaxMind GeoLite2-City.mmdb")
}

func buildTracerConfig(cmd *cobra.Command, target string) (*trace.TracerConfig, error) {
    noEnrich := getBool(cmd, "no-enrich")
    
    config := &trace.TracerConfig{
        // ... existing fields ...
        
        EnableEnrichment: !noEnrich,
        EnableRDNS:       !noEnrich && !getBool(cmd, "no-rdns"),
        EnableASN:        !noEnrich && !getBool(cmd, "no-asn"),
        EnableGeoIP:      !noEnrich && !getBool(cmd, "no-geoip"),
        ASNDBPath:        getDefaultASNDBPath(cmd),
        GeoDBPath:        getDefaultGeoDBPath(cmd),
    }
    
    return config, nil
}

func getDefaultASNDBPath(cmd *cobra.Command) string {
    if path, _ := cmd.Flags().GetString("asn-db"); path != "" {
        return path
    }
    
    // Check default locations
    paths := []string{
        "./data/GeoLite2-ASN.mmdb",
        "~/.local/share/poros/GeoLite2-ASN.mmdb",
        "/usr/share/GeoIP/GeoLite2-ASN.mmdb",
    }
    
    for _, p := range paths {
        expanded := expandPath(p)
        if fileExists(expanded) {
            return expanded
        }
    }
    
    return ""
}
```

#### Files to Touch
- `cmd/poros/root.go` (update)
- `cmd/poros/flags.go` (update)
- `cmd/poros/config.go` (new - default path handling)

#### Dependencies
- T045: Enricher integration

#### Success Criteria
- [ ] All enrichment flags work
- [ ] `--no-enrich` disables everything
- [ ] Database auto-detection works
- [ ] Help text is clear

---

### T047: Create GeoIP Database Download Script

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create a script to download MaxMind GeoLite2 databases. Note: Requires MaxMind license key (free registration).

#### Technical Details
```bash
#!/bin/bash
# scripts/download-geoip.sh

set -e

DEST_DIR="${POROS_DATA_DIR:-$HOME/.local/share/poros}"
LICENSE_KEY="${MAXMIND_LICENSE_KEY:-}"

if [ -z "$LICENSE_KEY" ]; then
    echo "Error: MAXMIND_LICENSE_KEY environment variable not set"
    echo "Get a free license key at: https://www.maxmind.com/en/geolite2/signup"
    exit 1
fi

mkdir -p "$DEST_DIR"

# Download ASN database
echo "Downloading GeoLite2-ASN..."
curl -s -L "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-ASN&license_key=${LICENSE_KEY}&suffix=tar.gz" | \
    tar -xzf - -C "$DEST_DIR" --strip-components=1 --wildcards "*.mmdb"

# Download City database
echo "Downloading GeoLite2-City..."
curl -s -L "https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-City&license_key=${LICENSE_KEY}&suffix=tar.gz" | \
    tar -xzf - -C "$DEST_DIR" --strip-components=1 --wildcards "*.mmdb"

echo "Done! Databases saved to: $DEST_DIR"
ls -la "$DEST_DIR"/*.mmdb
```

Also create Go-based download option:
```go
// cmd/poros/download.go
var downloadCmd = &cobra.Command{
    Use:   "download-db",
    Short: "Download MaxMind GeoLite2 databases",
    RunE:  runDownload,
}
```

#### Files to Touch
- `scripts/download-geoip.sh` (new)
- `scripts/download-geoip.ps1` (new - Windows)
- `cmd/poros/download.go` (new - optional Go implementation)

#### Dependencies
- None (standalone script)

#### Success Criteria
- [ ] Downloads ASN database
- [ ] Downloads City database
- [ ] Works on Linux/macOS/Windows
- [ ] Clear error for missing license key

---

### T048: Add Enrichment Integration Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create integration tests that verify enrichment works with real data.

#### Technical Details
```go
// internal/enrich/enricher_integration_test.go
//go:build integration

func TestEnrichment_RealIPs(t *testing.T) {
    enricher, err := NewEnricher(EnricherConfig{
        EnableRDNS:  true,
        EnableASN:   true,
        EnableGeoIP: true,
        RDNSTimeout: 2 * time.Second,
        CacheSize:   100,
        CacheTTL:    time.Minute,
    }, MaxMindConfig{})
    require.NoError(t, err)
    
    // Test with Google DNS
    hops := []trace.Hop{
        {IP: net.ParseIP("8.8.8.8"), Responded: true},
        {IP: net.ParseIP("1.1.1.1"), Responded: true},
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    enricher.EnrichHops(ctx, hops)
    
    // Verify Google DNS enrichment
    assert.NotEmpty(t, hops[0].Hostname)
    assert.NotNil(t, hops[0].ASN)
    assert.Equal(t, 15169, hops[0].ASN.Number) // Google ASN
}

func TestEnrichment_Cache(t *testing.T) {
    // Verify cache improves performance
    // First lookup vs second lookup timing
}
```

#### Files to Touch
- `internal/enrich/enricher_integration_test.go` (new)
- `internal/enrich/cache_bench_test.go` (new)

#### Dependencies
- T044: Enricher complete

#### Success Criteria
- [ ] Tests pass with external services
- [ ] Tests pass with local MaxMind DB
- [ ] Cache performance verified

---

## Performance Targets
- rDNS: < 2s for all hops (parallel)
- ASN lookup: < 100μs with local DB
- GeoIP lookup: < 100μs with local DB
- Cache hit: < 1μs

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| MaxMind DB not available | Medium | Medium | External API fallbacks |
| rDNS timeouts | High | Low | Aggressive timeout, skip on failure |
| API rate limits | Medium | Medium | Cache aggressively, local DB preference |
| External API changes | Low | Medium | Version pin, monitoring |

## Notes
- MaxMind requires free account registration for GeoLite2
- Consider bundling a small ASN dataset for common networks
- ip-api.com has rate limits (45 requests/minute for free tier)
- Cache should persist across traces in long-running TUI mode
