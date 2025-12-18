# Feature 004: Text Output Formatters

**Feature ID:** F004
**Feature Name:** Text Output Formatters
**Priority:** P1 - CRITICAL
**Target Version:** v0.1.0
**Estimated Duration:** 1 week
**Status:** NOT_STARTED

## Overview

Implement the output formatting layer that transforms TraceResult data into human-readable text output. This includes classic traceroute-style output, verbose table format, and the formatter interface that will be extended for JSON, CSV, and other formats.

Good output formatting is essential for usability. The classic format should be familiar to users of traditional traceroute tools, while the verbose format should leverage the additional data (ASN, GeoIP) that Poros provides.

## Goals
- Create a flexible Formatter interface for extensibility
- Implement classic traceroute-style text output
- Implement detailed table output with all hop information
- Support colored output for terminal
- Handle terminal width constraints

## Success Criteria
- [ ] All tasks completed (T020-T025)
- [ ] Classic output matches traditional traceroute format
- [ ] Verbose table is readable and well-aligned
- [ ] Colors enhance readability (when enabled)
- [ ] Output works correctly on all terminal sizes

## Tasks

### T020: Define Formatter Interface

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Create the Formatter interface that all output formatters (text, JSON, CSV, HTML) will implement. This ensures consistent API across different output types.

#### Technical Details
```go
// internal/output/formatter.go
type Formatter interface {
    // Format converts TraceResult to formatted output
    Format(result *trace.TraceResult) ([]byte, error)
    
    // ContentType returns the MIME type (e.g., "text/plain", "application/json")
    ContentType() string
    
    // FileExtension returns the typical file extension (e.g., "txt", "json")
    FileExtension() string
}

type OutputFormat int

const (
    FormatText OutputFormat = iota
    FormatVerbose
    FormatJSON
    FormatCSV
    FormatHTML
)

// FormatterConfig holds common configuration
type FormatterConfig struct {
    Colors     bool   // Enable ANSI colors
    NoHostname bool   // Skip hostname display
    NoASN      bool   // Skip ASN display
    NoGeoIP    bool   // Skip GeoIP display
}

// NewFormatter creates formatter based on format type
func NewFormatter(format OutputFormat, config FormatterConfig) Formatter {
    switch format {
    case FormatText:
        return NewTextFormatter(config)
    case FormatVerbose:
        return NewTableFormatter(config)
    case FormatJSON:
        return NewJSONFormatter(config)
    // ...
    }
}
```

#### Files to Touch
- `internal/output/formatter.go` (new)
- `internal/output/config.go` (new)

#### Dependencies
- T002: Core data structures (TraceResult)

#### Success Criteria
- [ ] Interface is well-documented
- [ ] Config covers all common options
- [ ] Factory function works correctly
- [ ] Easy to add new formatters

---

### T021: Implement Classic Text Formatter

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Implement the classic traceroute output format that users are familiar with. Each hop shows: hop number, IP address, and RTT for each probe.

#### Technical Details
```go
// internal/output/text.go
type TextFormatter struct {
    config FormatterConfig
    color  *color.Color
}

func NewTextFormatter(config FormatterConfig) *TextFormatter {
    return &TextFormatter{
        config: config,
        color:  color.New(color.FgCyan),
    }
}

func (f *TextFormatter) Format(result *trace.TraceResult) ([]byte, error) {
    var buf bytes.Buffer
    
    // Header
    fmt.Fprintf(&buf, "traceroute to %s (%s), %d hops max\n\n",
        result.Target, result.ResolvedIP, len(result.Hops)+5)
    
    for _, hop := range result.Hops {
        // Format: "  1  192.168.1.1  1.234 ms  1.456 ms  1.123 ms"
        fmt.Fprintf(&buf, "%3d  ", hop.Number)
        
        if !hop.Responded {
            fmt.Fprintf(&buf, "* * *\n")
            continue
        }
        
        // IP and optional hostname
        if hop.Hostname != "" && !f.config.NoHostname {
            fmt.Fprintf(&buf, "%s (%s)  ", hop.Hostname, hop.IP)
        } else {
            fmt.Fprintf(&buf, "%s  ", hop.IP)
        }
        
        // RTT values
        for _, rtt := range hop.RTTs {
            if rtt < 0 {
                fmt.Fprintf(&buf, "*  ")
            } else {
                fmt.Fprintf(&buf, "%.3f ms  ", rtt)
            }
        }
        
        // ASN info (if available and not disabled)
        if hop.ASN != nil && !f.config.NoASN {
            fmt.Fprintf(&buf, "[AS%d %s]", hop.ASN.Number, hop.ASN.Org)
        }
        
        fmt.Fprintf(&buf, "\n")
    }
    
    return buf.Bytes(), nil
}

func (f *TextFormatter) ContentType() string {
    return "text/plain"
}

func (f *TextFormatter) FileExtension() string {
    return "txt"
}
```

