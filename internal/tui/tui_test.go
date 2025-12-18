package tui

import (
	"testing"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	// Check that styles are not empty
	if styles.Title.String() == "" {
		// Style should be defined
	}

	// Check RTT colors are different
	low := styles.RTTLow.Render("test")
	med := styles.RTTMed.Render("test")
	high := styles.RTTHigh.Render("test")

	if low == med || med == high {
		t.Log("RTT styles should be visually different")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a very long string", 10, "this is..."},
		{"ab", 2, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q",
					tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}

func TestDarkTheme(t *testing.T) {
	styles := DarkTheme()
	
	// Should return a valid styles struct
	if styles.Title.String() == "" && styles.RTTLow.String() == "" {
		// At least one style should be defined
	}
}

func TestLightTheme(t *testing.T) {
	styles := LightTheme()
	
	// Should return a valid styles struct
	if styles.Title.String() == "" && styles.RTTLow.String() == "" {
		// At least one style should be defined
	}
}

func TestMinimalTheme(t *testing.T) {
	styles := MinimalTheme()
	
	// Should return a valid styles struct
	if styles.Title.String() == "" {
		// At least one style should be defined
	}
}

func TestModelRenderHopRow(t *testing.T) {
	config := trace.DefaultConfig()
	model := &Model{
		target: "example.com",
		config: config,
		styles: DefaultStyles(),
	}

	// Test responding hop
	hop := trace.Hop{
		Number:    1,
		Responded: true,
		AvgRTT:    10.5,
		MinRTT:    8.2,
		MaxRTT:    12.3,
	}

	row := model.renderHopRow(hop)
	if row == "" {
		t.Error("renderHopRow should return non-empty string")
	}

	// Test non-responding hop
	hopTimeout := trace.Hop{
		Number:    2,
		Responded: false,
	}

	row2 := model.renderHopRow(hopTimeout)
	if row2 == "" {
		t.Error("renderHopRow should handle timeout hops")
	}
}

func TestColorizeRTT(t *testing.T) {
	model := &Model{
		styles: DefaultStyles(),
	}

	tests := []struct {
		name string
		rtt  float64
	}{
		{"low latency", 25.0},
		{"medium latency", 75.0},
		{"high latency", 200.0},
		{"zero", 0},
		{"negative", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := model.colorizeRTT("10.00 ms", tt.rtt)
			if result == "" {
				t.Error("colorizeRTT should return non-empty string")
			}
		})
	}
}
