// Package enrich provides IP enrichment functionality.
package enrich

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

// MaxMindDB provides ASN and GeoIP lookups using MaxMind GeoLite2 databases.
type MaxMindDB struct {
	asnDB      *maxminddb.Reader
	geoDB      *maxminddb.Reader
	licenseKey string
	asnPath    string
	geoPath    string
	mu         sync.RWMutex
}

// MaxMindDBConfig holds configuration for MaxMind database.
type MaxMindDBConfig struct {
	LicenseKey  string // MaxMind license key
	ASNDBPath   string // Path to GeoLite2-ASN.mmdb
	GeoDBPath   string // Path to GeoLite2-City.mmdb
	AutoUpdate  bool   // Enable auto-update check
	UpdateHours int    // Hours between update checks
}

// MaxMind ASN record structure
type maxmindASNRecord struct {
	AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

// MaxMind City record structure
type maxmindCityRecord struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"country"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
		TimeZone  string  `maxminddb:"time_zone"`
	} `maxminddb:"location"`
	Subdivisions []struct {
		ISOCode string            `maxminddb:"iso_code"`
		Names   map[string]string `maxminddb:"names"`
	} `maxminddb:"subdivisions"`
}

// NewMaxMindDB creates a new MaxMind database instance.
// It attempts to open existing database files and optionally downloads them if missing.
func NewMaxMindDB(config MaxMindDBConfig) (*MaxMindDB, error) {
	db := &MaxMindDB{
		licenseKey: config.LicenseKey,
		asnPath:    config.ASNDBPath,
		geoPath:    config.GeoDBPath,
	}

	// Try to open ASN database
	if config.ASNDBPath != "" {
		if _, err := os.Stat(config.ASNDBPath); err == nil {
			asnDB, err := maxminddb.Open(config.ASNDBPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open ASN database: %w", err)
			}
			db.asnDB = asnDB
		}
	}

	// Try to open GeoIP database
	if config.GeoDBPath != "" {
		if _, err := os.Stat(config.GeoDBPath); err == nil {
			geoDB, err := maxminddb.Open(config.GeoDBPath)
			if err != nil {
				if db.asnDB != nil {
					db.asnDB.Close()
				}
				return nil, fmt.Errorf("failed to open GeoIP database: %w", err)
			}
			db.geoDB = geoDB
		}
	}

	return db, nil
}

// HasASN returns true if ASN database is available.
func (db *MaxMindDB) HasASN() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.asnDB != nil
}

// HasGeo returns true if GeoIP database is available.
func (db *MaxMindDB) HasGeo() bool {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.geoDB != nil
}

// LookupASN looks up ASN information for an IP address.
func (db *MaxMindDB) LookupASN(ip net.IP) (*ASNInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.asnDB == nil {
		return nil, fmt.Errorf("ASN database not loaded")
	}

	var record maxmindASNRecord
	err := db.asnDB.Lookup(ip, &record)
	if err != nil {
		return nil, err
	}

	if record.AutonomousSystemNumber == 0 {
		return nil, nil // No ASN data for this IP
	}

	// Extract country code from org name if possible (e.g., "GOOGLE, US")
	country := ""
	if idx := strings.LastIndex(record.AutonomousSystemOrganization, ", "); idx != -1 {
		country = record.AutonomousSystemOrganization[idx+2:]
	}

	return &ASNInfo{
		Number:  int(record.AutonomousSystemNumber),
		Org:     record.AutonomousSystemOrganization,
		Country: country,
	}, nil
}

// LookupGeo looks up geographic information for an IP address.
func (db *MaxMindDB) LookupGeo(ip net.IP) (*GeoInfo, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	if db.geoDB == nil {
		return nil, fmt.Errorf("GeoIP database not loaded")
	}

	var record maxmindCityRecord
	err := db.geoDB.Lookup(ip, &record)
	if err != nil {
		return nil, err
	}

	info := &GeoInfo{
		CountryCode: record.Country.ISOCode,
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
		Timezone:    record.Location.TimeZone,
	}

	// Get English names
	if name, ok := record.Country.Names["en"]; ok {
		info.Country = name
	}
	if name, ok := record.City.Names["en"]; ok {
		info.City = name
	}
	if len(record.Subdivisions) > 0 {
		if name, ok := record.Subdivisions[0].Names["en"]; ok {
			info.Region = name
		}
	}

	return info, nil
}

