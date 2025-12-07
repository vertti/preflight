package promcheck

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

// mockHTTPClient implements HTTPClient for testing.
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// mockResponse creates a mock HTTP response with the given status code and body.
func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// Sample Prometheus responses
const (
	promSuccessVector = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {"__name__": "up", "job": "myapp"},
					"value": [1702000000, "1"]
				}
			]
		}
	}`

	promSuccessScalar = `{
		"status": "success",
		"data": {
			"resultType": "scalar",
			"result": [1702000000, "42.5"]
		}
	}`

	promEmptyResult = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": []
		}
	}`

	promMultipleResults = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{"metric": {"job": "a"}, "value": [1702000000, "1"]},
				{"metric": {"job": "b"}, "value": [1702000000, "2"]}
			]
		}
	}`

	promError = `{
		"status": "error",
		"errorType": "bad_data",
		"error": "invalid expression"
	}`

	promValueFloat = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [{"metric": {}, "value": [1702000000, "0.05"]}]
		}
	}`

	promValueHigh = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [{"metric": {}, "value": [1702000000, "100"]}]
		}
	}`

	promValueLow = `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [{"metric": {}, "value": [1702000000, "5"]}]
		}
	}`
)

func ptr(f float64) *float64 {
	return &f
}

func TestPrometheusCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantName      string
		wantDetailSub string // substring to find in details or error
	}{
		// --- Validation ---
		{
			name: "missing URL fails",
			check: Check{
				Query: "up",
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "URL is required",
		},
		{
			name: "missing query fails",
			check: Check{
				URL: "http://prometheus:9090",
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "query is required",
		},
		{
			name: "invalid URL fails",
			check: Check{
				URL:   "not-a-url",
				Query: "up",
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "invalid URL",
		},

		// --- Successful Queries ---
		{
			name: "successful vector query",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up{job=\"myapp\"}",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil
					},
				},
			},
			wantStatus:    check.StatusOK,
			wantName:      "prometheus: http://prometheus:9090",
			wantDetailSub: "value: 1",
		},
		{
			name: "successful scalar query",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "42.5",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessScalar), nil
					},
				},
			},
			wantStatus:    check.StatusOK,
			wantDetailSub: "value: 42.5",
		},
		{
			name: "query shown in details",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "error_rate",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil
					},
				},
			},
			wantStatus:    check.StatusOK,
			wantDetailSub: "query: error_rate",
		},

		// --- Error Cases ---
		{
			name: "connection error",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return nil, errors.New("connection refused")
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "connection refused",
		},
		{
			name: "non-200 status",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(500, ""), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "prometheus returned status 500",
		},
		{
			name: "prometheus error response",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "invalid(",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promError), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "invalid expression",
		},
		{
			name: "empty result",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "nonexistent_metric",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promEmptyResult), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "returned no data",
		},
		{
			name: "multiple results fails",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promMultipleResults), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "returned 2 results",
		},

		// --- Value Thresholds ---
		{
			name: "exact match passes",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Exact: ptr(1),
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "exact match fails",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Exact: ptr(0),
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "does not equal 0",
		},
		{
			name: "min threshold passes",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "request_count",
				Min:   ptr(50),
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promValueHigh), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "min threshold fails",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "request_count",
				Min:   ptr(50),
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promValueLow), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "< minimum 50",
		},
		{
			name: "max threshold passes",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "error_rate",
				Max:   ptr(0.1),
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promValueFloat), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "max threshold fails",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "error_rate",
				Max:   ptr(0.01),
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promValueFloat), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "> maximum 0.01",
		},

		// --- Headers ---
		{
			name: "custom headers are sent",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Headers: map[string]string{
					"Authorization": "Bearer token123",
				},
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if req.Header.Get("Authorization") != "Bearer token123" {
							t.Error("Authorization header not set")
						}
						return mockResponse(200, promSuccessVector), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- URL Construction ---
		{
			name: "query is URL encoded",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up{job=\"myapp\"}",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						expectedQuery := "up%7Bjob%3D%22myapp%22%7D"
						if !strings.Contains(req.URL.RawQuery, expectedQuery) {
							t.Errorf("query not properly encoded: %s", req.URL.RawQuery)
						}
						return mockResponse(200, promSuccessVector), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (details: %v)", result.Status, tt.wantStatus, result.Details)
			}

			if tt.wantName != "" && result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}

			if tt.wantDetailSub != "" {
				found := false
				for _, d := range result.Details {
					if strings.Contains(d, tt.wantDetailSub) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Details %v should contain %q", result.Details, tt.wantDetailSub)
				}
			}
		})
	}
}

func TestPrometheusCheck_AdditionalCases(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantDetailSub string
	}{
		{
			name: "unsupported result type",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, `{"status":"success","data":{"resultType":"matrix","result":[]}}`), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "unsupported result type",
		},
		{
			name: "non-numeric value",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1702000000,"not-a-number"]}]}}`), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "failed to parse metric value",
		},
		{
			name: "prometheus status without error message",
			check: Check{
				URL:   "http://prometheus:9090",
				Query: "up",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, `{"status":"error"}`), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "prometheus returned status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (details: %v)", result.Status, tt.wantStatus, result.Details)
			}
			if tt.wantDetailSub != "" {
				found := false
				for _, d := range result.Details {
					if strings.Contains(d, tt.wantDetailSub) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Details %v should contain %q", result.Details, tt.wantDetailSub)
				}
			}
		})
	}
}

