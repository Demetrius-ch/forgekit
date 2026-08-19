package errs

import (
	"errors"
	"fmt"

	"github.com/Demetrius-ch/forgekit/internal/cli"
)

// Code identifies error categories for exit status mapping.
type Code int

const (
	CodeGeneric Code = iota + 1
	CodeUsage
	CodeValidation
	CodeNotFound
	CodeConflict
	CodeInternal
)

// Error is a user-facing ForgeKit error with optional internal cause.
type Error struct {
	Code    Code
	Message string
	Hint    string
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.cause)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.cause }

// UserMessage returns a message safe to print to CLI users.
func (e *Error) UserMessage() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s\n  → %s", e.Message, e.Hint)
	}
	return e.Message
}

// New creates a user-facing error.
func New(code Code, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Wrap wraps an internal cause with a user-facing message.
func Wrap(code Code, message string, cause error) *Error {
	return &Error{Code: code, Message: message, cause: cause}
}

// AsForge returns a ForgeKit error if present.
func AsForge(err error) (*Error, bool) {
	var fe *Error
	if errors.As(err, &fe) {
		return fe, true
	}
	return nil, false
}

// ExitCode maps an error to a CLI exit code.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if fe, ok := AsForge(err); ok {
		switch fe.Code {
		case CodeUsage:
			return 2
		case CodeValidation, CodeConflict, CodeNotFound:
			return 1
		case CodeInternal:
			return 1
		default:
			return 1
		}
	}
	// Check for CI mode error with explicit exit code
	var ciErr *cli.CiError
	if errors.As(err, &ciErr) {
		return ciErr.Code
	}
	return 1
}
