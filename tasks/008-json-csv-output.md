# Feature 008: JSON and CSV Output Formats

**Feature ID:** F008
**Feature Name:** JSON and CSV Output Formats
**Priority:** P2 - HIGH
**Target Version:** v0.3.0
**Estimated Duration:** 0.5 weeks
**Status:** NOT_STARTED

## Overview

Implement JSON and CSV output formatters for structured data export. These formats enable integration with other tools, scripting, log aggregation systems, and data analysis pipelines. JSON provides hierarchical data with full detail, while CSV offers flat tabular data for spreadsheet compatibility.

## Goals
- Implement JSON output with pretty-print option
- Implement CSV output with configurable columns
- Ensure all enrichment data is included
- Maintain consistency with text output data

## Success Criteria
- [ ] All tasks completed (T049-T053)
- [ ] JSON output parses correctly with jq
- [ ] CSV output opens correctly in Excel
- [ ] All hop data fields are present
- [ ] Pretty-print JSON is human-readable

## Tasks

### T049: Implement JSON Formatter

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Implement JSON output formatter that serializes TraceResult to JSON format with optional pretty-printing.

#### Technical Details
```go
// internal/output/json.go
type JSONFormatter struct {
    config FormatterConfig
    pretty bool
}

func NewJSONFormatter(config FormatterConfig, pretty bool) *JSONFormatter {
    return &JSONFormatter{
        config: config,
        pretty: pretty,
    }
}

func (f *JSONFormatter) Format(result *trace.TraceResult) ([]byte, error) {
    // Convert to output format (may differ slightly from internal format)
    output := JSONOutput{
        Target:      result.Target,
        ResolvedIP:  result.ResolvedIP.String(),
        Timestamp:   result.Timestamp.Format(time.RFC3339),
        ProbeMethod: result.ProbeMethod,
        Hops:        make([]JSONHop, len(result.Hops)),
        Summary:     f.formatSummary(result.Summary),
    }
    
    for i, hop := range result.Hops {
        output.Hops[i] = f.formatHop(hop)
    }
    
    if f.pretty {
        return json.MarshalIndent(output, "", "  ")
    }
    return json.Marshal(output)
}

type JSONOutput struct {
    Target      string      `json:"target"`
    ResolvedIP  string      `json:"resolved_ip"`
    Timestamp   string      `json:"timestamp"`
    ProbeMethod string      `json:"probe_method"`
    Hops        []JSONHop   `json:"hops"`
    Summary     JSONSummary `json:"summary"`
}

type JSONHop struct {
    Hop         int       `json:"hop"`
    IP          string    `json:"ip,omitempty"`
    Hostname    string    `json:"hostname,omitempty"`
    ASN         *JSONASN  `json:"asn,omitempty"`
    Geo         *JSONGeo  `json:"geo,omitempty"`
    RTTs        []float64 `json:"rtts"`
    AvgRTT      float64   `json:"avg_rtt_ms"`
    MinRTT      float64   `json:"min_rtt_ms"`
    MaxRTT      float64   `json:"max_rtt_ms"`
    Jitter      float64   `json:"jitter_ms"`
    LossPercent float64   `json:"loss_percent"`
    Responded   bool      `json:"responded"`
}

func (f *JSONFormatter) formatHop(hop trace.Hop) JSONHop {
    jh := JSONHop{
        Hop:         hop.Number,
        RTTs:        hop.RTTs,
        AvgRTT:      roundFloat(hop.AvgRTT, 3),
        MinRTT:      roundFloat(hop.MinRTT, 3),
        MaxRTT:      roundFloat(hop.MaxRTT, 3),
        Jitter:      roundFloat(hop.Jitter, 3),
        LossPercent: roundFloat(hop.LossPercent, 1),
        Responded:   hop.Responded,
    }
    
    if hop.IP != nil {
        jh.IP = hop.IP.String()
    }
    jh.Hostname = hop.Hostname
    
    if hop.ASN != nil {
        jh.ASN = &JSONASN{
            Number: hop.ASN.Number,
            Org:    hop.ASN.Org,
        }
    }
    
    if hop.Geo != nil {
        jh.Geo = &JSONGeo{
            Country:     hop.Geo.Country,
            CountryCode: hop.Geo.CountryCode,
            City:        hop.Geo.City,
            Latitude:    hop.Geo.Latitude,
            Longitude:   hop.Geo.Longitude,
        }
    }
    
    return jh
}

func (f *JSONFormatter) ContentType() string {
    return "application/json"
}

func (f *JSONFormatter) FileExtension() string {
    return "json"
}
```

