package client

import (
	"fmt"
	"strings"
)

// ValidationErrorDetail represents a single field-level validation error from the API.
type ValidationErrorDetail struct {
	Loc  []interface{} `json:"loc"`
	Msg  string        `json:"msg"`
	Type string        `json:"type"`
}

// APIError represents an error response from the Devin API.
type APIError struct {
	StatusCode int
	Message    string
	Detail     []ValidationErrorDetail `json:"detail"`
}

func (e *APIError) Error() string {
	if len(e.Detail) > 0 {
		parts := make([]string, len(e.Detail))
		for i, d := range e.Detail {
			parts[i] = fmt.Sprintf("%s: %s", d.Type, d.Msg)
		}
		return fmt.Sprintf("API error %d: %s", e.StatusCode, strings.Join(parts, "; "))
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found response.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}
