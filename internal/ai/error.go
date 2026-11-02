package ai

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/helightxu/xinkgo/internal/locale"
)

// The ways an answer can go wrong. Each of them is a sentence a person can
// act on: what the HTTP library said is not one of them.
//
// The sentence is not written down with the error, for a language is
// chosen while the app stands: what the error says is looked up when it
// is said (Error), and so follows the language the app shows.
var (
	ErrNotConfigured = Failure{notConfigured}
	ErrNoModel       = Failure{noModel}
	ErrAuth          = Failure{notAuthorized}
	ErrRateLimit     = Failure{heldBack}
	ErrTimeout       = Failure{tookTooLong}
	// ErrTruncated is a stream that went dead before it said it was
	// over. It is not the same as a model that ran out of budget: that
	// one came to an end of its own, this one did not.
	ErrTruncated   = Failure{brokeOff}
	ErrEmptyResult = Failure{noWords}
	ErrCanceled    = Failure{givenUpOn}
	// ErrSteps is a model that kept reaching for tools and never came
	// to an answer.
	ErrSteps = Failure{tooManySteps}
	// ErrSilent is a model that was asked and said nothing at all, not
	// even a first word, before it was given up on.
	ErrSilent = Failure{silent}
)

// Failure is one of the ways an answer can go wrong. It is only which one:
// what is said of it is looked up when it is said, so that it follows the
// language the app shows.
type Failure struct{ kind wentWrong }

// The ways an answer can go wrong, as a Failure names them.
type wentWrong int

const (
	notConfigured wentWrong = iota
	noModel
	notAuthorized
	heldBack
	tookTooLong
	brokeOff
	noWords
	givenUpOn
	tooManySteps
	silent
)

// Error is what went wrong, in the language the app shows now.
func (f Failure) Error() string {
	switch f.kind {
	case notConfigured:
		return locale.AiErrNotConfigured()
	case noModel:
		return locale.AiErrNoModel()
	case notAuthorized:
		return locale.AiErrAuth()
	case heldBack:
		return locale.AiErrRateLimit()
	case tookTooLong:
		return locale.AiErrTimeout()
	case brokeOff:
		return locale.AiErrTruncated()
	case noWords:
		return locale.AiErrEmptyResult()
	case givenUpOn:
		return locale.AiErrCanceled()
	case tooManySteps:
		return locale.AiErrSteps()
	default:
		return locale.AiErrSilent()
	}
}

// Network is a model that could not be reached at all: the sentence says
// what a person can do about it.
func Network(detail string) error {
	return unreachable{detail: detail}
}

// Protocol is a model that answered something that could not be read.
func Protocol(detail string) error {
	return sentence{locale.AiErrProtocol(detail)}
}

// NoSuchModel is a model the server has never heard of.
func NoSuchModel(name string) error {
	return sentence{locale.AiErrNoSuchModel(name)}
}

// unreachable is a model that could not be reached.
type unreachable struct{ detail string }

// Error says what happened, and what to do about it, in the language the
// app shows now.
func (e unreachable) Error() string {
	return locale.AiErrUnreachable(e.detail)
}

// sentence is one thing that went wrong, named in the language the app
// shows: what is said of it is worked out when it is made, which is when
// it is shown.
type sentence struct{ said string }

func (f sentence) Error() string { return f.said }

// Status is an answer that came back with a status that is not a good one.
type Status struct {
	Code int
	Body string
}

// Error says the status, and whatever came with it.
func (s Status) Error() string {
	return fmt.Sprintf("the model answered %d: %s", s.Code, s.Body)
}

// failed is what a status means to the one who was waiting for words.
func (s Status) failed(model string) error {
	switch {
	case s.Code == http.StatusUnauthorized, s.Code == http.StatusForbidden:
		return ErrAuth
	case s.Code == http.StatusTooManyRequests:
		return ErrRateLimit
	case s.Code == http.StatusNotFound:
		return NoSuchModel(model)
	}
	return Protocol(s.Error())
}

// Retryable is whether asking again is worth the words. A key that is
// wrong, or a model that does not exist, comes back the same however
// often it is asked; a model that was busy, or slow, or that broke off
// mid-sentence, may well answer whole the second time.
func Retryable(err error) bool {
	if errors.Is(err, ErrRateLimit) || errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrTruncated) {
		return true
	}
	var never unreachable
	return errors.As(err, &never)
}

// failure is what went wrong, in words a person can act on, told which
// model it was that failed so the sentence can name it.
func failure(err error, model string) error {
	switch {
	case errors.Is(err, context.Canceled):
		return ErrCanceled
	case errors.Is(err, ErrSilent), errors.Is(err, context.DeadlineExceeded):
		return ErrTimeout
	case errors.Is(err, ErrTruncated):
		return ErrTruncated
	}
	var timed interface{ Timeout() bool }
	if errors.As(err, &timed) && timed.Timeout() {
		return ErrTimeout
	}
	var status Status
	if errors.As(err, &status) {
		return status.failed(model)
	}
	return Network(err.Error())
}