func TestPrometheusCheckRetryFailure(t *testing.T) {
	// Test retry exhaustion with "after N attempts" message
	attempts := 0
	c := Check{
		URL:        "http://prometheus:9090",
		Query:      "up",
		Retry:      2,
		RetryDelay: 1,
		Client: &mockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				attempts++
				return mockResponse(500, ""), nil
			},
		},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail", result.Status)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
	found := false
	for _, d := range result.Details {
		if strings.Contains(d, "after 3 attempts") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Details %v should contain 'after 3 attempts'", result.Details)
	}
}

func TestPrometheusCheckRetry(t *testing.T) {
	attempts := 0
	c := Check{
		URL:        "http://prometheus:9090",
		Query:      "up",
		Retry:      2,
		RetryDelay: 1, // 1 nanosecond for fast tests
		Client: &mockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				attempts++
				if attempts < 3 {
					return nil, errors.New("connection refused")
				}
				return mockResponse(200, promSuccessVector), nil
			},
		},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want OK", result.Status)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestPrometheusCheckRetryConnectionFailure(t *testing.T) {
	// Test connection failure with retry exhaustion
	c := Check{
		URL:        "http://prometheus:9090",
		Query:      "up",
		Retry:      1,
		RetryDelay: 1,
		Client: &mockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail", result.Status)
	}
	found := false
	for _, d := range result.Details {
		if strings.Contains(d, "after 2 attempts") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Details %v should contain 'after 2 attempts'", result.Details)
	}
}

func TestPrometheusCheckRetryEmptyResult(t *testing.T) {
	// Test empty result with retry exhaustion
	c := Check{
		URL:        "http://prometheus:9090",
		Query:      "nonexistent",
		Retry:      1,
		RetryDelay: 1,
		Client: &mockHTTPClient{
			DoFunc: func(req *http.Request) (*http.Response, error) {
				return mockResponse(200, promEmptyResult), nil
			},
		},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want Fail", result.Status)
	}
	found := false
	for _, d := range result.Details {
		if strings.Contains(d, "after 2 attempts") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Details %v should contain 'after 2 attempts'", result.Details)
	}
}

func TestPrometheusCheckRetryThresholdFailure(t *testing.T) {
	tests := []struct {
		name  string
		check Check
	}{
		{
			name: "exact mismatch retry",
			check: Check{
				URL:        "http://prometheus:9090",
				Query:      "up",
				Exact:      ptr(0),
				Retry:      1,
				RetryDelay: 1,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil // value is 1, not 0
					},
				},
			},
		},
		{
			name: "min threshold retry",
			check: Check{
				URL:        "http://prometheus:9090",
				Query:      "up",
				Min:        ptr(10),
				Retry:      1,
				RetryDelay: 1,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil // value is 1, < 10
					},
				},
			},
		},
		{
			name: "max threshold retry",
			check: Check{
				URL:        "http://prometheus:9090",
				Query:      "up",
				Max:        ptr(0.5),
				Retry:      1,
				RetryDelay: 1,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200, promSuccessVector), nil // value is 1, > 0.5
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != check.StatusFail {
				t.Errorf("Status = %v, want Fail", result.Status)
			}
			found := false
			for _, d := range result.Details {
				if strings.Contains(d, "after 2 attempts") {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Details %v should contain 'after 2 attempts'", result.Details)
			}
		})
	}
}