#### Files to Touch
- `internal/output/text.go` (new)
- `internal/output/text_test.go` (new)

#### Dependencies
- T020: Formatter interface
- T006: fatih/color dependency

#### Success Criteria
- [ ] Output matches classic traceroute format
- [ ] Handles timeout hops (*)
- [ ] Hostname display is optional
- [ ] ASN display works
- [ ] Unit tests cover all cases

---

### T022: Implement Table Formatter (Verbose Mode)

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description
Implement the detailed table output format that shows all available information in a structured, aligned table format.

#### Technical Details
```go
// internal/output/table.go
type TableFormatter struct {
    config FormatterConfig
    writer *tablewriter.Table
}

func (f *TableFormatter) Format(result *trace.TraceResult) ([]byte, error) {
    var buf bytes.Buffer
    
    // Header with target info
    fmt.Fprintf(&buf, "Target: %s (%s)\n", result.Target, result.ResolvedIP)
    fmt.Fprintf(&buf, "Method: %s | Time: %s\n\n", 
        result.ProbeMethod, result.Timestamp.Format(time.RFC3339))
    
    // Create table
    table := tablewriter.NewWriter(&buf)
    table.SetHeader([]string{"Hop", "IP Address", "Hostname", "ASN", "Org", "Geo", "Avg", "Min", "Max", "Loss"})
    table.SetBorder(true)
    table.SetRowLine(true)
    table.SetAlignment(tablewriter.ALIGN_LEFT)
    table.SetHeaderAlignment(tablewriter.ALIGN_CENTER)
    
    // Column alignment
    table.SetColumnAlignment([]int{
        tablewriter.ALIGN_RIGHT,  // Hop
        tablewriter.ALIGN_LEFT,   // IP
        tablewriter.ALIGN_LEFT,   // Hostname
        tablewriter.ALIGN_RIGHT,  // ASN
        tablewriter.ALIGN_LEFT,   // Org
        tablewriter.ALIGN_LEFT,   // Geo
        tablewriter.ALIGN_RIGHT,  // Avg
        tablewriter.ALIGN_RIGHT,  // Min
        tablewriter.ALIGN_RIGHT,  // Max
        tablewriter.ALIGN_RIGHT,  // Loss
    })
    
    for _, hop := range result.Hops {
        row := f.formatHopRow(hop)
        table.Append(row)
    }
    
    table.Render()
    
    // Summary
    fmt.Fprintf(&buf, "\nSummary:\n")
    fmt.Fprintf(&buf, "  Total Hops: %d\n", result.Summary.TotalHops)
    fmt.Fprintf(&buf, "  Total Time: %.2f ms\n", result.Summary.TotalTimeMs)
    fmt.Fprintf(&buf, "  Packet Loss: %.1f%%\n", result.Summary.PacketLossPercent)
    
    return buf.Bytes(), nil
}

func (f *TableFormatter) formatHopRow(hop trace.Hop) []string {
    row := make([]string, 10)
    
    row[0] = fmt.Sprintf("%d", hop.Number)
    
    if !hop.Responded {
        row[1] = "*"
        return row
    }
    
    row[1] = hop.IP.String()
    row[2] = truncate(hop.Hostname, 25)
    
    if hop.ASN != nil {
        row[3] = fmt.Sprintf("%d", hop.ASN.Number)
        row[4] = truncate(hop.ASN.Org, 15)
    }
    
    if hop.Geo != nil {
        row[5] = fmt.Sprintf("%s, %s", hop.Geo.CountryCode, hop.Geo.City)
    }
    
    row[6] = formatRTT(hop.AvgRTT)
    row[7] = formatRTT(hop.MinRTT)
    row[8] = formatRTT(hop.MaxRTT)
    row[9] = fmt.Sprintf("%.0f%%", hop.LossPercent)
    
    return row
}
```

#### Files to Touch
- `internal/output/table.go` (new)
- `internal/output/table_test.go` (new)
- `internal/output/helpers.go` (new - truncate, formatRTT)

#### Dependencies
- T020: Formatter interface
- T006: tablewriter dependency

#### Success Criteria
- [ ] Table is properly aligned
- [ ] All columns display correctly
- [ ] Long values are truncated appropriately
- [ ] Summary section is accurate
- [ ] Handles missing data gracefully

---

### T023: Implement Color Support

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Add color support to text and table formatters for better terminal readability. Colors should indicate status (success, timeout, high latency) and improve visual scanning.

