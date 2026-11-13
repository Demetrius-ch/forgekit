package output

import (
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// Spinner is a minimal terminal spinner used only in human mode.
type Spinner struct {
	out           io.Writer
	frames        []rune
	delay         time.Duration
	mu            sync.Mutex
	stop          chan struct{}
	done          chan struct{}
	lastLineWidth int
	static        bool // output is not a terminal: no animation
	active        bool // static mode: Start called, Stop pending
}

// NewSpinner creates a spinner that writes to out.
func NewSpinner(out io.Writer) *Spinner {
	// Animation frames on a file that is not a terminal (pipe, CI log,
	// redirect) end up as garbage in the output: only print final lines.
	static := false
	if f, ok := out.(*os.File); ok && !term.IsTerminal(int(f.Fd())) {
		static = true
	}
	return &Spinner{
		out:    out,
		frames: []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
		delay:  80 * time.Millisecond,
		stop:   nil,
		static: static,
	}
}

// Start begins the spinner with a message. It is safe to call repeatedly.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.static {
		s.active = true
		return
	}
	if s.stop != nil {
		return
	}
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go func() {
		defer close(s.done)
		i := 0
		for {
			select {
			case <-s.stop:
				return
			default:
				frame := s.frames[i%len(s.frames)]
				i++
				line := string(frame) + " " + msg
				s.mu.Lock()
				s.lastLineWidth = len(line)
				io.WriteString(s.out, "\r")
				io.WriteString(s.out, line)
				s.mu.Unlock()
				time.Sleep(s.delay)
			}
		}
	}()
}

// Stop stops the spinner and prints a final line with the provided final message.
func (s *Spinner) Stop(final string) {
	if s.static {
		s.mu.Lock()
		wasActive := s.active
		s.active = false
		s.mu.Unlock()
		if wasActive {
			io.WriteString(s.out, final+"\n")
		}
		return
	}
	g := false
	s.mu.Lock()
	if s.stop != nil {
		close(s.stop)
		g = true
	}
	s.mu.Unlock()
	if !g {
		return
	}
	<-s.done

	s.mu.Lock()
	width := s.lastLineWidth
	if len(final) > width {
		width = len(final)
	}
	s.stop = nil
	s.done = nil
	s.mu.Unlock()

	s.mu.Lock()
	io.WriteString(s.out, "\r")
	io.WriteString(s.out, strings.Repeat(" ", width))
	io.WriteString(s.out, "\r")
	s.mu.Unlock()
	io.WriteString(s.out, final+"\n")
}
