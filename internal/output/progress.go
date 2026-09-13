package output

import (
	"fmt"
	"io"
	"os"
	"time"
)

// Progress provides a consistent step-by-step display for CLI operations.
type Progress interface {
	// Title displays the operation header.
	Title(name string)
	// Step displays a completed step with optional duration.
	Step(name string, duration time.Duration)
	// StepStart begins a step with a spinner (for long operations).
	StepStart(name string)
	// StepDone completes a step that was started with StepStart.
	StepDone(duration time.Duration)
	// Warning displays a warning message.
	Warning(msg string)
	// Info displays an informational message.
	Info(msg string)
	// Error displays an error message.
	Error(msg string)
	// Finish displays the final success message with total duration.
	Finish(name string, duration time.Duration)
	// FinishError displays a final error message.
	FinishError(name string)
}

// humanProgress renders steps with colors and timing for terminal output.
type humanProgress struct {
	out     io.Writer
	color   bool
	quiet   bool
	started bool
}

// newHumanProgress creates a human-readable progress display.
func newHumanProgress(out io.Writer, color, quiet bool) *humanProgress {
	return &humanProgress{out: out, color: color, quiet: quiet}
}

func (p *humanProgress) Title(name string) {
	if p.quiet {
		return
	}
	if p.color {
		fmt.Fprintf(p.out, "\033[36m◆ %s\033[0m\n", name)
	} else {
		fmt.Fprintf(p.out, "◆ %s\n", name)
	}
	fmt.Fprintln(p.out)
	p.started = true
}

func (p *humanProgress) Step(name string, duration time.Duration) {
	if p.quiet {
		return
	}
	prefix := p.successSymbol()
	if duration > 0 {
		fmt.Fprintf(p.out, "  %s %-38s %s\n", prefix, name, formatDuration(duration))
	} else {
		fmt.Fprintf(p.out, "  %s %s\n", prefix, name)
	}
}

func (p *humanProgress) StepStart(name string) {
	// Handled by spinner in commands.go; this is a no-op for human progress.
}

func (p *humanProgress) StepDone(duration time.Duration) {
	// Handled by spinner in commands.go; this is a no-op for human progress.
}

func (p *humanProgress) Warning(msg string) {
	if p.quiet {
		return
	}
	if p.color {
		fmt.Fprintf(p.out, "  \033[33m⚠ %s\033[0m\n", msg)
	} else {
		fmt.Fprintf(p.out, "  ⚠ %s\n", msg)
	}
}

func (p *humanProgress) Info(msg string) {
	if p.quiet {
		return
	}
	if p.color {
		fmt.Fprintf(p.out, "  \033[36mℹ %s\033[0m\n", msg)
	} else {
		fmt.Fprintf(p.out, "  ℹ %s\n", msg)
	}
}

func (p *humanProgress) Error(msg string) {
	if p.color {
		fmt.Fprintf(p.out, "  \033[31m✗ %s\033[0m\n", msg)
	} else {
		fmt.Fprintf(p.out, "  ✗ %s\n", msg)
	}
}

func (p *humanProgress) Finish(name string, duration time.Duration) {
	if p.quiet {
		return
	}
	fmt.Fprintln(p.out)
	symbol := p.successSymbol()
	if p.color {
		fmt.Fprintf(p.out, "\033[32m%s %s créé avec succès\033[0m", symbol, name)
	} else {
		fmt.Fprintf(p.out, "%s %s créé avec succès", symbol, name)
	}
	if duration > 0 {
		fmt.Fprintf(p.out, " en %s", formatDuration(duration))
	}
	fmt.Fprintln(p.out)
}

func (p *humanProgress) FinishError(name string) {
	if p.color {
		fmt.Fprintf(p.out, "\033[31m✗ Échec de la création de %q\033[0m\n", name)
	} else {
		fmt.Fprintf(p.out, "✗ Échec de la création de %q\n", name)
	}
}

func (p *humanProgress) successSymbol() string {
	if p.color {
		return "\033[32m✓\033[0m"
	}
	return "✓"
}

// jsonProgress renders structured JSON events for machine consumption.
type jsonProgress struct {
	out      io.Writer
	events   []progressEvent
	start    time.Time
}

type progressEvent struct {
	Name       string  `json:"name"`
	DurationMs int64   `json:"duration_ms,omitempty"`
	Status     string  `json:"status"`
	Message    string  `json:"message,omitempty"`
}

func newJSONProgress(out io.Writer) *jsonProgress {
	return &jsonProgress{out: out, start: time.Now()}
}

func (p *jsonProgress) Title(name string) {
	p.events = append(p.events, progressEvent{Name: name, Status: "start"})
}

func (p *jsonProgress) Step(name string, duration time.Duration) {
	evt := progressEvent{Name: name, Status: "success"}
	if duration > 0 {
		evt.DurationMs = duration.Milliseconds()
	}
	p.events = append(p.events, evt)
}

func (p *jsonProgress) StepStart(name string) {
	p.events = append(p.events, progressEvent{Name: name, Status: "start"})
}

func (p *jsonProgress) StepDone(duration time.Duration) {
	if len(p.events) > 0 {
		p.events[len(p.events)-1].Status = "success"
		p.events[len(p.events)-1].DurationMs = duration.Milliseconds()
	}
}

func (p *jsonProgress) Warning(msg string) {
	p.events = append(p.events, progressEvent{Name: "warning", Status: "warning", Message: msg})
}

func (p *jsonProgress) Info(msg string) {
	p.events = append(p.events, progressEvent{Name: "info", Status: "info", Message: msg})
}

func (p *jsonProgress) Error(msg string) {
	p.events = append(p.events, progressEvent{Name: "error", Status: "error", Message: msg})
}

func (p *jsonProgress) Finish(name string, duration time.Duration) {
	p.events = append(p.events, progressEvent{Name: "finish", Status: "success", DurationMs: duration.Milliseconds()})
}

func (p *jsonProgress) FinishError(name string) {
	p.events = append(p.events, progressEvent{Name: "finish", Status: "error", Message: name})
}

// silentProgress suppresses all output (quiet mode).
type silentProgress struct{}

func (silentProgress) Title(string)                      {}
func (silentProgress) Step(string, time.Duration)        {}
func (silentProgress) StepStart(string)                  {}
func (silentProgress) StepDone(time.Duration)            {}
func (silentProgress) Warning(string)                    {}
func (silentProgress) Info(string)                       {}
func (silentProgress) Error(string)                     {}
func (silentProgress) Finish(string, time.Duration)      {}
func (silentProgress) FinishError(string)                {}

// NewProgress creates the appropriate Progress implementation based on format and quiet settings.
func NewProgress(out io.Writer, format Format, quiet bool) Progress {
	if format == FormatJSON {
		return newJSONProgress(out)
	}
	if quiet {
		return silentProgress{}
	}
	color := os.Getenv("NO_COLOR") == "" && os.Getenv("CI") == "" && os.Getenv("FORGE_NO_COLOR") == ""
	return newHumanProgress(out, color, quiet)
}

// FormatDuration formats a duration for human-readable display.
// Examples: "2 ms", "47 ms", "1.24 s"
func FormatDuration(d time.Duration) string {
	return formatDuration(d)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		ms := d.Milliseconds()
		if ms == 0 {
			return "<1 ms"
		}
		return fmt.Sprintf("%d ms", ms)
	}
	s := d.Seconds()
	if s < 10 {
		return fmt.Sprintf("%.2f s", s)
	}
	return fmt.Sprintf("%.1f s", s)
}
