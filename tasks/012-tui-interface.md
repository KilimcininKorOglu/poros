# Feature 012: TUI (Text User Interface)

**Feature ID:** F012
**Feature Name:** TUI (Text User Interface)
**Priority:** P2 - HIGH
**Target Version:** v0.5.0
**Estimated Duration:** 2 weeks
**Status:** NOT_STARTED

## Overview

Implement an interactive Text User Interface using Bubble Tea framework. The TUI provides real-time trace updates, visual latency indicators, keyboard navigation, and an enhanced user experience for interactive traceroute sessions.

The TUI is particularly valuable for continuous monitoring, troubleshooting, and presenting trace information in a visually appealing format.

## Goals
- Create interactive TUI with Bubble Tea
- Show real-time trace progress
- Display latency bar graphs
- Implement keyboard shortcuts
- Support continuous tracing mode

## Success Criteria
- [ ] All tasks completed (T076-T084)
- [ ] TUI displays trace progress in real-time
- [ ] Keyboard shortcuts work reliably
- [ ] Colors and formatting are consistent
- [ ] TUI gracefully handles terminal resize

## Tasks

### T076: Set Up Bubble Tea Application Structure

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1 day

#### Description
Create the basic Bubble Tea application structure with model, update, and view components.

#### Technical Details
```go
// internal/tui/app.go
import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type App struct {
    program *tea.Program
}

func New(config Config) *App {
    model := NewModel(config)
    program := tea.NewProgram(model, tea.WithAltScreen())
    
    return &App{
        program: program,
    }
}

func (a *App) Run() error {
    _, err := a.program.Run()
    return err
}

// internal/tui/model.go
type Model struct {
    config     Config
    target     string
    hops       []HopView
    status     string
    err        error
    width      int
    height     int
    tracing    bool
    completed  bool
    startTime  time.Time
}

type HopView struct {
    Number    int
    IP        string
    Hostname  string
    ASN       string
    RTTs      []float64
    AvgRTT    float64
    Status    HopStatus
}

type HopStatus int

const (
    HopPending HopStatus = iota
    HopProbing
    HopComplete
    HopTimeout
)

func NewModel(config Config) Model {
    return Model{
        config:    config,
        target:    config.Target,
        hops:      make([]HopView, config.MaxHops),
        status:    "Initializing...",
        startTime: time.Now(),
    }
}

func (m Model) Init() tea.Cmd {
    return tea.Batch(
        startTrace(m.config),
        tickCmd(),
    )
}
```

#### Files to Touch
- `internal/tui/app.go` (new)
- `internal/tui/model.go` (new)
- `internal/tui/config.go` (new)

#### Dependencies
- T006: Bubble Tea dependency

#### Success Criteria
- [ ] App initializes correctly
- [ ] Model holds trace state
- [ ] Basic structure compiles

---

### T077: Implement TUI Update Logic

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description
Implement the Update function that handles messages for trace updates, user input, and system events.

#### Technical Details
```go
// internal/tui/update.go
type HopUpdateMsg struct {
    Hop   int
    IP    net.IP
    RTT   time.Duration
    Final bool
}

type TraceCompleteMsg struct {
    Result *trace.TraceResult
}

type ErrorMsg struct {
    Err error
}

type TickMsg time.Time

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)
    
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    
    case HopUpdateMsg:
        return m.handleHopUpdate(msg)
    
    case TraceCompleteMsg:
        m.tracing = false
        m.completed = true
        m.status = "Trace complete"
        return m, nil
    
    case ErrorMsg:
        m.err = msg.Err
        m.tracing = false
        m.status = fmt.Sprintf("Error: %v", msg.Err)
        return m, nil
    
    case TickMsg:
        // Update elapsed time display
        return m, tickCmd()
    }
    
    return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "q", "ctrl+c":
        return m, tea.Quit
    
    case "r":
        // Restart trace
        if !m.tracing {
            m.hops = make([]HopView, m.config.MaxHops)
            m.tracing = true
            m.completed = false
            m.startTime = time.Now()
            return m, startTrace(m.config)
        }
    
    case "p":
        // Pause/resume (for continuous mode)
        m.paused = !m.paused
        return m, nil
    
    case "e":
        // Export results
        return m, exportResults(m)
    
    case "m":
        // Toggle probe method
        return m, nil
    
    case "+", "=":
        // Increase probes
        m.config.ProbeCount = min(10, m.config.ProbeCount+1)
        return m, nil
    
    case "-":
        // Decrease probes
        m.config.ProbeCount = max(1, m.config.ProbeCount-1)
        return m, nil
    }
    
    return m, nil
}

func (m Model) handleHopUpdate(msg HopUpdateMsg) (Model, tea.Cmd) {
    if msg.Hop <= len(m.hops) && msg.Hop > 0 {
        hop := &m.hops[msg.Hop-1]
        hop.Number = msg.Hop
        
        if msg.IP != nil {
            hop.IP = msg.IP.String()
            hop.Status = HopComplete
        } else {
            hop.Status = HopTimeout
        }
        
        if msg.RTT > 0 {
            rtt := float64(msg.RTT.Microseconds()) / 1000.0
            hop.RTTs = append(hop.RTTs, rtt)
            hop.AvgRTT = average(hop.RTTs)
        }
    }
    
    return m, nil
}

func tickCmd() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}
```

