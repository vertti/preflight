package promcheck

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	}

	return client.Do(req)
}

// Check queries Prometheus and validates metric values.
type Check struct {
	URL        string            // Prometheus server URL (required)
	Query      string            // PromQL query (required)
	Min        *float64          // minimum value (fail if below)
	Max        *float64          // maximum value (fail if above)
	Exact      *float64          // exact value match
	Timeout    time.Duration     // request timeout (default: 5s)
	Retry      int               // retry count on failure
	RetryDelay time.Duration     // delay between retries (default: 1s)
	Insecure   bool              // skip TLS verification
	Headers    map[string]string // custom headers (for auth)
	Client     HTTPClient        // injected for testing
}

// Run executes the Prometheus query check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "prometheus: " + c.URL,
	}

	// Validate required fields
	if c.URL == "" {
		return result.Failf("URL is required")
	}
	if c.Query == "" {
		return result.Failf("query is required")
	}

	// Validate URL
	parsedURL, err := url.Parse(c.URL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return result.Failf("invalid URL: %s", c.URL)
	}

	// Set defaults
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

	// Build query URL
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", c.URL, url.QueryEscape(c.Query))

	// Retry loop
	maxAttempts := c.Retry + 1
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequest("GET", queryURL, http.NoBody)
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

		// Read response body
		bodyBytes := make([]byte, 0, 4096)
		buf := make([]byte, 4096)
		for {
			n, err := resp.Body.Read(buf)
			if n > 0 {
				bodyBytes = append(bodyBytes, buf[:n]...)
			}
			if err != nil {
				break
			}
		}
		_ = resp.Body.Close()
		respBody := string(bodyBytes)

		// Check HTTP status
		if resp.StatusCode != 200 {
			lastErr = fmt.Errorf("prometheus returned status %d", resp.StatusCode)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("prometheus returned status %d (after %d attempts)", resp.StatusCode, maxAttempts)
			}
			return result.Failf("prometheus returned status %d", resp.StatusCode)
		}

		// Parse Prometheus response
		status := gjson.Get(respBody, "status").String()
		if status != "success" {
			errorMsg := gjson.Get(respBody, "error").String()
			if errorMsg != "" {
				return result.Failf("prometheus error: %s", errorMsg)
			}
			return result.Failf("prometheus returned status %q", status)
		}

		// Get result type and extract value
		resultType := gjson.Get(respBody, "data.resultType").String()
		var valueStr string
		var metricLabels string

		switch resultType {
		case "vector":
			results := gjson.Get(respBody, "data.result")
			if !results.Exists() || len(results.Array()) == 0 {
				lastErr = errors.New("query returned no data")
				if attempt < maxAttempts {
					time.Sleep(retryDelay)
					continue
				}
				if maxAttempts > 1 {
					return result.Failf("query %q returned no data (after %d attempts)", c.Query, maxAttempts)
				}
				return result.Failf("query %q returned no data", c.Query)
			}
			if len(results.Array()) > 1 {
				return result.Failf("query returned %d results, expected 1 (use a more specific query)", len(results.Array()))
			}
			valueStr = gjson.Get(respBody, "data.result.0.value.1").String()
			metricLabels = gjson.Get(respBody, "data.result.0.metric").String()
		case "scalar":
			valueStr = gjson.Get(respBody, "data.result.1").String()
		default:
			return result.Failf("unsupported result type: %s", resultType)
		}

		// Parse value as float64
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			return result.Failf("failed to parse metric value %q as number", valueStr)
		}

		// Validate against thresholds
		if c.Exact != nil && value != *c.Exact {
			lastErr = fmt.Errorf("value %v does not equal %v", value, *c.Exact)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("value %v does not equal %v (after %d attempts)", value, *c.Exact, maxAttempts)
			}
			return result.Failf("value %v does not equal %v", value, *c.Exact)
		}

		if c.Min != nil && value < *c.Min {
			lastErr = fmt.Errorf("value %v < minimum %v", value, *c.Min)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("value %v < minimum %v (after %d attempts)", value, *c.Min, maxAttempts)
			}
			return result.Failf("value %v < minimum %v", value, *c.Min)
		}

		if c.Max != nil && value > *c.Max {
			lastErr = fmt.Errorf("value %v > maximum %v", value, *c.Max)
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			if maxAttempts > 1 {
				return result.Failf("value %v > maximum %v (after %d attempts)", value, *c.Max, maxAttempts)
			}
			return result.Failf("value %v > maximum %v", value, *c.Max)
		}

		// Success
		result.Status = check.StatusOK
		result.AddDetailf("query: %s", c.Query)
		if metricLabels != "" {
			result.AddDetailf("metric: %s", metricLabels)
		}
		result.AddDetailf("value: %v", value)
		if maxAttempts > 1 && attempt > 1 {
			result.AddDetailf("succeeded on attempt %d of %d", attempt, maxAttempts)
		}
		return result
	}

	// Should not reach here, but handle edge case
	return result.Failf("unexpected error: %v", lastErr)
}
