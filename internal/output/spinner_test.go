package output

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func TestSpinnerStopsCleanly(t *testing.T) {
	var buf bytes.Buffer
	w := &lockedWriter{w: &buf}
	s := NewSpinner(w)
	s.delay = 5 * time.Millisecond

	s.Start("Analyse Docker...")
	// Give the spinner time to render at least one frame.
	time.Sleep(40 * time.Millisecond)
	s.Stop("✓ Docker — 100/100")

	out := buf.String()
	if !strings.Contains(out, "✓ Docker — 100/100\n") {
		t.Fatalf("expected final message in output, got %q", out)
	}

	// Interpret the last rendered terminal line after carriage returns.
	rendered := out
	if strings.HasSuffix(rendered, "\n") {
		rendered = rendered[:len(rendered)-1]
	}
	if idx := strings.LastIndex(rendered, "\r"); idx != -1 {
		rendered = rendered[idx+1:]
	}
	if strings.TrimSpace(rendered) != "✓ Docker — 100/100" {
		t.Fatalf("expected clean final line after carriage returns, got %q", rendered)
	}

	// Ensure no spinner glyphs appear after the final rendered line.
	lastIndex := strings.LastIndex(out, "✓ Docker — 100/100")
	if lastIndex == -1 {
		t.Fatal("final message not found in output")
	}
	remaining := out[lastIndex+len("✓ Docker — 100/100"):]
	if strings.ContainsAny(remaining, "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏") {
		t.Fatalf("spinner glyphs found after final message: %q", remaining)
	}
}
