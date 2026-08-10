package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Demetrius-ch/forgekit/internal/engine"
	"github.com/Demetrius-ch/forgekit/internal/report"
)

// Format selects human or JSON rendering.
type Format string

const (
	FormatHuman Format = "human"
	FormatJSON  Format = "json"
)

// Console wraps stdout/stderr with CI-aware styling.
type Console struct {
	Out    io.Writer
	Err    io.Writer
	Color  bool
	Quiet  bool
	Format Format
}

// NewConsole creates a console respecting NO_COLOR and CI env vars.
func NewConsole(out, err io.Writer, format Format, quiet bool) *Console {
	color := os.Getenv("NO_COLOR") == "" && os.Getenv("CI") == "" && os.Getenv("FORGE_NO_COLOR") == ""
	return &Console{Out: out, Err: err, Color: color, Quiet: quiet, Format: format}
}

func (c *Console) Printf(format string, args ...any) {
	if c.Quiet {
		return
	}
	fmt.Fprintf(c.Out, format, args...)
}

func (c *Console) PrintResult(res report.Result) error {
	if c.Format == FormatJSON {
		enc := json.NewEncoder(c.Out)
		enc.SetIndent("", "  ")
		return enc.Encode(res)
	}
	return c.printHuman(res)
}

func (c *Console) printHuman(res report.Result) error {
	// Special pretty rendering for analyze command.
	if res.Command == "analyze" {
		// Header
		fmt.Fprintln(c.Out, "ForgeKit Analyze")
		fmt.Fprintln(c.Out, strings.Repeat("─", 40))
		fmt.Fprintln(c.Out)

		// Short list of findings summary
		fmt.Fprintln(c.Out, "Résumé des vérifications :")
		for _, f := range res.Findings {
			marker := "✓"
			if f.Severity == report.SeverityWarning {
				marker = "⚠"
			} else if f.Severity == report.SeverityError || f.Severity == report.SeverityCritical {
				marker = "✗"
			}
			fmt.Fprintf(c.Out, "%s %s\n", marker, f.Message)
		}

		// Compute scores and render bars
		score := report.ComputeScore(res)

		// Project/module info
		module := ""
		if res.Project != "" {
			if data, err := os.ReadFile(filepath.Join(res.Project, "go.mod")); err == nil {
				lines := strings.Split(string(data), "\n")
				if len(lines) > 0 {
					for _, ln := range lines {
						ln = strings.TrimSpace(ln)
						if strings.HasPrefix(ln, "module ") {
							module = strings.TrimSpace(strings.TrimPrefix(ln, "module"))
							break
						}
					}
				}
			}
		}
		fmt.Fprintf(c.Out, "\nProjet : %s\n", res.Project)
		if module != "" {
			fmt.Fprintf(c.Out, "Module  : %s\n", module)
		}

		// render each category
		bar := func(n int) string {
			total := 40
			filled := (n * total) / 100
			if filled > total {
				filled = total
			}
			if filled < 0 {
				filled = 0
			}
			return strings.Repeat("█", filled) + strings.Repeat("░", total-filled)
		}

		fmt.Fprintln(c.Out)
		for _, cat := range []string{"Architecture", "Tests", "Security", "Configuration", "Docker", "Documentation"} {
			val := score.Categories[cat]
			barStr := bar(val)
			if c.Color {
				// color by thresholds
				if val >= 80 {
					barStr = "\033[32m" + barStr + "\033[0m"
				} else if val >= 50 {
					barStr = "\033[33m" + barStr + "\033[0m"
				} else {
					barStr = "\033[31m" + barStr + "\033[0m"
				}
				cat = "\033[36m" + cat + "\033[0m"
			}
			fmt.Fprintf(c.Out, "%s\n%s %3d/100\n\n", cat, barStr, val)
		}

		// Global score and recommendations
		fmt.Fprintln(c.Out, strings.Repeat("─", 40))
		fmt.Fprintf(c.Out, "Score global : %d/100\n\n", score.Global)
		if len(score.Notes) > 0 {
			fmt.Fprintln(c.Out, "Recommandations :")
			for _, n := range score.Notes {
				fmt.Fprintf(c.Out, "⚠ %s\n", n)
			}
			fmt.Fprintln(c.Out)
		}

		// duration since Timestamp (Timestamp set at analysis start)
		if !c.Quiet {
			if !res.Timestamp.IsZero() {
				dur := time.Since(res.Timestamp)
				if dur < time.Second {
					fmt.Fprintf(c.Out, "Analyse terminée en %dms\n", dur.Milliseconds())
				} else {
					fmt.Fprintf(c.Out, "Analyse terminée en %.2fs\n", dur.Seconds())
				}
			}
		}

		return nil
	}

	// default behavior for other commands
	for _, f := range res.Findings {
		label := f.Severity.LegacySeverity()
		if f.Category == "pass" {
			label = report.PassLabel()
		}
		prefix := label
		if c.Color {
			prefix = colorize(label, f.Severity)
		}
		loc := ""
		if f.File != "" {
			loc = fmt.Sprintf(" (%s", f.File)
			if f.Line > 0 {
				loc += fmt.Sprintf(":%d", f.Line)
			}
			loc += ")"
		}
		fmt.Fprintf(c.Out, "%-8s %s%s\n", prefix, f.Message, loc)
		if f.Suggestion != "" {
			fmt.Fprintf(c.Out, "         → %s\n", f.Suggestion)
		}
	}
	if !c.Quiet {
		fmt.Fprintf(c.Out, "\nRésumé : %d pass, %d info, %d warning, %d error, %d critical\n",
			res.Summary.Pass, res.Summary.Info, res.Summary.Warning, res.Summary.Error, res.Summary.Critical)
	}
	return nil
}

