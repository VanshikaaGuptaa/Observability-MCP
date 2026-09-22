package errors

import (
	"fmt"
)

type ErrorCategory string

const (
	CategoryValidation         ErrorCategory = "validation_error"
	CategoryBackendUnavailable ErrorCategory = "backend_unavailable"
	CategoryQuery              ErrorCategory = "query_error"
	CategoryTimeout            ErrorCategory = "timeout"
	CategoryNoData             ErrorCategory = "no_data"
	CategoryInternal           ErrorCategory = "internal_error"
)

type MCPError struct {
	Category ErrorCategory `json:"category"`
	Message  string        `json:"message"`
	Err      error         `json:"-"`
}

func (e *MCPError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Category, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Category, e.Message)
}

func (e *MCPError) Unwrap() error {
	return e.Err
}

func NewValidationError(msg string) *MCPError {
	return &MCPError{Category: CategoryValidation, Message: msg}
}

func NewBackendUnavailableError(backend string, err error) *MCPError {
	return &MCPError{
		Category: CategoryBackendUnavailable,
		Message:  fmt.Sprintf("%s backend unavailable", backend),
		Err:      err,
	}
}

func NewQueryError(query string, err error) *MCPError {
	return &MCPError{
		Category: CategoryQuery,
		Message:  fmt.Sprintf("query error: %s", query),
		Err:      err,
	}
}

func NewTimeoutError(msg string, err error) *MCPError {
	return &MCPError{
		Category: CategoryTimeout,
		Message:  msg,
		Err:      err,
	}
}

func NewNoDataError(msg string) *MCPError {
	return &MCPError{
		Category: CategoryNoData,
		Message:  msg,
	}
}

func NewInternalError(msg string, err error) *MCPError {
	return &MCPError{
		Category: CategoryInternal,
		Message:  msg,
		Err:      err,
	}
}
