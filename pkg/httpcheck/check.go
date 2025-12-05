package httpcheck

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/vertti/preflight/pkg/check"
)

// HTTPClient abstracts HTTP requests for testability.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// RealHTTPClient uses the real net/http package.
type RealHTTPClient struct {
	Timeout  time.Duration
	Insecure bool
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
		// Disable automatic redirects - check actual response status
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return client.Do(req)
}

// Check verifies HTTP endpoint health.
type Check struct {
	URL            string            // target URL (required)
	ExpectedStatus int               // expected HTTP status (default: 200)
	Timeout        time.Duration     // request timeout (default: 5s)
	Method         string            // HTTP method (default: GET)
	Headers        map[string]string // custom headers
	Insecure       bool              // skip TLS verification
	Retry          int               // retry count on failure
	RetryDelay     time.Duration     // delay between retries
	Client         HTTPClient        // injected for testing
}

// Run executes the HTTP health check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("http: %s", c.URL),
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
		client = &RealHTTPClient{Timeout: timeout, Insecure: c.Insecure}
	}

	// Retry loop
	maxAttempts := c.Retry + 1
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest(method, c.URL, http.NoBody)
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
		_ = resp.Body.Close()

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
