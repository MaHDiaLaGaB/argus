package llm

import (
	"context"
	"errors"
	"fmt"
)

// ErrorKind classifies a failed call so wrappers can decide, in code, whether
// to retry, fall back to another model (E02.07) or stop.
type ErrorKind string

// Error kinds. Retryable kinds may be retried or sent to a fallback model;
// the others fail the step.
const (
	ErrRateLimited    ErrorKind = "rate_limited"    // HTTP 429
	ErrOverloaded     ErrorKind = "overloaded"      // provider says it is overloaded (for example 529)
	ErrServer         ErrorKind = "server"          // HTTP 5xx
	ErrTimeout        ErrorKind = "timeout"         // no reply in time
	ErrInvalidRequest ErrorKind = "invalid_request" // our request is wrong (Validate or HTTP 400)
	ErrAuth           ErrorKind = "auth"            // HTTP 401/403: key missing, wrong or not allowed
	ErrBudget         ErrorKind = "budget"          // a budget would be exceeded (E02.06); never retried
	ErrRefused        ErrorKind = "refused"         // the provider declined the content
)

// Error is a failed model call.
type Error struct {
	Provider string // "anthropic", "openai", "xai", "cursor"
	Model    string // model ref
	Kind     ErrorKind
	Status   int // HTTP status, 0 if none
	Err      error
}

func (e *Error) Error() string {
	s := fmt.Sprintf("llm %s", e.Kind)
	if e.Model != "" {
		s += " (" + e.Model + ")"
	}
	if e.Status != 0 {
		s += fmt.Sprintf(" status %d", e.Status)
	}
	if e.Err != nil {
		s += ": " + e.Err.Error()
	}
	return s
}

func (e *Error) Unwrap() error { return e.Err }

// Retryable reports whether the same request may succeed if sent again or to a fallback.
func (e *Error) Retryable() bool {
	switch e.Kind {
	case ErrRateLimited, ErrOverloaded, ErrServer, ErrTimeout:
		return true
	}
	return false
}

// IsRetryable reports whether err, anywhere in its chain, is a retryable call
// failure. A context deadline is retryable; cancellation is not.
func IsRetryable(err error) bool {
	if e, ok := errors.AsType[*Error](err); ok {
		return e.Retryable()
	}
	return errors.Is(err, context.DeadlineExceeded)
}
