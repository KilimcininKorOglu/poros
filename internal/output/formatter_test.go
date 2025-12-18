package output

import (
	"encoding/csv"
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// Helper function to create a sample trace result
func sampleTraceResult() *trace.TraceResult {
	return &trace.TraceResult{
		Target:      "google.com",
		ResolvedIP:  net.ParseIP("142.250.185.238"),
		Timestamp:   time.Date(2025, 12, 18, 12, 0, 0, 0, time.UTC),
		ProbeMethod: "icmp",
		Completed:   true,
		Hops: []trace.Hop{
			{
				Number:      1,
				IP:          net.ParseIP("192.168.1.1"),
				Hostname:    "router.local",
				RTTs:        []float64{1.234, 1.456, 1.123},
				AvgRTT:      1.271,
				MinRTT:      1.123,
				MaxRTT:      1.456,
				Jitter:      0.333,
				LossPercent: 0,
				Responded:   true,
			},
			{
				Number:      2,
				IP:          net.ParseIP("10.0.0.1"),
				Hostname:    "",
				RTTs:        []float64{5.678, -1, 5.432},
				AvgRTT:      5.555,
				MinRTT:      5.432,
				MaxRTT:      5.678,
				Jitter:      0.246,
				LossPercent: 33.33,
				Responded:   true,
				ASN: &trace.ASNInfo{
					Number: 15169,
					Org:    "Google LLC",
				},
			},
			{
				Number:      3,
				RTTs:        []float64{-1, -1, -1},
				LossPercent: 100,
				Responded:   false,
			},
		},
		Summary: trace.Summary{
			TotalHops:         3,
			TotalTimeMs:       5.555,
			PacketLossPercent: 44.44,
		},
	}
}

func TestTextFormatter(t *testing.T) {
	config := Config{Colors: false}
	formatter := NewTextFormatter(config)

	result := sampleTraceResult()
	data, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(data)

	// Check header
	if !strings.Contains(output, "traceroute to google.com") {
		t.Error("Output should contain target in header")
	}

	// Check hop 1
	if !strings.Contains(output, "192.168.1.1") {
		t.Error("Output should contain hop 1 IP")
	}
	if !strings.Contains(output, "router.local") {
		t.Error("Output should contain hop 1 hostname")
	}

	// Check hop 2 with ASN
	if !strings.Contains(output, "10.0.0.1") {
		t.Error("Output should contain hop 2 IP")
	}
	if !strings.Contains(output, "AS15169") {
		t.Error("Output should contain ASN")
	}

	// Check hop 3 (timeout)
	if !strings.Contains(output, "* * *") {
		t.Error("Output should contain timeout markers")
	}

	// Check summary
	if !strings.Contains(output, "complete") || !strings.Contains(output, "3 hops") {
		t.Error("Output should contain summary")
	}
}

func TestTableFormatter(t *testing.T) {
	config := Config{Colors: false}
	formatter := NewTableFormatter(config)

	result := sampleTraceResult()
	data, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(data)

	// Check header
	if !strings.Contains(output, "Target: google.com") {
		t.Error("Output should contain target")
	}

	// Check table structure
	if !strings.Contains(output, "HOP") {
		t.Error("Output should contain HOP column")
	}
	if !strings.Contains(output, "IP ADDRESS") {
		t.Error("Output should contain IP ADDRESS column")
	}

	// Check data
	if !strings.Contains(output, "192.168.1.1") {
		t.Error("Output should contain hop IP")
	}

	// Check summary
	if !strings.Contains(output, "Total Hops") {
		t.Error("Output should contain summary")
	}
}

func TestJSONFormatter(t *testing.T) {
	config := Config{}
	formatter := NewJSONFormatter(config)

	result := sampleTraceResult()
	data, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed JSONOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parsing error: %v", err)
	}

	// Check values
	if parsed.Target != "google.com" {
		t.Errorf("Target = %q, want %q", parsed.Target, "google.com")
	}

	if len(parsed.Hops) != 3 {
		t.Errorf("len(Hops) = %d, want 3", len(parsed.Hops))
	}

	if parsed.Hops[0].IP != "192.168.1.1" {
		t.Errorf("Hops[0].IP = %q, want %q", parsed.Hops[0].IP, "192.168.1.1")
	}

	if parsed.Hops[1].ASN == nil {
		t.Error("Hops[1].ASN should not be nil")
	} else if parsed.Hops[1].ASN.Number != 15169 {
		t.Errorf("Hops[1].ASN.Number = %d, want 15169", parsed.Hops[1].ASN.Number)
	}

	if parsed.Completed != true {
		t.Error("Completed should be true")
	}
}

