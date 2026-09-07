package utils

import "fmt"

type HTTPStatusError struct {
	StatusCode int
	Status     string
	Body       string
}

func (err *HTTPStatusError) Error() string {
	if err.Status != "" {
		return fmt.Sprintf("upstream HTTP %d %s: %s", err.StatusCode, err.Status, err.Body)
	}
	return fmt.Sprintf("upstream HTTP %d: %s", err.StatusCode, err.Body)
}

func NewHTTPStatusError(statusCode int, status, body string) error {
	return &HTTPStatusError{StatusCode: statusCode, Status: status, Body: body}
}