#### Files to Touch
- `internal/tui/update.go` (new)
- `internal/tui/messages.go` (new)
- `internal/tui/commands.go` (new)

#### Dependencies
- T076: App structure

#### Success Criteria
- [ ] Key handling works
- [ ] Hop updates display correctly
- [ ] Window resize handled
- [ ] Quit works cleanly

---

### T078: Implement TUI View Rendering

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 2 days

#### Description
Implement the View function that renders the TUI interface with header, hop list, and status bar.

#### Technical Details
```go
// internal/tui/view.go
import "github.com/charmbracelet/lipgloss"

var (
    headerStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("39")).
        Border(lipgloss.NormalBorder()).
        BorderForeground(lipgloss.Color("240")).
        Padding(0, 1)
    
    hopStyle = lipgloss.NewStyle().
        Padding(0, 1)
    
    selectedHopStyle = hopStyle.Copy().
        Background(lipgloss.Color("236"))
    
    rttBarStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("42"))
    
    statusStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Padding(0, 1)
)

func (m Model) View() string {
    var b strings.Builder
    
    // Header
    b.WriteString(m.renderHeader())
    b.WriteString("\n\n")
    
    // Hop list
    b.WriteString(m.renderHops())
    b.WriteString("\n")
    
    // Status bar
    b.WriteString(m.renderStatus())
    
    // Help
    b.WriteString(m.renderHelp())
    
    return b.String()
}

func (m Model) renderHeader() string {
    elapsed := time.Since(m.startTime).Round(time.Millisecond)
    
    status := "Tracing..."
    if m.completed {
        status = "Complete"
    }
    
    header := fmt.Sprintf(
        "Target: %s (%s)    Mode: %s    Status: %s    Time: %s",
        m.target,
        m.resolvedIP,
        m.config.ProbeMethod,
        status,
        elapsed,
    )
    
    return headerStyle.Width(m.width - 2).Render(header)
}

func (m Model) renderHops() string {
    var lines []string
    
    for _, hop := range m.hops {
        if hop.Number == 0 {
            continue
        }
        
        line := m.renderHopLine(hop)
        lines = append(lines, line)
        
        // Stop at completed trace
        if hop.Status == HopComplete && hop.IP == m.resolvedIP {
            break
        }
    }
    
    return strings.Join(lines, "\n")
}

func (m Model) renderHopLine(hop HopView) string {
    // Status indicator
    var indicator string
    switch hop.Status {
    case HopPending:
        indicator = "○"
    case HopProbing:
        indicator = "◐"
    case HopComplete:
        indicator = "●"
    case HopTimeout:
        indicator = "✗"
    }
    
    // IP and hostname
    ipStr := hop.IP
    if ipStr == "" {
        ipStr = "*"
    }
    if hop.Hostname != "" {
        ipStr = fmt.Sprintf("%s (%s)", hop.Hostname, hop.IP)
    }
    
    // RTT bar
    rttBar := m.renderRTTBar(hop.AvgRTT)
    
    // ASN info
    asnStr := ""
    if hop.ASN != "" {
        asnStr = lipgloss.NewStyle().
            Foreground(lipgloss.Color("140")).
            Render(hop.ASN)
    }
    
    return fmt.Sprintf("  %s %2d  %-40s %s  %s",
        indicator,
        hop.Number,
        truncate(ipStr, 40),
        rttBar,
        asnStr,
    )
}

func (m Model) renderRTTBar(rtt float64) string {
    if rtt <= 0 {
        return strings.Repeat("░", 20)
    }
    
    // Scale: 0-200ms maps to 0-20 chars
    filled := int(rtt / 10)
    if filled > 20 {
        filled = 20
    }
    
    // Color based on latency
    var color string
    switch {
    case rtt < 50:
        color = "42" // Green
    case rtt < 150:
        color = "226" // Yellow
    default:
        color = "196" // Red
    }
    
    bar := strings.Repeat("█", filled) + strings.Repeat("░", 20-filled)
    rttStr := fmt.Sprintf("%6.1fms", rtt)
    
    return lipgloss.NewStyle().
        Foreground(lipgloss.Color(color)).
        Render(bar + " " + rttStr)
}

func (m Model) renderHelp() string {
    help := "[q] Quit  [r] Retry  [p] Pause  [e] Export  [m] Mode  [+/-] Probes"
    return statusStyle.Width(m.width - 2).Render(help)
}
```

