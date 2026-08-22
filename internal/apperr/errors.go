package apperr

import (
	"errors"
	"net/http"
)

// Code identifies the category of an application error.
type Code string

const (
	CodeValidation   Code = "VALIDATION"
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeForbidden    Code = "FORBIDDEN"
	CodeInternal     Code = "INTERNAL"
	CodeRateLimited  Code = "RATE_LIMITED"
	CodeUnavailable  Code = "UNAVAILABLE"
)

// AppError is a structured application error that carries an HTTP status,
// an error code, and a safe-to-display message.
type AppError struct {
	Status  int    `json:"-"`
	Code    Code   `json:"code"`
	Message string `json:"error"`
	Cause   error  `json:"-"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying cause.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// --- Constructors ---

func New(status int, code Code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg}
}

func Wrap(err error, status int, code Code, msg string) *AppError {
	return &AppError{Status: status, Code: code, Message: msg, Cause: err}
}

func NotFound(msg string) *AppError {
	return New(http.StatusNotFound, CodeNotFound, msg)
}

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, CodeValidation, msg)
}

func Conflict(msg string) *AppError {
	return New(http.StatusConflict, CodeConflict, msg)
}

func Unauthorized(msg string) *AppError {
	return New(http.StatusUnauthorized, CodeUnauthorized, msg)
}

func Forbidden(msg string) *AppError {
	return New(http.StatusForbidden, CodeForbidden, msg)
}

func Internal(msg string) *AppError {
	return New(http.StatusInternalServerError, CodeInternal, msg)
}

// FromError attempts to cast an error to *AppError. If it is not one,
// it returns an Internal error wrapping the original.
func FromError(err error) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Wrap(err, http.StatusInternalServerError, CodeInternal, "internal server error")
}
