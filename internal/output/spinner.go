package output

import (
	"io"
	"strings"
	"sync"
	"time"
)

// Spinner is a minimal, dependency-free terminal spinner used only in human mode.
type Spinner struct {
	out           io.Writer
	frames        []rune
	delay         time.Duration
	mu            sync.Mutex
	stop          chan struct{}
	done          chan struct{}
	lastLineWidth int
}

// NewSpinner creates a spinner that writes to out.
func NewSpinner(out io.Writer) *Spinner {
	return &Spinner{
		out:    out,
		frames: []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
		delay:  80 * time.Millisecond,
		stop:   nil,
	}
}

// Start begins the spinner with a message. It is safe to call repeatedly.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
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
