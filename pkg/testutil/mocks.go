package testutil

import (
	"io"
	"net/http"
	"strings"
)

// MockHTTPClient is a test double for HTTP clients.
type MockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// MockResponse creates an http.Response with given status and body.
func MockResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

// Ptr returns a pointer to the value (useful for optional fields in tests).
func Ptr[T any](v T) *T {
	return &v
}

// ContainsDetail checks if any detail string contains the given substring.
func ContainsDetail(details []string, substr string) bool {
	for _, d := range details {
		if strings.Contains(d, substr) {
			return true
		}
	}
	return false
}