#### Files to Touch
- `internal/output/json.go` (new)
- `internal/output/json_test.go` (new)

#### Dependencies
- T020: Formatter interface

#### Success Criteria
- [ ] Valid JSON output
- [ ] Pretty-print option works
- [ ] All fields serialized correctly
- [ ] Handles nil values properly

---

### T050: Implement CSV Formatter

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Implement CSV output formatter with one row per hop and configurable columns.

#### Technical Details
```go
// internal/output/csv.go
type CSVFormatter struct {
    config  FormatterConfig
    columns []string
}

var DefaultCSVColumns = []string{
    "hop", "ip", "hostname", "asn", "org", "country", "city",
    "avg_rtt_ms", "min_rtt_ms", "max_rtt_ms", "jitter_ms", "loss_percent",
}

func NewCSVFormatter(config FormatterConfig) *CSVFormatter {
    return &CSVFormatter{
        config:  config,
        columns: DefaultCSVColumns,
    }
}

func (f *CSVFormatter) Format(result *trace.TraceResult) ([]byte, error) {
    var buf bytes.Buffer
    writer := csv.NewWriter(&buf)
    
    // Write header
    if err := writer.Write(f.columns); err != nil {
        return nil, err
    }
    
    // Write rows
    for _, hop := range result.Hops {
        row := f.formatRow(hop)
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

func (f *CSVFormatter) formatRow(hop trace.Hop) []string {
    row := make([]string, len(f.columns))
    
    for i, col := range f.columns {
        row[i] = f.getValue(hop, col)
    }
    
    return row
}

func (f *CSVFormatter) getValue(hop trace.Hop, column string) string {
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
    default:
        return ""
    }
}

func formatFloat(f float64) string {
    if f <= 0 {
        return ""
    }
    return fmt.Sprintf("%.3f", f)
}

func (f *CSVFormatter) ContentType() string {
    return "text/csv"
}

func (f *CSVFormatter) FileExtension() string {
    return "csv"
}
```

#### Files to Touch
- `internal/output/csv.go` (new)
- `internal/output/csv_test.go` (new)

#### Dependencies
- T020: Formatter interface

#### Success Criteria
- [ ] Valid CSV output
- [ ] Headers match data
- [ ] Handles special characters (quotes, commas)
- [ ] Opens correctly in Excel

---

### T051: Add CLI Flags for JSON/CSV

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Add command-line flags for JSON and CSV output selection.

#### Technical Details
```go
// cmd/poros/root.go (update)
func init() {
    // Output format flags
    rootCmd.Flags().BoolP("json", "j", false, "Output in JSON format")
    rootCmd.Flags().Bool("pretty", false, "Pretty-print JSON output")
    rootCmd.Flags().Bool("csv", false, "Output in CSV format")
    rootCmd.Flags().StringP("output", "o", "", "Write output to file")
}

func getOutputFormat(cmd *cobra.Command) output.OutputFormat {
    if getBool(cmd, "json") {
        return output.FormatJSON
    }
    if getBool(cmd, "csv") {
        return output.FormatCSV
    }
    if getBool(cmd, "verbose") {
        return output.FormatVerbose
    }
    return output.FormatText
}

func runTrace(cmd *cobra.Command, args []string) error {
    // ... trace execution ...
    
    format := getOutputFormat(cmd)
    config := output.FormatterConfig{
        Colors: !getBool(cmd, "no-color") && !getBool(cmd, "json") && !getBool(cmd, "csv"),
    }
    
    var formatter output.Formatter
    switch format {
    case output.FormatJSON:
        formatter = output.NewJSONFormatter(config, getBool(cmd, "pretty"))
    case output.FormatCSV:
        formatter = output.NewCSVFormatter(config)
    default:
        // ...
    }
    
    data, err := formatter.Format(result)
    if err != nil {
        return err
    }
    
    // Handle output destination
    outputFile := getString(cmd, "output")
    if outputFile != "" {
        return os.WriteFile(outputFile, data, 0644)
    }
    
    _, err = os.Stdout.Write(data)
    return err
}
```

#### Files to Touch
- `cmd/poros/root.go` (update)
- `cmd/poros/flags.go` (update)

#### Dependencies
- T049: JSON formatter
- T050: CSV formatter

#### Success Criteria
- [ ] `-j/--json` enables JSON output
- [ ] `--pretty` works with JSON
- [ ] `--csv` enables CSV output
- [ ] `-o` writes to file

---

