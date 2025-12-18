// Package config provides configuration file support for Poros.
package config

import (
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the Poros configuration file structure.
type Config struct {
	// Defaults are applied when flags are not specified
	Defaults Defaults `yaml:"defaults"`

	// Aliases for common targets
	Aliases map[string]string `yaml:"aliases,omitempty"`
}

// Defaults holds default values for trace parameters.
type Defaults struct {
	// Output mode
	TUI     bool `yaml:"tui"`
	Verbose bool `yaml:"verbose"`
	JSON    bool `yaml:"json"`
	CSV     bool `yaml:"csv"`
	NoColor bool `yaml:"no_color"`

	// Probe method: icmp, udp, tcp, paris
	ProbeMethod string `yaml:"probe_method"`
	Paris       bool   `yaml:"paris"`

	// Trace parameters
	MaxHops    int           `yaml:"max_hops"`
	Queries    int           `yaml:"queries"`
	Timeout    time.Duration `yaml:"timeout"`
	FirstHop   int           `yaml:"first_hop"`
	Sequential bool          `yaml:"sequential"`

	// Network
	IPv4 bool `yaml:"ipv4"`
	IPv6 bool `yaml:"ipv6"`
	Port int  `yaml:"port"`

	// Enrichment
	Enrichment EnrichmentConfig `yaml:"enrichment"`
}

// EnrichmentConfig holds enrichment settings.
type EnrichmentConfig struct {
	Enabled bool `yaml:"enabled"`
	RDNS    bool `yaml:"rdns"`
	ASN     bool `yaml:"asn"`
	GeoIP   bool `yaml:"geoip"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Defaults: Defaults{
			TUI:         false,
			Verbose:     false,
			JSON:        false,
			CSV:         false,
			NoColor:     false,
			ProbeMethod: "icmp",
			Paris:       false,
			MaxHops:     30,
			Queries:     3,
			Timeout:     3 * time.Second,
			FirstHop:    1,
			Sequential:  false,
			IPv4:        false,
			IPv6:        false,
			Port:        0, // 0 means use default for probe method
			Enrichment: EnrichmentConfig{
				Enabled: true,
				RDNS:    true,
				ASN:     true,
				GeoIP:   true,
			},
		},
		Aliases: make(map[string]string),
	}
}

// Load reads configuration from the default config file locations.
// It searches in order:
//  1. ./poros.yaml (current directory)
//  2. ~/.config/poros/config.yaml (Linux/macOS)
//  3. %APPDATA%\poros\config.yaml (Windows)
//
// If no config file is found, returns default configuration.
func Load() (*Config, error) {
	paths := getConfigPaths()

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return LoadFrom(path)
		}
	}

	// No config file found, return defaults
	return DefaultConfig(), nil
}

// LoadFrom reads configuration from a specific file path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// Save writes the configuration to the default user config path.
func (c *Config) Save() error {
	path := getUserConfigPath()

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveTo writes the configuration to a specific file path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// getConfigPaths returns the list of config file paths to search.
func getConfigPaths() []string {
	paths := []string{
		"poros.yaml",
		"poros.yml",
		".poros.yaml",
		".poros.yml",
	}

	// Add user config path
	userPath := getUserConfigPath()
	if userPath != "" {
		paths = append(paths, userPath)
	}

	return paths
}

// getUserConfigPath returns the user-specific config file path.
func getUserConfigPath() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "poros", "config.yaml")
		}
	default: // Linux, macOS, etc.
		home, err := os.UserHomeDir()
		if err == nil {
			// Check XDG_CONFIG_HOME first
			xdgConfig := os.Getenv("XDG_CONFIG_HOME")
			if xdgConfig != "" {
				return filepath.Join(xdgConfig, "poros", "config.yaml")
			}
			return filepath.Join(home, ".config", "poros", "config.yaml")
		}
	}
	return ""
}

// GetConfigPath returns the path where user config would be saved.
func GetConfigPath() string {
	return getUserConfigPath()
}

// GenerateExample generates an example configuration file content.
func GenerateExample() string {
	return `# Poros Configuration File
# Location: ~/.config/poros/config.yaml (Linux/macOS)
#           %APPDATA%\poros\config.yaml (Windows)
#           ./poros.yaml (current directory)

defaults:
  # Output mode (only one should be true)
  tui: false              # Interactive TUI mode
  verbose: false          # Detailed table output
  json: false             # JSON output
  csv: false              # CSV output
  no_color: false         # Disable colors

  # Probe method: icmp, udp, tcp
  probe_method: icmp
  paris: false            # Use Paris traceroute algorithm

  # Trace parameters
  max_hops: 30            # Maximum number of hops
  queries: 3              # Probes per hop
  timeout: 3s             # Probe timeout
  first_hop: 1            # Starting hop
  sequential: false       # Use sequential mode

  # Network settings
  ipv4: false             # Force IPv4
  ipv6: false             # Force IPv6
  port: 0                 # Destination port (0 = default)

  # Enrichment settings
  enrichment:
    enabled: true         # Master switch for all enrichment
    rdns: true            # Reverse DNS lookups
    asn: true             # ASN lookups
    geoip: true           # GeoIP lookups

# Target aliases (optional)
aliases:
  dns: 8.8.8.8
  cf: 1.1.1.1
  google: google.com
`
}