#### Files to Touch
- `internal/tui/view.go` (new)
- `internal/tui/styles.go` (new)
- `internal/tui/components.go` (new)

#### Dependencies
- T077: Update logic

#### Success Criteria
- [ ] Header displays correctly
- [ ] Hop list renders
- [ ] RTT bars show progress
- [ ] Help text visible

---

### T079: Implement Real-Time Trace Updates

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 1.5 days

#### Description
Connect the tracer to the TUI with real-time hop updates via channels and tea.Cmd.

#### Technical Details
```go
// internal/tui/commands.go
func startTrace(config Config) tea.Cmd {
    return func() tea.Msg {
        tracer, err := trace.NewTracer(&trace.TracerConfig{
            ProbeMethod: config.ProbeMethod,
            MaxHops:     config.MaxHops,
            ProbeCount:  config.ProbeCount,
            Timeout:     config.Timeout,
        })
        if err != nil {
            return ErrorMsg{Err: err}
        }
        
        // Create channel for real-time updates
        updateCh := make(chan HopUpdateMsg, 100)
        
        // Start trace in background
        go func() {
            defer close(updateCh)
            
            ctx := context.Background()
            result, err := tracer.TraceWithUpdates(ctx, config.Target, updateCh)
            
            if err != nil {
                // Error handled via channel
                return
            }
            
            // Signal completion
            updateCh <- TraceCompleteMsg{Result: result}
        }()
        
        return listenForUpdates(updateCh)
    }
}

func listenForUpdates(ch <-chan interface{}) tea.Cmd {
    return func() tea.Msg {
        msg, ok := <-ch
        if !ok {
            return TraceCompleteMsg{}
        }
        return msg
    }
}

// internal/trace/tracer.go (update)
func (t *Tracer) TraceWithUpdates(ctx context.Context, target string, 
    updateCh chan<- interface{}) (*TraceResult, error) {
    
    dest, err := t.resolveTarget(ctx, target)
    if err != nil {
        return nil, err
    }
    
    hops := make([]Hop, 0, t.config.MaxHops)
    
    for ttl := t.config.FirstHop; ttl <= t.config.MaxHops; ttl++ {
        // Send "probing" status
        updateCh <- HopUpdateMsg{
            Hop:    ttl,
            Status: HopProbing,
        }
        
        hop := Hop{Number: ttl}
        
        for i := 0; i < t.config.ProbeCount; i++ {
            result, err := t.prober.Probe(ctx, dest, ttl)
            
            // Send update for each probe
            updateCh <- HopUpdateMsg{
                Hop:      ttl,
                IP:       result.ResponseIP,
                RTT:      result.RTT,
                ProbeNum: i,
            }
            
            // ... aggregate results
        }
        
        hops = append(hops, hop)
        
        if hop.isDestination(dest) {
            break
        }
    }
    
    return t.buildResult(target, dest, hops), nil
}
```

#### Files to Touch
- `internal/tui/commands.go` (update)
- `internal/trace/tracer.go` (add TraceWithUpdates)

#### Dependencies
- T078: View rendering
- T014: Tracer core

#### Success Criteria
- [ ] Real-time hop updates show
- [ ] Progress indicator animates
- [ ] Completion is signaled
- [ ] Errors display correctly

---

### T080: Implement Continuous Tracing Mode

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 1 day

#### Description
Add continuous tracing mode that repeatedly traces to the target, similar to mtr.