func TestJSONFormatterCompact(t *testing.T) {
	config := Config{}
	formatter := NewJSONFormatterCompact(config)

	result := sampleTraceResult()
	data, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Compact JSON should not have newlines (except in strings)
	lines := strings.Split(string(data), "\n")
	if len(lines) > 1 {
		// Allow trailing newline
		if len(lines) > 2 || lines[1] != "" {
			t.Error("Compact JSON should be on single line")
		}
	}
}

func TestCSVFormatter(t *testing.T) {
	config := Config{}
	formatter := NewCSVFormatter(config)

	result := sampleTraceResult()
	data, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	// Parse CSV
	reader := csv.NewReader(strings.NewReader(string(data)))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parsing error: %v", err)
	}

	// Check header
	if records[0][0] != "hop" {
		t.Errorf("Header[0] = %q, want %q", records[0][0], "hop")
	}
	if records[0][1] != "ip" {
		t.Errorf("Header[1] = %q, want %q", records[0][1], "ip")
	}

	// Check data rows (header + 3 hops)
	if len(records) != 4 {
		t.Errorf("len(records) = %d, want 4", len(records))
	}

	// Check first data row
	if records[1][0] != "1" {
		t.Errorf("Row 1 hop = %q, want %q", records[1][0], "1")
	}
	if records[1][1] != "192.168.1.1" {
		t.Errorf("Row 1 IP = %q, want %q", records[1][1], "192.168.1.1")
	}
}

func TestNewFormatter(t *testing.T) {
	config := DefaultConfig()

	tests := []struct {
		format   Format
		expected string
	}{
		{FormatText, "text/plain"},
		{FormatVerbose, "text/plain"},
		{FormatJSON, "application/json"},
		{FormatCSV, "text/csv"},
		{FormatHTML, "text/html"},
	}

	for _, tt := range tests {
		t.Run(tt.format.String(), func(t *testing.T) {
			formatter := NewFormatter(tt.format, config)
			if formatter.ContentType() != tt.expected {
				t.Errorf("ContentType() = %q, want %q", formatter.ContentType(), tt.expected)
			}
		})
	}
}

func TestHTMLFormatter(t *testing.T) {
	config := Config{Colors: false}
	formatter := NewHTMLFormatter(config)

	result := sampleTraceResult()
	data, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	output := string(data)

	// Check DOCTYPE
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("Output should contain DOCTYPE")
	}

	// Check title
	if !strings.Contains(output, "google.com") {
		t.Error("Output should contain target")
	}

	// Check hop data
	if !strings.Contains(output, "192.168.1.1") {
		t.Error("Output should contain hop IP")
	}

	// Check CSS
	if !strings.Contains(output, "<style>") {
		t.Error("Output should contain embedded CSS")
	}

	// Check summary
	if !strings.Contains(output, "Total Hops") {
		t.Error("Output should contain summary")
	}
}

func TestHTMLFormatter_RTTClass(t *testing.T) {
	tests := []struct {
		rtt      float64
		expected string
	}{
		{0, "neutral"},
		{-1, "neutral"},
		{25, "good"},
		{75, "medium"},
		{200, "bad"},
	}

	for _, tt := range tests {
		result := rttClass(tt.rtt)
		if result != tt.expected {
			t.Errorf("rttClass(%v) = %q, want %q", tt.rtt, result, tt.expected)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a long string", 10, "this is..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncateString(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		input     float64
		precision int
		expected  float64
	}{
		{1.2345, 2, 1.23},
		{1.2355, 2, 1.24},
		{1.5, 0, 2},
		{1.4, 0, 1},
		{1.23456789, 3, 1.235},
	}

	for _, tt := range tests {
		result := roundFloat(tt.input, tt.precision)
		if result != tt.expected {
			t.Errorf("roundFloat(%v, %d) = %v, want %v",
				tt.input, tt.precision, result, tt.expected)
		}
	}
}
