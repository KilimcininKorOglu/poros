package output

import (
	"bytes"
	"fmt"

	"github.com/KilimcininKorOglu/poros/internal/trace"
	"github.com/fatih/color"
)

// TextFormatter formats trace results in classic traceroute style.
type TextFormatter struct {
	config Config
	colors *ColorScheme
}

// NewTextFormatter creates a new text formatter.
func NewTextFormatter(config Config) *TextFormatter {
	var colors *ColorScheme
	if config.Colors {
		colors = DefaultColorScheme()
	}

	return &TextFormatter{
		config: config,
		colors: colors,
	}
}

// Format formats the trace result as classic traceroute text output.
func (f *TextFormatter) Format(result *trace.TraceResult) ([]byte, error) {
	var buf bytes.Buffer

	// Header
	fmt.Fprintf(&buf, "traceroute to %s (%s), %d hops max\n\n",
		result.Target, result.ResolvedIP, len(result.Hops)+5)

	// Hops
	for _, hop := range result.Hops {
		f.formatHop(&buf, &hop)
	}

	// Summary
	buf.WriteString("\n")
	if result.Completed {
		fmt.Fprintf(&buf, "Trace complete. %d hops, %.2f ms total\n",
			result.Summary.TotalHops, result.Summary.TotalTimeMs)
	} else {
		fmt.Fprintf(&buf, "Trace incomplete after %d hops\n",
			result.Summary.TotalHops)
	}

	return buf.Bytes(), nil
}

// FormatHop formats a single hop and returns it as a string.
// This can be used for streaming output.
func (f *TextFormatter) FormatHop(hop *trace.Hop) string {
	var buf bytes.Buffer
	f.formatHop(&buf, hop)
	return buf.String()
}

// formatHop formats a single hop line.
func (f *TextFormatter) formatHop(buf *bytes.Buffer, hop *trace.Hop) {
	// Hop number
	hopNum := fmt.Sprintf("%3d  ", hop.Number)
	if f.colors != nil {
		hopNum = f.colors.Hop.Sprint(hopNum)
	}
	buf.WriteString(hopNum)

	// No response
	if !hop.Responded {
		timeout := "* * *"
		if f.colors != nil {
			timeout = f.colors.Timeout.Sprint(timeout)
		}
		buf.WriteString(timeout)
		buf.WriteString("\n")
		return
	}

	// IP address
	ipStr := hop.IP.String()
	if f.colors != nil {
		ipStr = f.colors.IP.Sprint(ipStr)
	}

	// Hostname (if available and not disabled)
	if hop.Hostname != "" && !f.config.NoHostname {
		hostname := hop.Hostname
		if f.colors != nil {
			hostname = f.colors.Hostname.Sprint(hostname)
		}
		fmt.Fprintf(buf, "%s (%s)  ", hostname, ipStr)
	} else {
		fmt.Fprintf(buf, "%s  ", ipStr)
	}

	// RTT values
	for _, rtt := range hop.RTTs {
		if rtt < 0 {
			timeout := "*"
			if f.colors != nil {
				timeout = f.colors.Timeout.Sprint(timeout)
			}
			fmt.Fprintf(buf, "%s  ", timeout)
		} else {
			rttStr := fmt.Sprintf("%.3f ms", rtt)
			if f.colors != nil {
				rttStr = f.colorizeRTT(rtt)
			}
			fmt.Fprintf(buf, "%s  ", rttStr)
		}
	}

	// ASN info (if available and not disabled)
	if hop.ASN != nil && !f.config.NoASN {
		asnStr := fmt.Sprintf("[AS%d %s]", hop.ASN.Number, truncateString(hop.ASN.Org, 20))
		if f.colors != nil {
			asnStr = f.colors.ASN.Sprint(asnStr)
		}
		buf.WriteString(asnStr)
	}

	buf.WriteString("\n")
}

// colorizeRTT returns a colored RTT string based on latency thresholds.
func (f *TextFormatter) colorizeRTT(rtt float64) string {
	str := fmt.Sprintf("%.3f ms", rtt)
	if f.colors == nil {
		return str
	}

	switch {
	case rtt < 50:
		return f.colors.RTTLow.Sprint(str)
	case rtt < 150:
		return f.colors.RTTMed.Sprint(str)
	default:
		return f.colors.RTTHigh.Sprint(str)
	}
}

// ContentType returns the MIME type for text output.
func (f *TextFormatter) ContentType() string {
	return "text/plain"
}

// FileExtension returns the file extension for text output.
func (f *TextFormatter) FileExtension() string {
	return "txt"
}

// ColorScheme defines colors for different output elements.
type ColorScheme struct {
	Hop      *color.Color
	IP       *color.Color
	Hostname *color.Color
	RTTLow   *color.Color // < 50ms
	RTTMed   *color.Color // 50-150ms
	RTTHigh  *color.Color // > 150ms
	Timeout  *color.Color
	ASN      *color.Color
	Geo      *color.Color
	Header   *color.Color
}

// DefaultColorScheme returns the default color scheme.
func DefaultColorScheme() *ColorScheme {
	return &ColorScheme{
		Hop:      color.New(color.FgCyan, color.Bold),
		IP:       color.New(color.FgWhite),
		Hostname: color.New(color.FgGreen),
		RTTLow:   color.New(color.FgGreen),
		RTTMed:   color.New(color.FgYellow),
		RTTHigh:  color.New(color.FgRed),
		Timeout:  color.New(color.FgRed, color.Bold),
		ASN:      color.New(color.FgMagenta),
		Geo:      color.New(color.FgBlue),
		Header:   color.New(color.FgWhite, color.Bold),
	}
}

// Helper functions

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
