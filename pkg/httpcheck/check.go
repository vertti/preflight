package httpcheck

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"

	"github.com/vertti/preflight/pkg/check"
)

// HTTPClient abstracts HTTP requests for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RealHTTPClient uses the real net/http package.
type RealHTTPClient struct {
	Timeout         time.Duration
	Insecure        bool
	FollowRedirects bool
}

// Do executes an HTTP request.
func (c *RealHTTPClient) Do(req *http.Request) (*http.Response, error) {
	transport := &http.Transport{}
	if c.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional for --insecure flag
	}

	client := &http.Client{
		Timeout:   c.Timeout,
		Transport: transport,
	}

	// Disable automatic redirects unless explicitly enabled
	if !c.FollowRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client.Do(req)
}

// FileReader abstracts file reading for testability.
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// RealFileReader uses the real os package.
type RealFileReader struct{}

// ReadFile reads a file from the filesystem.
func (r *RealFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Check verifies HTTP endpoint health.
type Check struct {
	URL             string            // target URL (required)
	ExpectedStatus  int               // expected HTTP status (default: 200)
	Timeout         time.Duration     // request timeout (default: 5s)
	Method          string            // HTTP method (default: GET)
	Headers         map[string]string // custom headers
	Insecure        bool              // skip TLS verification
	Retry           int               // retry count on failure
	RetryDelay      time.Duration     // delay between retries
	Body            string            // request body string
	BodyFile        string            // path to file containing request body
	Contains        string            // response body must contain this string
	FollowRedirects bool              // follow HTTP redirects (3xx)
	JSONPath        string            // JSON path to check (format: "path=expectedValue" or just "path")
	Client          HTTPClient        // injected for testing
	FileReader      FileReader        // injected for testing
}

// Run executes the HTTP health check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "http: " + c.URL,
	}

	// Validate URL
	if c.URL == "" {
		return result.Failf("URL is required")
	}
	parsedURL, err := url.Parse(c.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return result.Failf("invalid URL: %s", c.URL)
	}

	// Set defaults
	method := c.Method
	if method == "" {
		method = "GET"
	}
	expectedStatus := c.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}
	timeout := c.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	retryDelay := c.RetryDelay
	if retryDelay == 0 {
		retryDelay = 1 * time.Second
	}

	// Initialize client if not injected
	client := c.Client
	if client == nil {
		client = &RealHTTPClient{Timeout: timeout, Insecure: c.Insecure, FollowRedirects: c.FollowRedirects}
	}

	// Resolve request body
	var bodyBytes []byte
	if c.BodyFile != "" {
		reader := c.FileReader
		if reader == nil {
			reader = &RealFileReader{}
		}
		var err error
		bodyBytes, err = reader.ReadFile(c.BodyFile)
		if err != nil {
			return result.Failf("failed to read body file: %v", err)
		}
	} else if c.Body != "" {
		bodyBytes = []byte(c.Body)
	}

	// Retry loop
	maxAttempts := c.Retry + 1
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		var bodyReader io.Reader = http.NoBody
		if len(bodyBytes) > 0 {
			bodyReader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequest(method, c.URL, bodyReader)
		if err != nil {
			return result.Failf("failed to create request: %v", err)
		}

		// Add custom headers
		for k, v := range c.Headers {
			req.Header.Set(k, v)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("request failed after %d attempts: %v", maxAttempts, err)
			}
			return result.Failf("request failed: %v", err)
		}

		statusCode := resp.StatusCode

		// Read response body if needed for --contains or --json-path check
		var respBody string
		if c.Contains != "" || c.JSONPath != "" {
			bodyBytes, err := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if err != nil {
				return result.Failf("failed to read response body: %v", err)
			}
			respBody = string(bodyBytes)
		} else {
			_ = resp.Body.Close()
		}

		if statusCode != expectedStatus {
			lastErr = fmt.Errorf("status %d, expected %d", statusCode, expectedStatus)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("status %d, expected %d (after %d attempts)", statusCode, expectedStatus, maxAttempts)
			}
			return result.Failf("status %d, expected %d", statusCode, expectedStatus)
		}

		// Check --contains
		if c.Contains != "" && !strings.Contains(respBody, c.Contains) {
			lastErr = fmt.Errorf("response body does not contain %q", c.Contains)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("response body does not contain %q (after %d attempts)", c.Contains, maxAttempts)
			}
			return result.Failf("response body does not contain %q", c.Contains)
		}

		// Check --json-path
		if c.JSONPath != "" {
			path, expectedValue, hasExpectedValue := parseJSONPath(c.JSONPath)
			jsonResult := gjson.Get(respBody, path)
			if !jsonResult.Exists() {
				lastErr = fmt.Errorf("JSON path %q not found", path)
				if attempt < maxAttempts {
					time.Sleep(retryDelay)
					continue
				}
				if maxAttempts > 1 {
					return result.Failf("JSON path %q not found (after %d attempts)", path, maxAttempts)
				}
				return result.Failf("JSON path %q not found", path)
			}
			if hasExpectedValue && jsonResult.String() != expectedValue {
				lastErr = fmt.Errorf("JSON path %q: got %q, expected %q", path, jsonResult.String(), expectedValue)
				if attempt < maxAttempts {
					time.Sleep(retryDelay)
					continue
				}
				if maxAttempts > 1 {
					return result.Failf("JSON path %q: got %q, expected %q (after %d attempts)", path, jsonResult.String(), expectedValue, maxAttempts)
				}
				return result.Failf("JSON path %q: got %q, expected %q", path, jsonResult.String(), expectedValue)
			}
		}

		// Success
		result.Status = check.StatusOK
		result.AddDetailf("status %d", statusCode)
		if maxAttempts > 1 && attempt > 1 {
			result.AddDetailf("succeeded on attempt %d of %d", attempt, maxAttempts)
		}
		return result
	}

	// Should not reach here, but handle edge case
	return result.Failf("unexpected error: %v", lastErr)
}

// parseJSONPath parses "path=value" or "path" format.
func parseJSONPath(jsonPath string) (path, expectedValue string, hasExpectedValue bool) {
	if idx := strings.Index(jsonPath, "="); idx != -1 {
		return jsonPath[:idx], jsonPath[idx+1:], true
	}
	return jsonPath, "", false
}