// Close releases database resources.
func (db *MaxMindDB) Close() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	var errs []error
	if db.asnDB != nil {
		if err := db.asnDB.Close(); err != nil {
			errs = append(errs, err)
		}
		db.asnDB = nil
	}
	if db.geoDB != nil {
		if err := db.geoDB.Close(); err != nil {
			errs = append(errs, err)
		}
		db.geoDB = nil
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// NeedsUpdate checks if databases need to be updated.
func (db *MaxMindDB) NeedsUpdate(maxAge time.Duration) bool {
	// Check ASN database age
	if db.asnPath != "" {
		if info, err := os.Stat(db.asnPath); err != nil || time.Since(info.ModTime()) > maxAge {
			return true
		}
	}

	// Check GeoIP database age
	if db.geoPath != "" {
		if info, err := os.Stat(db.geoPath); err != nil || time.Since(info.ModTime()) > maxAge {
			return true
		}
	}

	return false
}

// DownloadDatabases downloads the latest GeoLite2 databases from MaxMind.
func (db *MaxMindDB) DownloadDatabases(ctx context.Context) error {
	if db.licenseKey == "" {
		return fmt.Errorf("MaxMind license key not configured")
	}

	// Download ASN database
	if db.asnPath != "" {
		if err := db.downloadDatabase(ctx, "GeoLite2-ASN", db.asnPath); err != nil {
			return fmt.Errorf("failed to download ASN database: %w", err)
		}
	}

	// Download GeoIP database
	if db.geoPath != "" {
		if err := db.downloadDatabase(ctx, "GeoLite2-City", db.geoPath); err != nil {
			return fmt.Errorf("failed to download GeoIP database: %w", err)
		}
	}

	// Reload databases
	return db.reload()
}

// downloadDatabase downloads a single database from MaxMind.
func (db *MaxMindDB) downloadDatabase(ctx context.Context, edition, destPath string) error {
	url := fmt.Sprintf(
		"https://download.maxmind.com/app/geoip_download?edition_id=%s&license_key=%s&suffix=tar.gz",
		edition, db.licenseKey,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	// Create destination directory
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Extract .mmdb file from tar.gz
	return extractMMDB(resp.Body, destPath)
}

// extractMMDB extracts the .mmdb file from a tar.gz archive.
func extractMMDB(r io.Reader, destPath string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Look for .mmdb file
		if strings.HasSuffix(header.Name, ".mmdb") {
			// Create destination file
			outFile, err := os.Create(destPath)
			if err != nil {
				return err
			}
			defer outFile.Close()

			// Copy content
			if _, err := io.Copy(outFile, tr); err != nil {
				return err
			}

			return nil
		}
	}

	return fmt.Errorf("no .mmdb file found in archive")
}

// reload reloads the databases after download.
func (db *MaxMindDB) reload() error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Close existing databases
	if db.asnDB != nil {
		db.asnDB.Close()
		db.asnDB = nil
	}
	if db.geoDB != nil {
		db.geoDB.Close()
		db.geoDB = nil
	}

	// Reopen ASN database
	if db.asnPath != "" {
		if _, err := os.Stat(db.asnPath); err == nil {
			asnDB, err := maxminddb.Open(db.asnPath)
			if err != nil {
				return fmt.Errorf("failed to reload ASN database: %w", err)
			}
			db.asnDB = asnDB
		}
	}

	// Reopen GeoIP database
	if db.geoPath != "" {
		if _, err := os.Stat(db.geoPath); err == nil {
			geoDB, err := maxminddb.Open(db.geoPath)
			if err != nil {
				return fmt.Errorf("failed to reload GeoIP database: %w", err)
			}
			db.geoDB = geoDB
		}
	}

	return nil
}

// UpdateIfNeeded checks if databases need updating and downloads them if necessary.
func (db *MaxMindDB) UpdateIfNeeded(ctx context.Context, maxAge time.Duration) error {
	if !db.NeedsUpdate(maxAge) {
		return nil
	}
	return db.DownloadDatabases(ctx)
}
