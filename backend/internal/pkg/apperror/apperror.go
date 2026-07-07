package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Kind classifies an application error so handlers can map it to an
// HTTP status without inspecting error strings.
type Kind int

const (
	KindInternal Kind = iota
	KindNotFound
	KindValidation
	KindUnauthorized
	KindForbidden
	KindConflict
	KindRateLimited
)

// Error is the application-level error carried from services to handlers.
type Error struct {
	Kind    Kind
	Message string   // user-safe message
	Errors  []string // field-level details, user-safe
	Err     error    // wrapped cause, logged server-side only
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// HTTPStatus maps the error kind to an HTTP status code.
func (e *Error) HTTPStatus() int {
	switch e.Kind {
	case KindNotFound:
		return http.StatusNotFound
	case KindValidation:
		return http.StatusUnprocessableEntity
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindConflict:
		return http.StatusConflict
	case KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}

func New(kind Kind, message string) *Error {
	return &Error{Kind: kind, Message: message}
}

func Wrap(kind Kind, message string, err error) *Error {
	return &Error{Kind: kind, Message: message, Err: err}
}

func NotFound(resource string) *Error {
	return &Error{Kind: KindNotFound, Message: resource + " not found"}
}

func Validation(message string, details ...string) *Error {
	return &Error{Kind: KindValidation, Message: message, Errors: details}
}

func Unauthorized(message string) *Error {
	return &Error{Kind: KindUnauthorized, Message: message}
}

func Forbidden(message string) *Error {
	return &Error{Kind: KindForbidden, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Kind: KindConflict, Message: message}
}

func Internal(err error) *Error {
	return &Error{Kind: KindInternal, Message: "something went wrong", Err: err}
}

// From extracts an *Error from err, or wraps it as internal.
func From(err error) *Error {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