func (c *Console) PrintPlan(plan []engine.PlanEntry) {
	if c.Format == FormatJSON {
		enc := json.NewEncoder(c.Out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"dry_run": true, "entries": plan})
		return
	}
	fmt.Fprintln(c.Out, "Dry-run — aucun fichier ne sera écrit :")
	fmt.Fprintln(c.Out)
	for _, e := range plan {
		switch e.Action {
		case engine.ActionCreate:
			fmt.Fprintf(c.Out, "  CREATE %s (%v)\n", e.Path, e.Mode)
		case engine.ActionSkip:
			fmt.Fprintf(c.Out, "  SKIP   %s", e.Path)
			if e.Reason != "" {
				fmt.Fprintf(c.Out, " — %s", e.Reason)
			}
			fmt.Fprintln(c.Out)
		}
	}
	fmt.Fprintf(c.Out, "\n%d fichier(s) seraient créés\n", countCreates(plan))
}

func countCreates(plan []engine.PlanEntry) int {
	n := 0
	for _, e := range plan {
		if e.Action == engine.ActionCreate {
			n++
		}
	}
	return n
}

func colorize(label string, sev report.Severity) string {
	switch sev {
	case report.SeverityError, report.SeverityCritical:
		return "\033[31m" + label + "\033[0m"
	case report.SeverityWarning:
		return "\033[33m" + label + "\033[0m"
	case report.SeverityInfo:
		if label == report.PassLabel() {
			return "\033[32m" + label + "\033[0m"
		}
		return "\033[36m" + label + "\033[0m"
	default:
		return label
	}
}

// Debug prints internal error details when --debug is set.
func Debug(w io.Writer, debug bool, err error) {
	if !debug || err == nil {
		return
	}
	fmt.Fprintf(w, "\n[debug] %v\n", err)
	if unw := unwrapChain(err); len(unw) > 1 {
		fmt.Fprintln(w, "[debug] cause chain:")
		for _, e := range unw {
			fmt.Fprintf(w, "  - %v\n", e)
		}
	}
}

func unwrapChain(err error) []error {
	var chain []error
	for err != nil {
		chain = append(chain, err)
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			break
		}
		err = u.Unwrap()
	}
	return chain
}

var _ = strings.Builder{}