#### Technical Details
```go
// internal/tui/continuous.go
type ContinuousModel struct {
    Model
    interval  time.Duration
    iteration int
    history   [][]HopView
    paused    bool
}

func (m ContinuousModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case TraceCompleteMsg:
        m.iteration++
        m.history = append(m.history, m.hops)
        
        // Keep last N iterations for statistics
        if len(m.history) > 100 {
            m.history = m.history[1:]
        }
        
        // Merge statistics
        m.hops = m.mergeHistory()
        
        if !m.paused {
            // Start next iteration after interval
            return m, tea.Tick(m.interval, func(time.Time) tea.Msg {
                return StartIterationMsg{}
            })
        }
        return m, nil
    
    case StartIterationMsg:
        return m, startTrace(m.config)
    }
    
    // ... parent update handling
}

func (m ContinuousModel) mergeHistory() []HopView {
    // Calculate statistics across all iterations
    merged := make([]HopView, len(m.hops))
    
    for hopIdx := range m.hops {
        var allRTTs []float64
        
        for _, iteration := range m.history {
            if hopIdx < len(iteration) {
                allRTTs = append(allRTTs, iteration[hopIdx].RTTs...)
            }
        }
        
        if len(allRTTs) > 0 {
            merged[hopIdx] = m.hops[hopIdx]
            merged[hopIdx].AvgRTT = average(allRTTs)
            merged[hopIdx].MinRTT = min(allRTTs)
            merged[hopIdx].MaxRTT = max(allRTTs)
            merged[hopIdx].Jitter = max(allRTTs) - min(allRTTs)
        }
    }
    
    return merged
}

func (m ContinuousModel) View() string {
    // Add iteration counter to header
    header := fmt.Sprintf(
        "Target: %s    Iterations: %d    Interval: %s",
        m.target,
        m.iteration,
        m.interval,
    )
    
    // Rest of view...
}
```

#### Files to Touch
- `internal/tui/continuous.go` (new)
- `internal/tui/model.go` (update)

#### Dependencies
- T079: Real-time updates

#### Success Criteria
- [ ] Continuous mode repeats trace
- [ ] Statistics aggregate over time
- [ ] Pause/resume works
- [ ] History is bounded

---

### T081: Implement Export from TUI

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Add ability to export trace results to file from within the TUI.

#### Technical Details
```go
// internal/tui/export.go
func exportResults(m Model) tea.Cmd {
    return func() tea.Msg {
        // Convert TUI model to TraceResult
        result := m.toTraceResult()
        
        // Generate filename
        timestamp := time.Now().Format("20060102-150405")
        filename := fmt.Sprintf("poros-%s-%s.json", m.target, timestamp)
        
        // Format as JSON
        formatter := output.NewJSONFormatter(output.FormatterConfig{}, true)
        data, err := formatter.Format(result)
        if err != nil {
            return ErrorMsg{Err: err}
        }
        
        // Write file
        if err := os.WriteFile(filename, data, 0644); err != nil {
            return ErrorMsg{Err: err}
        }
        
        return ExportCompleteMsg{Filename: filename}
    }
}

type ExportCompleteMsg struct {
    Filename string
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case ExportCompleteMsg:
        m.status = fmt.Sprintf("Exported to %s", msg.Filename)
        return m, nil
    // ...
    }
}
```

#### Files to Touch
- `internal/tui/export.go` (new)
- `internal/tui/update.go` (update)

#### Dependencies
- T049: JSON formatter

#### Success Criteria
- [ ] Export creates file
- [ ] Status shows filename
- [ ] JSON format is valid

---

### T082: Add TUI CLI Flag

**Status:** NOT_STARTED
**Priority:** P1
**Estimated Effort:** 0.5 days

#### Description
Add `--tui` flag to CLI and wire up TUI mode.

#### Technical Details
```go
// cmd/poros/root.go (update)
func init() {
    rootCmd.Flags().BoolP("tui", "t", false, "Run in TUI mode")
    rootCmd.Flags().Bool("continuous", false, 
        "Continuous tracing mode (TUI only)")
    rootCmd.Flags().Duration("interval", time.Second, 
        "Interval between traces in continuous mode")
}

func runTrace(cmd *cobra.Command, args []string) error {
    if getBool(cmd, "tui") {
        return runTUI(cmd, args)
    }
    
    // ... regular trace
}

func runTUI(cmd *cobra.Command, args []string) error {
    target := args[0]
    
    config := tui.Config{
        Target:      target,
        ProbeMethod: getProbeMethod(cmd),
        MaxHops:     getInt(cmd, "max-hops"),
        ProbeCount:  getInt(cmd, "queries"),
        Timeout:     getDuration(cmd, "timeout"),
        Continuous:  getBool(cmd, "continuous"),
        Interval:    getDuration(cmd, "interval"),
    }
    
    app := tui.New(config)
    return app.Run()
}
```

