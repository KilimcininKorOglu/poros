package output

import (
	"io"
	"os"

	"github.com/KilimcininKorOglu/poros/internal/trace"
	"github.com/mattn/go-isatty"
)

// Writer handles output formatting and writing.
type Writer struct {
	formatter Formatter
	output    io.Writer
	isTTY     bool
}

// NewWriter creates a new output writer.
func NewWriter(format Format, config Config) *Writer {
	// Auto-detect TTY and disable colors if not a terminal
	isTTY := isTerminal(os.Stdout)
	if !isTTY {
		config.Colors = false
	}

	return &Writer{
		formatter: NewFormatter(format, config),
		output:    os.Stdout,
		isTTY:     isTTY,
	}
}

// NewWriterWithFormatter creates a writer with a specific formatter.
func NewWriterWithFormatter(formatter Formatter, output io.Writer) *Writer {
	isTTY := false
	if f, ok := output.(*os.File); ok {
		isTTY = isTerminal(f)
	}

	return &Writer{
		formatter: formatter,
		output:    output,
		isTTY:     isTTY,
	}
}

// Write formats and writes the trace result.
func (w *Writer) Write(result *trace.TraceResult) error {
	data, err := w.formatter.Format(result)
	if err != nil {
		return err
	}

	_, err = w.output.Write(data)
	if err != nil {
		return err
	}

	// Flush output if it's a file (ensures output is visible immediately)
	if f, ok := w.output.(*os.File); ok {
		f.Sync()
	}

	return nil
}

// SetOutput changes the output destination.
func (w *Writer) SetOutput(output io.Writer) {
	w.output = output
	if f, ok := output.(*os.File); ok {
		w.isTTY = isTerminal(f)
	} else {
		w.isTTY = false
	}
}

// IsTTY returns whether the output is a terminal.
func (w *Writer) IsTTY() bool {
	return w.isTTY
}

// Formatter returns the underlying formatter.
func (w *Writer) Formatter() Formatter {
	return w.formatter
}

// isTerminal checks if the given file is a terminal.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// WriteToFile writes the trace result to a file.
func WriteToFile(result *trace.TraceResult, filename string, formatter Formatter) error {
	data, err := formatter.Format(result)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0644)
}
