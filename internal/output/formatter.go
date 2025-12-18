// Package output provides formatting and output functionality for trace results.
package output

import (
	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// Format represents the output format type.
type Format int

const (
	// FormatText is the classic traceroute-style output
	FormatText Format = iota
	// FormatVerbose is the detailed table output
	FormatVerbose
	// FormatJSON is JSON output
	FormatJSON
	// FormatCSV is CSV output
	FormatCSV
	// FormatHTML is HTML report output
	FormatHTML
)

// String returns the string representation of the format.
func (f Format) String() string {
	switch f {
	case FormatText:
		return "text"
	case FormatVerbose:
		return "verbose"
	case FormatJSON:
		return "json"
	case FormatCSV:
		return "csv"
	case FormatHTML:
		return "html"
	default:
		return "unknown"
	}
}

// Formatter defines the interface for output formatters.
type Formatter interface {
	// Format converts a TraceResult to formatted output bytes.
	Format(result *trace.TraceResult) ([]byte, error)

	// ContentType returns the MIME type for the output.
	ContentType() string

	// FileExtension returns the typical file extension for the output.
	FileExtension() string
}

// Config holds configuration for formatters.
type Config struct {
	// Colors enables ANSI color output
	Colors bool

	// NoHostname disables hostname display
	NoHostname bool

	// NoASN disables ASN information display
	NoASN bool

	// NoGeoIP disables GeoIP information display
	NoGeoIP bool

	// Width is the terminal width (0 = auto-detect)
	Width int
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Colors: true,
		Width:  0, // Auto-detect
	}
}

// NewFormatter creates a formatter based on the specified format.
func NewFormatter(format Format, config Config) Formatter {
	switch format {
	case FormatText:
		return NewTextFormatter(config)
	case FormatVerbose:
		return NewTableFormatter(config)
	case FormatJSON:
		return NewJSONFormatter(config)
	case FormatCSV:
		return NewCSVFormatter(config)
	default:
		return NewTextFormatter(config)
	}
}
