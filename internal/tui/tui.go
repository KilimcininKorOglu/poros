package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/KilimcininKorOglu/poros/internal/trace"
)

// Run starts the TUI with the given target and configuration.
func Run(target string, config *trace.Config) error {
	model, err := New(target, config)
	if err != nil {
		return fmt.Errorf("failed to create TUI model: %w", err)
	}
	defer model.Close()

	p := tea.NewProgram(model, tea.WithAltScreen())
	
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if there was an error during the trace
	if m, ok := finalModel.(Model); ok {
		if m.state == StateError && m.err != nil {
			return m.err
		}
	}

	return nil
}