#### Files to Touch
- `cmd/poros/root.go` (update)
- `cmd/poros/tui.go` (new)

#### Dependencies
- T076-T080: TUI implementation

#### Success Criteria
- [ ] `--tui` launches TUI
- [ ] `--continuous` enables continuous mode
- [ ] All other flags work with TUI

---

### T083: Implement TUI Color Themes

**Status:** NOT_STARTED
**Priority:** P3
**Estimated Effort:** 0.5 days

#### Description
Add color theme support for TUI with light/dark mode detection.

#### Technical Details
```go
// internal/tui/themes.go
type Theme struct {
    Name       string
    Header     lipgloss.Style
    HopNormal  lipgloss.Style
    HopTimeout lipgloss.Style
    RTTLow     lipgloss.Style
    RTTMed     lipgloss.Style
    RTTHigh    lipgloss.Style
    Status     lipgloss.Style
    Help       lipgloss.Style
}

var DarkTheme = Theme{
    Name: "dark",
    Header: lipgloss.NewStyle().
        Foreground(lipgloss.Color("39")).
        Bold(true),
    RTTLow:  lipgloss.NewStyle().Foreground(lipgloss.Color("42")),
    RTTMed:  lipgloss.NewStyle().Foreground(lipgloss.Color("226")),
    RTTHigh: lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
}

var LightTheme = Theme{
    Name: "light",
    Header: lipgloss.NewStyle().
        Foreground(lipgloss.Color("27")).
        Bold(true),
    // ...
}

func DetectTheme() Theme {
    // Check COLORFGBG or terminal background
    if lipgloss.HasDarkBackground() {
        return DarkTheme
    }
    return LightTheme
}
```

#### Files to Touch
- `internal/tui/themes.go` (new)
- `internal/tui/styles.go` (update)

#### Dependencies
- T078: View rendering

#### Success Criteria
- [ ] Dark theme works
- [ ] Light theme works
- [ ] Auto-detection works

---

### T084: Add TUI Tests

**Status:** NOT_STARTED
**Priority:** P2
**Estimated Effort:** 0.5 days

#### Description
Create unit tests for TUI components.

#### Technical Details
```go
// internal/tui/model_test.go
func TestModel_Init(t *testing.T) {
    config := Config{
        Target:  "google.com",
        MaxHops: 30,
    }
    
    model := NewModel(config)
    
    assert.Equal(t, "google.com", model.target)
    assert.Len(t, model.hops, 30)
    assert.False(t, model.completed)
}

func TestModel_Update_HopUpdate(t *testing.T) {
    model := NewModel(Config{MaxHops: 30})
    
    msg := HopUpdateMsg{
        Hop: 1,
        IP:  net.ParseIP("192.168.1.1"),
        RTT: 5 * time.Millisecond,
    }
    
    newModel, _ := model.Update(msg)
    m := newModel.(Model)
    
    assert.Equal(t, "192.168.1.1", m.hops[0].IP)
    assert.Equal(t, HopComplete, m.hops[0].Status)
}

func TestModel_Update_KeyQuit(t *testing.T) {
    model := NewModel(Config{})
    
    _, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
    
    // Verify quit command
    assert.NotNil(t, cmd)
}

// internal/tui/view_test.go
func TestRenderRTTBar(t *testing.T) {
    model := NewModel(Config{})
    model.width = 80
    
    tests := []struct {
        rtt      float64
        contains string
    }{
        {0, "░░░░░░░░░░░░░░░░░░░░"},
        {50, "█████"},
        {200, "████████████████████"},
    }
    
    for _, tt := range tests {
        bar := model.renderRTTBar(tt.rtt)
        assert.Contains(t, bar, tt.contains)
    }
}
```

#### Files to Touch
- `internal/tui/model_test.go` (new)
- `internal/tui/view_test.go` (new)
- `internal/tui/update_test.go` (new)

#### Dependencies
- T076-T082: TUI implementation

#### Success Criteria
- [ ] Model tests pass
- [ ] Update tests pass
- [ ] View rendering tests pass

---

## Performance Targets
- TUI render: < 16ms (60fps)
- Memory usage: < 20MB additional
- Startup time: < 200ms

## Risk Assessment

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Terminal compatibility | Medium | Medium | Test on popular terminals |
| Performance on slow terminals | Low | Low | Throttle updates |
| Unicode rendering issues | Medium | Low | ASCII fallback option |

## Notes
- Consider adding mouse support later
- Sparklines for RTT history would be nice
- Could add hop details popup with full info
- Consider vim-like navigation keys