### T052: Add File Output Support

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Implement file output with automatic extension handling and overwrite protection.

#### Technical Details
```go
// internal/output/file.go
type FileWriter struct {
    path      string
    formatter Formatter
    overwrite bool
}

func NewFileWriter(path string, formatter Formatter, overwrite bool) *FileWriter {
    return &FileWriter{
        path:      path,
        formatter: formatter,
        overwrite: overwrite,
    }
}

func (w *FileWriter) Write(result *trace.TraceResult) error {
    // Add extension if missing
    path := w.path
    ext := w.formatter.FileExtension()
    if filepath.Ext(path) == "" {
        path = path + "." + ext
    }
    
    // Check if file exists
    if !w.overwrite {
        if _, err := os.Stat(path); err == nil {
            return fmt.Errorf("file %s already exists (use --force to overwrite)", path)
        }
    }
    
    // Format data
    data, err := w.formatter.Format(result)
    if err != nil {
        return fmt.Errorf("format error: %w", err)
    }
    
    // Write to file
    if err := os.WriteFile(path, data, 0644); err != nil {
        return fmt.Errorf("write error: %w", err)
    }
    
    return nil
}

// CLI flag
rootCmd.Flags().Bool("force", false, "Overwrite existing output file")
```

#### Files to Touch
- `internal/output/file.go` (new)
- `cmd/poros/root.go` (update)

#### Dependencies
- T051: CLI flags for output

#### Success Criteria
- [ ] Creates output file
- [ ] Adds extension automatically
- [ ] Prevents accidental overwrite
- [ ] `--force` allows overwrite

---

### T053: Add Output Format Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create comprehensive tests for JSON and CSV output formatting.

#### Technical Details
```go
// internal/output/json_test.go
func TestJSONFormatter(t *testing.T) {
    result := &trace.TraceResult{
        Target:     "google.com",
        ResolvedIP: net.ParseIP("142.250.185.238"),
        Timestamp:  time.Now(),
        Hops: []trace.Hop{
            {
                Number:   1,
                IP:       net.ParseIP("192.168.1.1"),
                Hostname: "router.local",
                RTTs:     []float64{1.234, 1.456, 1.123},
                AvgRTT:   1.271,
                MinRTT:   1.123,
                MaxRTT:   1.456,
            },
        },
    }
    
    formatter := NewJSONFormatter(FormatterConfig{}, false)
    data, err := formatter.Format(result)
    require.NoError(t, err)
    
    // Verify valid JSON
    var parsed JSONOutput
    err = json.Unmarshal(data, &parsed)
    require.NoError(t, err)
    
    assert.Equal(t, "google.com", parsed.Target)
    assert.Len(t, parsed.Hops, 1)
    assert.Equal(t, "192.168.1.1", parsed.Hops[0].IP)
}

func TestJSONFormatter_PrettyPrint(t *testing.T) {
    // Verify indentation
    formatter := NewJSONFormatter(FormatterConfig{}, true)
    data, err := formatter.Format(result)
    require.NoError(t, err)
    
    assert.Contains(t, string(data), "\n")
    assert.Contains(t, string(data), "  ")
}

// internal/output/csv_test.go
func TestCSVFormatter(t *testing.T) {
    formatter := NewCSVFormatter(FormatterConfig{})
    data, err := formatter.Format(result)
    require.NoError(t, err)
    
    // Parse CSV
    reader := csv.NewReader(bytes.NewReader(data))
    records, err := reader.ReadAll()
    require.NoError(t, err)
    
    // Verify header
    assert.Equal(t, "hop", records[0][0])
    assert.Equal(t, "ip", records[0][1])
    
    // Verify data row
    assert.Equal(t, "1", records[1][0])
    assert.Equal(t, "192.168.1.1", records[1][1])
}
```

#### Files to Touch
- `internal/output/json_test.go` (update)
- `internal/output/csv_test.go` (update)

#### Dependencies
- T049: JSON formatter
- T050: CSV formatter

#### Success Criteria
- [ ] JSON parsing tests pass
- [ ] CSV parsing tests pass
- [ ] Edge cases covered
- [ ] Format consistency verified

---

## Performance Targets
- JSON formatting: < 1ms for 30 hops
- CSV formatting: < 1ms for 30 hops
- File write: < 10ms

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Large JSON output | Low | Low | Consider streaming for huge traces |
| CSV special chars | Medium | Low | Proper escaping in csv package |

## Notes
- JSON output is essential for scripting and automation
- Consider adding JSONL (line-delimited) for streaming later
- CSV column order should match common analysis workflows
