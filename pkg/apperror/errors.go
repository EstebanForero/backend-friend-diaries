package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code int, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

func Wrap(code int, message string, err error) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

func NotFound(resource string) *AppError {
	return New(http.StatusNotFound, fmt.Sprintf("%s not found", resource))
}

func Unauthorized(message string) *AppError {
	return New(http.StatusUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return New(http.StatusForbidden, message)
}

func BadRequest(message string) *AppError {
	return New(http.StatusBadRequest, message)
}

func Conflict(message string) *AppError {
	return New(http.StatusConflict, message)
}

func Internal(message string, err error) *AppError {
	return Wrap(http.StatusInternalServerError, message, err)
}

func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
