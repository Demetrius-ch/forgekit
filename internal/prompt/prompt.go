package prompt

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"
)

// Prompter asks simple questions on stdin/stdout.
type Prompter struct {
	in  *bufio.Reader
	out io.Writer
}

func New(in io.Reader, out io.Writer) *Prompter {
	return &Prompter{in: bufio.NewReader(in), out: out}
}

func (p *Prompter) AskString(prompt, defaultValue string) (string, error) {
	fmt.Fprintf(p.out, "%s [%s]: ", prompt, defaultValue)
	text, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue, nil
	}
	return text, nil
}

func (p *Prompter) AskInt(prompt string, defaultValue int) (int, error) {
	fmt.Fprintf(p.out, "%s [%d]: ", prompt, defaultValue)
	text, err := p.in.ReadString('\n')
	if err != nil && err != io.EOF {
		return 0, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("valeur invalide pour %s: %w", prompt, err)
	}
	return value, nil
}

func (p *Prompter) AskPassword(prompt string) (string, error) {
	fmt.Fprintf(p.out, "%s: ", prompt)
	// Read password without echoing
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", err
	}
	fmt.Fprintln(p.out) // New line after password input
	return string(bytePassword), nil
}
