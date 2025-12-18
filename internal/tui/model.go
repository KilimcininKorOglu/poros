// Package tui provides an interactive terminal UI for traceroute.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// State represents the current state of the TUI.
type State int

const (
	StateRunning State = iota
	StateComplete
	StateError
)

// Model is the Bubble Tea model for the traceroute TUI.
type Model struct {
	// Configuration
	target string
	config *trace.Config
	width  int
	height int

	// State
	state     State
	hops      []trace.Hop
	err       error
	elapsed   time.Duration
	startTime time.Time

	// UI components
	spinner spinner.Model

	// Styles
	styles Styles

	// Channel for hop updates
	hopChan chan trace.Hop
}

// HopMsg is sent when a new hop is discovered.
type HopMsg struct {
	Hop trace.Hop
}

// CompleteMsg is sent when the trace is complete.
type CompleteMsg struct {
	Result *trace.TraceResult
}

// ErrorMsg is sent when an error occurs.
type ErrorMsg struct {
	Err error
}

// TickMsg is sent to update elapsed time.
type TickMsg time.Time

// New creates a new TUI model.
func New(target string, config *trace.Config) (*Model, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	m := &Model{
		target:    target,
		config:    config,
		state:     StateRunning,
		hops:      make([]trace.Hop, 0),
		spinner:   s,
		styles:    DefaultStyles(),
		width:     80,
		height:    24,
		startTime: time.Now(),
		hopChan:   make(chan trace.Hop, 100),
	}

	return m, nil
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runTrace(),
		m.tickCmd(),
		m.waitForHop(),
	)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case TickMsg:
		m.elapsed = time.Since(m.startTime)
		if m.state == StateRunning {
			return m, m.tickCmd()
		}

	case HopMsg:
		m.hops = append(m.hops, msg.Hop)
		// Continue waiting for more hops
		return m, m.waitForHop()

	case CompleteMsg:
		m.state = StateComplete
		// Don't replace hops - they've been added via HopMsg

	case ErrorMsg:
		m.state = StateError
		m.err = msg.Err
		return m, tea.Quit
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n\n")

	// Hop table
	b.WriteString(m.renderHops())

	// Footer
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

// renderHeader renders the header section.
func (m Model) renderHeader() string {
	title := m.styles.Title.Render("Poros Traceroute")

	var status string
	switch m.state {
	case StateRunning:
		status = m.spinner.View() + " Tracing..."
	case StateComplete:
		status = m.styles.Success.Render("✓ Complete")
	case StateError:
		status = m.styles.Error.Render("✗ Error")
	}

	info := fmt.Sprintf("Target: %s | Method: %s", m.target, m.config.ProbeMethod)

	return lipgloss.JoinVertical(lipgloss.Left,
		title,
		m.styles.Subtle.Render(info),
		status,
	)
}

// renderHops renders the hop table.
func (m Model) renderHops() string {
	if len(m.hops) == 0 {
		return m.styles.Subtle.Render("Waiting for responses...")
	}

	var rows []string

	// Header row
	header := fmt.Sprintf("%-4s %-15s %-25s %-10s %-10s %-10s",
		"Hop", "IP", "Hostname", "Avg", "Min", "Max")
	rows = append(rows, m.styles.Header.Render(header))

	// Separator
	rows = append(rows, m.styles.Subtle.Render(strings.Repeat("─", 80)))

	// Hop rows
	for _, hop := range m.hops {
		rows = append(rows, m.renderHopRow(hop))
	}

	return strings.Join(rows, "\n")
}

// renderHopRow renders a single hop row.
func (m Model) renderHopRow(hop trace.Hop) string {
	hopNum := fmt.Sprintf("%-4d", hop.Number)

	var ip, hostname, avg, min, max string

	if !hop.Responded {
		ip = "*"
		hostname = ""
		avg = "*"
		min = "*"
		max = "*"
	} else {
		if hop.IP != nil {
			ip = hop.IP.String()
		} else {
			ip = "*"
		}
		hostname = truncate(hop.Hostname, 25)

		if hop.AvgRTT > 0 {
			avg = fmt.Sprintf("%.2f ms", hop.AvgRTT)
			min = fmt.Sprintf("%.2f", hop.MinRTT)
			max = fmt.Sprintf("%.2f", hop.MaxRTT)
		} else {
			avg = "-"
			min = "-"
			max = "-"
		}
	}

	// Color RTT based on latency
	avgStyled := m.colorizeRTT(avg, hop.AvgRTT)

	return fmt.Sprintf("%-4s %-15s %-25s %-10s %-10s %-10s",
		m.styles.HopNum.Render(hopNum),
		m.styles.IP.Render(truncate(ip, 15)),
		m.styles.Hostname.Render(hostname),
		avgStyled,
		m.styles.Subtle.Render(min),
		m.styles.Subtle.Render(max),
	)
}

// colorizeRTT applies color based on latency.
func (m Model) colorizeRTT(s string, rtt float64) string {
	if rtt <= 0 {
		return m.styles.Subtle.Render(s)
	}

	switch {
	case rtt < 50:
		return m.styles.RTTLow.Render(s)
	case rtt < 150:
		return m.styles.RTTMed.Render(s)
	default:
		return m.styles.RTTHigh.Render(s)
	}
}

// renderFooter renders the footer section.
func (m Model) renderFooter() string {
	var parts []string

	if m.state == StateComplete {
		parts = append(parts, fmt.Sprintf("Hops: %d", len(m.hops)))
		if len(m.hops) > 0 && m.hops[len(m.hops)-1].AvgRTT > 0 {
			parts = append(parts, fmt.Sprintf("Total: %.2f ms", m.hops[len(m.hops)-1].AvgRTT))
		}
	}

	parts = append(parts, "Press 'q' to quit")

	return m.styles.Subtle.Render(strings.Join(parts, " | "))
}

// runTrace runs the traceroute in the background.
func (m Model) runTrace() tea.Cmd {
	return func() tea.Msg {
		// Set up OnHop callback to stream hops to channel
		m.config.OnHop = func(hop *trace.Hop) {
			m.hopChan <- *hop
		}

		// Create tracer with callback
		tracer, err := trace.New(m.config)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		defer tracer.Close()

		ctx := context.Background()
		result, err := tracer.Trace(ctx, m.target)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return CompleteMsg{Result: result}
	}
}

// waitForHop waits for a hop from the channel.
func (m Model) waitForHop() tea.Cmd {
	return func() tea.Msg {
		hop, ok := <-m.hopChan
		if !ok {
			return nil
		}
		return HopMsg{Hop: hop}
	}
}

// tickCmd returns a command that sends tick messages.
func (m Model) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// Close releases resources.
func (m *Model) Close() error {
	if m.hopChan != nil {
		close(m.hopChan)
	}
	return nil
}

// truncate truncates a string to maxLen.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