#### Technical Details
```go
// internal/output/colors.go
type ColorScheme struct {
    Hop       *color.Color
    IP        *color.Color
    Hostname  *color.Color
    RTTLow    *color.Color  // < 50ms
    RTTMed    *color.Color  // 50-150ms
    RTTHigh   *color.Color  // > 150ms
    Timeout   *color.Color  // *
    ASN       *color.Color
    Geo       *color.Color
    Header    *color.Color
}

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

func (f *TextFormatter) colorizeRTT(rtt float64) string {
    if !f.config.Colors {
        return formatRTT(rtt)
    }
    
    str := formatRTT(rtt)
    switch {
    case rtt < 0:
        return f.colors.Timeout.Sprint("*")
    case rtt < 50:
        return f.colors.RTTLow.Sprint(str)
    case rtt < 150:
        return f.colors.RTTMed.Sprint(str)
    default:
        return f.colors.RTTHigh.Sprint(str)
    }
}
```

#### Files to Touch
- `internal/output/colors.go` (new)
- `internal/output/text.go` (update)
- `internal/output/table.go` (update)

#### Dependencies
- T021: Text formatter
- T022: Table formatter

#### Success Criteria
- [ ] Colors work on Linux/macOS terminals
- [ ] Colors can be disabled
- [ ] RTT thresholds are sensible
- [ ] No color codes in piped output

---

### T024: Implement Output Writer Integration

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Integrate formatters with the main application output flow. Handle stdout/file output, terminal detection, and automatic color disabling for non-TTY output.

#### Technical Details
```go
// internal/output/writer.go
type OutputWriter struct {
    formatter Formatter
    writer    io.Writer
    isTTY     bool
}

func NewOutputWriter(format OutputFormat, config FormatterConfig) *OutputWriter {
    // Detect if stdout is a TTY
    isTTY := isTerminal(os.Stdout.Fd())
    
    // Disable colors if not TTY
    if !isTTY {
        config.Colors = false
    }
    
    return &OutputWriter{
        formatter: NewFormatter(format, config),
        writer:    os.Stdout,
        isTTY:     isTTY,
    }
}

func (w *OutputWriter) Write(result *trace.TraceResult) error {
    data, err := w.formatter.Format(result)
    if err != nil {
        return err
    }
    
    _, err = w.writer.Write(data)
    return err
}

func (w *OutputWriter) SetOutput(writer io.Writer) {
    w.writer = writer
}

func isTerminal(fd uintptr) bool {
    // Platform-specific TTY detection
    // Use golang.org/x/term or syscall
    return term.IsTerminal(int(fd))
}
```

#### Files to Touch
- `internal/output/writer.go` (new)
- `internal/output/terminal.go` (new - TTY detection)

#### Dependencies
- T020: Formatter interface
- T021: Text formatter
- T022: Table formatter

#### Success Criteria
- [ ] TTY detection works
- [ ] Colors auto-disable for pipes
- [ ] File output works
- [ ] Writer is reusable

---

### T025: Wire Output to CLI

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Connect the output formatting system to the CLI layer. Implement the `-v/--verbose` flag and output selection logic in the main command.

#### Technical Details
```go
// cmd/poros/root.go
var (
    verbose bool
    noColor bool
)

func init() {
    rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, 
        "Show detailed table output")
    rootCmd.Flags().Bool("no-color", false, 
        "Disable colored output")
}

func runTrace(cmd *cobra.Command, args []string) error {
    // ... tracer setup ...
    
    // Select output format
    format := output.FormatText
    if verbose {
        format = output.FormatVerbose
    }
    
    // Configure output
    config := output.FormatterConfig{
        Colors: !noColor,
    }
    
    writer := output.NewOutputWriter(format, config)
    
    // Run trace
    result, err := tracer.Trace(cmd.Context(), target)
    if err != nil {
        return err
    }
    
    // Output result
    return writer.Write(result)
}
```

#### Files to Touch
- `cmd/poros/root.go` (update)
- `cmd/poros/output.go` (new - output-related flag handling)

#### Dependencies
- T005: CLI framework
- T024: Output writer

#### Success Criteria
- [ ] `poros google.com` shows classic output
- [ ] `poros -v google.com` shows table
- [ ] `--no-color` disables colors
- [ ] Piped output has no colors

---

## Performance Targets
- Text formatting: < 1ms for 30 hops
- Table formatting: < 5ms for 30 hops
- Memory: < 1MB for output generation

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Terminal width issues | Medium | Medium | Implement truncation, test various widths |
| Color compatibility | Low | Low | Provide --no-color flag |
| Table alignment issues | Medium | Low | Thorough testing with various data |

## Notes
- Test output on various terminal emulators (iTerm2, Terminal.app, Windows Terminal, etc.)
- Consider adding a `--width` flag for manual terminal width override
- The table formatter may need responsive column hiding for narrow terminals
- Consider adding sparkline RTT visualization in table format later
