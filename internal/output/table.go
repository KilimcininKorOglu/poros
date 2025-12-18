package output

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/KilimcininKorOglu/poros/internal/trace"
	"github.com/olekukonko/tablewriter"
)

// TableFormatter formats trace results as a detailed table.
type TableFormatter struct {
	config Config
	colors *ColorScheme
}

// NewTableFormatter creates a new table formatter.
func NewTableFormatter(config Config) *TableFormatter {
	var colors *ColorScheme
	if config.Colors {
		colors = DefaultColorScheme()
	}

	return &TableFormatter{
		config: config,
		colors: colors,
	}
}

// Format formats the trace result as a detailed table.
func (f *TableFormatter) Format(result *trace.TraceResult) ([]byte, error) {
	var buf bytes.Buffer

	// Header information
	f.writeHeader(&buf, result)

	// Create table
	table := tablewriter.NewWriter(&buf)
	f.configureTable(table)

	// Add header row
	headers := f.getHeaders()
	table.SetHeader(headers)

	// Add data rows
	for _, hop := range result.Hops {
		row := f.formatHopRow(&hop)
		table.Append(row)
	}

	table.Render()

	// Summary
	f.writeSummary(&buf, result)

	return buf.Bytes(), nil
}

// writeHeader writes the trace header information.
func (f *TableFormatter) writeHeader(buf *bytes.Buffer, result *trace.TraceResult) {
	header := fmt.Sprintf("Target: %s (%s)\n", result.Target, result.ResolvedIP)
	header += fmt.Sprintf("Method: %s | Time: %s\n\n",
		strings.ToUpper(result.ProbeMethod),
		result.Timestamp.Format("2006-01-02 15:04:05"))

	if f.colors != nil {
		header = f.colors.Header.Sprint(header)
	}
	buf.WriteString(header)
}

// configureTable sets up the table appearance.
func (f *TableFormatter) configureTable(table *tablewriter.Table) {
	table.SetBorder(true)
	table.SetRowLine(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("│")
	table.SetColumnSeparator("│")
	table.SetRowSeparator("─")
	table.SetHeaderLine(true)
	table.SetTablePadding(" ")
}

// getHeaders returns the column headers.
func (f *TableFormatter) getHeaders() []string {
	headers := []string{"Hop", "IP Address", "Hostname"}

	if !f.config.NoASN {
		headers = append(headers, "ASN", "Organization")
	}

	if !f.config.NoGeoIP {
		headers = append(headers, "Location")
	}

	headers = append(headers, "Avg", "Min", "Max", "Loss")
	return headers
}

// formatHopRow formats a single hop as a table row.
func (f *TableFormatter) formatHopRow(hop *trace.Hop) []string {
	row := []string{
		fmt.Sprintf("%d", hop.Number),
	}

	// IP and Hostname
	if !hop.Responded {
		row = append(row, "*", "-")
	} else {
		row = append(row, hop.IP.String(), truncateString(hop.Hostname, 25))
	}

	// ASN
	if !f.config.NoASN {
		if hop.ASN != nil {
			row = append(row,
				fmt.Sprintf("%d", hop.ASN.Number),
				truncateString(hop.ASN.Org, 20))
		} else {
			row = append(row, "-", "-")
		}
	}

	// GeoIP
	if !f.config.NoGeoIP {
		if hop.Geo != nil {
			location := hop.Geo.CountryCode
			if hop.Geo.City != "" {
				location = fmt.Sprintf("%s, %s", hop.Geo.City, hop.Geo.CountryCode)
			}
			row = append(row, truncateString(location, 20))
		} else {
			row = append(row, "-")
		}
	}

	// RTT stats
	if hop.Responded && hop.AvgRTT > 0 {
		row = append(row,
			f.formatRTT(hop.AvgRTT),
			f.formatRTT(hop.MinRTT),
			f.formatRTT(hop.MaxRTT),
			fmt.Sprintf("%.0f%%", hop.LossPercent))
	} else {
		row = append(row, "-", "-", "-", "-")
	}

	return row
}

// formatRTT formats an RTT value with optional coloring.
func (f *TableFormatter) formatRTT(rtt float64) string {
	if rtt <= 0 {
		return "-"
	}

	str := fmt.Sprintf("%.2f", rtt)

	if f.colors != nil {
		switch {
		case rtt < 50:
			str = f.colors.RTTLow.Sprint(str)
		case rtt < 150:
			str = f.colors.RTTMed.Sprint(str)
		default:
			str = f.colors.RTTHigh.Sprint(str)
		}
	}

	return str
}

// writeSummary writes the trace summary.
func (f *TableFormatter) writeSummary(buf *bytes.Buffer, result *trace.TraceResult) {
	buf.WriteString("\nSummary:\n")

	// Count responding hops
	responding := 0
	for _, hop := range result.Hops {
		if hop.Responded {
			responding++
		}
	}

	fmt.Fprintf(buf, "  Total Hops:    %d\n", result.Summary.TotalHops)
	fmt.Fprintf(buf, "  Responding:    %d\n", responding)
	fmt.Fprintf(buf, "  Total Time:    %.2f ms\n", result.Summary.TotalTimeMs)
	fmt.Fprintf(buf, "  Packet Loss:   %.1f%%\n", result.Summary.PacketLossPercent)

	if result.Completed {
		buf.WriteString("  Status:        ")
		status := "Complete"
		if f.colors != nil {
			status = f.colors.RTTLow.Sprint(status)
		}
		buf.WriteString(status)
		buf.WriteString("\n")
	} else {
		buf.WriteString("  Status:        ")
		status := "Incomplete"
		if f.colors != nil {
			status = f.colors.RTTHigh.Sprint(status)
		}
		buf.WriteString(status)
		buf.WriteString("\n")
	}
}

// ContentType returns the MIME type for table output.
func (f *TableFormatter) ContentType() string {
	return "text/plain"
}

// FileExtension returns the file extension for table output.
func (f *TableFormatter) FileExtension() string {
	return "txt"
}
