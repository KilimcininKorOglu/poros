package output

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strconv"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// CSVFormatter formats trace results as CSV.
type CSVFormatter struct {
	config  Config
	columns []string
}

// Default CSV columns
var defaultCSVColumns = []string{
	"hop", "ip", "hostname", "asn", "org", "country", "city",
	"avg_rtt_ms", "min_rtt_ms", "max_rtt_ms", "jitter_ms", "loss_percent",
}

// NewCSVFormatter creates a new CSV formatter.
func NewCSVFormatter(config Config) *CSVFormatter {
	return &CSVFormatter{
		config:  config,
		columns: defaultCSVColumns,
	}
}

// SetColumns allows customizing which columns to include.
func (f *CSVFormatter) SetColumns(columns []string) {
	f.columns = columns
}

// Format formats the trace result as CSV.
func (f *CSVFormatter) Format(result *trace.TraceResult) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// Write header
	if err := writer.Write(f.columns); err != nil {
		return nil, err
	}

	// Write data rows
	for _, hop := range result.Hops {
		row := f.formatRow(&hop)
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// formatRow formats a single hop as a CSV row.
func (f *CSVFormatter) formatRow(hop *trace.Hop) []string {
	row := make([]string, len(f.columns))

	for i, col := range f.columns {
		row[i] = f.getValue(hop, col)
	}

	return row
}

// getValue returns the value for a specific column.
func (f *CSVFormatter) getValue(hop *trace.Hop, column string) string {
	switch column {
	case "hop":
		return strconv.Itoa(hop.Number)

	case "ip":
		if hop.IP != nil {
			return hop.IP.String()
		}
		return "*"

	case "hostname":
		return hop.Hostname

	case "asn":
		if hop.ASN != nil {
			return strconv.Itoa(hop.ASN.Number)
		}
		return ""

	case "org":
		if hop.ASN != nil {
			return hop.ASN.Org
		}
		return ""

	case "country":
		if hop.Geo != nil {
			return hop.Geo.CountryCode
		}
		return ""

	case "city":
		if hop.Geo != nil {
			return hop.Geo.City
		}
		return ""

	case "avg_rtt_ms":
		return formatFloat(hop.AvgRTT)

	case "min_rtt_ms":
		return formatFloat(hop.MinRTT)

	case "max_rtt_ms":
		return formatFloat(hop.MaxRTT)

	case "jitter_ms":
		return formatFloat(hop.Jitter)

	case "loss_percent":
		return formatFloat(hop.LossPercent)

	case "responded":
		if hop.Responded {
			return "true"
		}
		return "false"

	default:
		return ""
	}
}

// formatFloat formats a float for CSV output.
func formatFloat(f float64) string {
	if f <= 0 {
		return ""
	}
	return fmt.Sprintf("%.3f", f)
}

// ContentType returns the MIME type for CSV output.
func (f *CSVFormatter) ContentType() string {
	return "text/csv"
}

// FileExtension returns the file extension for CSV output.
func (f *CSVFormatter) FileExtension() string {
	return "csv"
}
