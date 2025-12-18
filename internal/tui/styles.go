package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the styles used in the TUI.
type Styles struct {
	// Text styles
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Header   lipgloss.Style
	Subtle   lipgloss.Style

	// Status styles
	Success lipgloss.Style
	Error   lipgloss.Style
	Warning lipgloss.Style

	// Hop styles
	HopNum   lipgloss.Style
	IP       lipgloss.Style
	Hostname lipgloss.Style
	Timeout  lipgloss.Style

	// RTT styles (color-coded by latency)
	RTTLow  lipgloss.Style // < 50ms
	RTTMed  lipgloss.Style // 50-150ms
	RTTHigh lipgloss.Style // > 150ms

	// Enrichment styles
	ASN    lipgloss.Style
	GeoIP  lipgloss.Style

	// Container styles
	Box       lipgloss.Style
	StatusBar lipgloss.Style
}

// DefaultStyles returns the default style set.
func DefaultStyles() Styles {
	return Styles{
		// Text styles
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1),

		Subtitle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("255")),

		Subtle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")),

		// Status styles
		Success: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("46")), // Green

		Error: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")), // Red

		Warning: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("214")), // Orange

		// Hop styles
		HopNum: lipgloss.NewStyle().
			Foreground(lipgloss.Color("87")), // Cyan

		IP: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")), // White

		Hostname: lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")), // Light green

		Timeout: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red

		// RTT styles
		RTTLow: lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")), // Green

		RTTMed: lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")), // Yellow

		RTTHigh: lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")), // Red

		// Enrichment styles
		ASN: lipgloss.NewStyle().
			Foreground(lipgloss.Color("141")), // Purple

		GeoIP: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")), // Blue

		// Container styles
		Box: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 2),

		StatusBar: lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1),
	}
}

// DarkTheme returns a dark theme style set.
func DarkTheme() Styles {
	return DefaultStyles()
}

// LightTheme returns a light theme style set.
func LightTheme() Styles {
	s := DefaultStyles()

	// Adjust colors for light backgrounds
	s.Subtle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	s.Header = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("0"))
	s.IP = lipgloss.NewStyle().Foreground(lipgloss.Color("0"))

	return s
}

// MinimalTheme returns a minimal style set with fewer colors.
func MinimalTheme() Styles {
	s := DefaultStyles()

	// Use fewer, more muted colors
	s.Title = lipgloss.NewStyle().Bold(true)
	s.HopNum = lipgloss.NewStyle().Bold(true)
	s.IP = lipgloss.NewStyle()
	s.Hostname = lipgloss.NewStyle().Italic(true)

	return s
}
