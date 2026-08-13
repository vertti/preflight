package promcheck

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/httpclient"
	"github.com/vertti/preflight/pkg/jsonpath"
)

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
	Client     httpclient.Client // injected for testing
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
		client = &httpclient.Real{Timeout: timeout, Insecure: c.Insecure}
	}

	// Build query URL (trim trailing slash to avoid double slash)
	baseURL := strings.TrimSuffix(c.URL, "/")
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", baseURL, url.QueryEscape(c.Query))

	// Retry loop
	maxAttempts := c.Retry + 1

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
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			return result.FailAfter(maxAttempts, "request failed: %v", err)
		}

		// Read response body
		respBody, err := httpclient.ReadBody(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return result.Failf("failed to read response body: %v", err)
		}

		// Check HTTP status
		if resp.StatusCode != 200 {
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			return result.FailAfter(maxAttempts, "prometheus returned status %d", resp.StatusCode)
		}

		// Parse Prometheus response
		status := jsonpath.Get(respBody, "status").String()
		if status != "success" {
			errorMsg := jsonpath.Get(respBody, "error").String()
			if errorMsg != "" {
				return result.Failf("prometheus error: %s", errorMsg)
			}
			return result.Failf("prometheus returned status %q", status)
		}

		// Get result type and extract value
		resultType := jsonpath.Get(respBody, "data.resultType").String()
		var valueStr string
		var metricLabels string

		switch resultType {
		case "vector":
			results := jsonpath.Get(respBody, "data.result")
			if !results.Exists() || len(results.Array()) == 0 {
				if attempt < maxAttempts {
					time.Sleep(retryDelay)
					continue
				}
				return result.FailAfter(maxAttempts, "query %q returned no data", c.Query)
			}
			if len(results.Array()) > 1 {
				return result.Failf("query returned %d results, expected 1 (use a more specific query)", len(results.Array()))
			}
			valueStr = jsonpath.Get(respBody, "data.result.0.value.1").String()
			metricLabels = jsonpath.Get(respBody, "data.result.0.metric").String()
		case "scalar":
			valueStr = jsonpath.Get(respBody, "data.result.1").String()
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
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			return result.FailAfter(maxAttempts, "value %v does not equal %v", value, *c.Exact)
		}

		if c.Min != nil && value < *c.Min {
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			return result.FailAfter(maxAttempts, "value %v < minimum %v", value, *c.Min)
		}

		if c.Max != nil && value > *c.Max {
			if attempt < maxAttempts {
				time.Sleep(retryDelay)
				continue
			}
			return result.FailAfter(maxAttempts, "value %v > maximum %v", value, *c.Max)
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

	// Unreachable: every path in the loop returns, or continues only while
	// tries remain. The compiler cannot see that, so the line has to exist.
	return result.Failf("check did not complete")
}
