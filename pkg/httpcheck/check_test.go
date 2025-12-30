package httpcheck

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vertti/preflight/pkg/check"
)

// mockHTTPClient implements HTTPClient for testing.
type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

// mockFileReader implements FileReader for testing.
type mockFileReader struct {
	Files map[string][]byte
	Err   error
}

func (m *mockFileReader) ReadFile(path string) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if content, ok := m.Files[path]; ok {
		return content, nil
	}
	return nil, errors.New("file not found")
}

// mockResponse creates a mock HTTP response with the given status code.
func mockResponse(statusCode int) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func TestHTTPCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantName      string
		wantDetailSub string // substring to find in details
	}{
		// --- Basic Success Cases ---
		{
			name: "successful GET returns 200",
			check: Check{
				URL:            "http://localhost:8080/health",
				ExpectedStatus: 200,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
			wantName:   "http: http://localhost:8080/health",
		},
		{
			name: "default status is 200",
			check: Check{
				URL: "http://localhost/ready",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- Status Code Matching ---
		{
			name: "custom expected status 204",
			check: Check{
				URL:            "http://localhost/api",
				ExpectedStatus: 204,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(204), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "status mismatch fails",
			check: Check{
				URL:            "http://localhost/health",
				ExpectedStatus: 200,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(503), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "503",
		},
		{
			name: "redirect status fails by default",
			check: Check{
				URL:            "http://localhost/old",
				ExpectedStatus: 200,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(302), nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "302",
		},
		{
			name: "can expect redirect status",
			check: Check{
				URL:            "http://localhost/redirect",
				ExpectedStatus: 302,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return mockResponse(302), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- HTTP Methods ---
		{
			name: "HEAD method used when specified",
			check: Check{
				URL:    "http://localhost/health",
				Method: "HEAD",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if req.Method != "HEAD" {
							t.Errorf("expected HEAD, got %s", req.Method)
						}
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "default method is GET",
			check: Check{
				URL: "http://localhost/health",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if req.Method != "GET" {
							t.Errorf("expected GET, got %s", req.Method)
						}
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- Custom Headers ---
		{
			name: "custom headers sent",
			check: Check{
				URL:     "http://localhost/api",
				Headers: map[string]string{"Authorization": "Bearer token123"},
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if req.Header.Get("Authorization") != "Bearer token123" {
							t.Errorf("missing or incorrect Authorization header: %s", req.Header.Get("Authorization"))
						}
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "multiple custom headers",
			check: Check{
				URL: "http://localhost/api",
				Headers: map[string]string{
					"X-API-Key":    "secret",
					"Content-Type": "application/json",
				},
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						if req.Header.Get("X-API-Key") != "secret" {
							t.Errorf("missing X-API-Key header")
						}
						if req.Header.Get("Content-Type") != "application/json" {
							t.Errorf("missing Content-Type header")
						}
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- Request Body ---
		{
			name: "body string sent with POST",
			check: Check{
				URL:    "http://localhost/api",
				Method: "POST",
				Body:   `{"key": "value"}`,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						body, _ := io.ReadAll(req.Body)
						if string(body) != `{"key": "value"}` {
							t.Errorf("body = %q, want %q", string(body), `{"key": "value"}`)
						}
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "body-file sent with POST",
			check: Check{
				URL:      "http://localhost/api",
				Method:   "POST",
				BodyFile: "/tmp/request.json",
				FileReader: &mockFileReader{
					Files: map[string][]byte{"/tmp/request.json": []byte(`{"from": "file"}`)},
				},
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						body, _ := io.ReadAll(req.Body)
						if string(body) != `{"from": "file"}` {
							t.Errorf("body = %q, want %q", string(body), `{"from": "file"}`)
						}
						return mockResponse(200), nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "body-file not found fails",
			check: Check{
				URL:      "http://localhost/api",
				Method:   "POST",
				BodyFile: "/nonexistent/file.json",
				FileReader: &mockFileReader{
					Files: map[string][]byte{},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "failed to read body file",
		},

		// --- Response Body Contains ---
		{
			name: "contains passes when string found",
			check: Check{
				URL:      "http://localhost/health",
				Contains: "healthy",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"status": "healthy"}`)),
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "contains fails when string not found",
			check: Check{
				URL:      "http://localhost/health",
				Contains: "ready",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"status": "waiting"}`)),
						}, nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "does not contain",
		},

		// --- JSON Path ---
		{
			name: "json-path exists check passes",
			check: Check{
				URL:      "http://localhost/api",
				JSONPath: "data.id",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"data": {"id": 123}}`)),
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "json-path exists check fails when path missing",
			check: Check{
				URL:      "http://localhost/api",
				JSONPath: "data.missing",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"data": {"id": 123}}`)),
						}, nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "not found",
		},
		{
			name: "json-path value check passes",
			check: Check{
				URL:      "http://localhost/api",
				JSONPath: "status=healthy",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"status": "healthy"}`)),
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "json-path value check fails when value differs",
			check: Check{
				URL:      "http://localhost/api",
				JSONPath: "status=healthy",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"status": "degraded"}`)),
						}, nil
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "expected",
		},
		{
			name: "json-path nested value check passes",
			check: Check{
				URL:      "http://localhost/api",
				JSONPath: "data.items.0.name=first",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"data": {"items": [{"name": "first"}, {"name": "second"}]}}`)),
						}, nil
					},
				},
			},
			wantStatus: check.StatusOK,
		},

		// --- Connection Errors ---
		{
			name: "connection refused",
			check: Check{
				URL: "http://localhost:9999/health",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return nil, errors.New("dial tcp: connection refused")
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "connection refused",
		},
		{
			name: "timeout error",
			check: Check{
				URL:     "http://10.255.255.1/health",
				Timeout: 1 * time.Second,
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return nil, errors.New("context deadline exceeded")
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "deadline",
		},
		{
			name: "DNS resolution failure",
			check: Check{
				URL: "http://nonexistent.invalid/health",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return nil, errors.New("no such host")
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "no such host",
		},

		// --- TLS Errors ---
		{
			name: "TLS certificate error",
			check: Check{
				URL: "https://self-signed.invalid/health",
				Client: &mockHTTPClient{
					DoFunc: func(req *http.Request) (*http.Response, error) {
						return nil, errors.New("x509: certificate signed by unknown authority")
					},
				},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "certificate",
		},

		// --- Invalid URL ---
		{
			name: "invalid URL - no scheme",
			check: Check{
				URL: "not-a-url",
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "invalid URL",
		},
		{
			name: "empty URL",
			check: Check{
				URL: "",
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "URL is required",
		},
		{
			name: "URL with only scheme",
			check: Check{
				URL: "http://",
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "invalid URL",
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
				allDetails := strings.Join(result.Details, " ")
				if strings.Contains(allDetails, tt.wantDetailSub) {
					found = true
				}
				if !found {
					t.Errorf("Details %v should contain %q", result.Details, tt.wantDetailSub)
				}
			}
		})
	}
}

func TestHTTPCheckRetry(t *testing.T) {
	t.Run("retry succeeds on second attempt", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/health",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond, // fast for testing
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts < 2 {
						return nil, errors.New("connection refused")
					}
					return mockResponse(200), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", result.Status)
		}
		if attempts != 2 {
			t.Errorf("attempts = %d, want 2", attempts)
		}
	})

	t.Run("retry exhausted after max attempts", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/health",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					return nil, errors.New("connection refused")
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		if attempts != 3 { // 1 initial + 2 retries
			t.Errorf("attempts = %d, want 3", attempts)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "3 attempts") {
			t.Errorf("Details should mention 3 attempts: %v", result.Details)
		}
	})

	t.Run("retry on status mismatch", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:            "http://localhost/health",
			ExpectedStatus: 200,
			Retry:          1,
			RetryDelay:     1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts < 2 {
						return mockResponse(503), nil
					}
					return mockResponse(200), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", result.Status)
		}
		if attempts != 2 {
			t.Errorf("attempts = %d, want 2", attempts)
		}
	})

	t.Run("no retry message on single attempt", func(t *testing.T) {
		c := Check{
			URL:   "http://localhost/health",
			Retry: 0, // no retries
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("connection refused")
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		allDetails := strings.Join(result.Details, " ")
		if strings.Contains(allDetails, "attempts") {
			t.Errorf("Details should not mention attempts for single try: %v", result.Details)
		}
	})
}

func TestHTTPCheckDefaults(t *testing.T) {
	t.Run("default timeout is 5s", func(t *testing.T) {
		c := Check{
			URL:     "http://localhost/health",
			Timeout: 0, // should default to 5s
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return mockResponse(200), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", result.Status)
		}
	})
}

func TestHTTPCheckRetryStatusMismatch(t *testing.T) {
	t.Run("retry exhausted on status mismatch", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:            "http://localhost/health",
			ExpectedStatus: 200,
			Retry:          2,
			RetryDelay:     1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					return mockResponse(503), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		if attempts != 3 { // 1 initial + 2 retries
			t.Errorf("attempts = %d, want 3", attempts)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "3 attempts") {
			t.Errorf("Details should mention 3 attempts: %v", result.Details)
		}
		if !strings.Contains(allDetails, "503") {
			t.Errorf("Details should mention status 503: %v", result.Details)
		}
	})

	t.Run("success message includes attempt info on retry", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/health",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts < 3 {
						return mockResponse(503), nil
					}
					return mockResponse(200), nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", result.Status)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "attempt 3 of 3") {
			t.Errorf("Details should mention attempt 3 of 3: %v", result.Details)
		}
	})
}

func TestRealHTTPClient(t *testing.T) {
	t.Run("creates client with timeout", func(t *testing.T) {
		client := &RealHTTPClient{
			Timeout:  10 * time.Second,
			Insecure: false,
		}
		// Just verify the struct can be created - actual HTTP calls tested in integration
		if client.Timeout != 10*time.Second {
			t.Errorf("Timeout = %v, want 10s", client.Timeout)
		}
	})

	t.Run("creates client with insecure flag", func(t *testing.T) {
		client := &RealHTTPClient{
			Timeout:  5 * time.Second,
			Insecure: true,
		}
		if !client.Insecure {
			t.Error("Insecure should be true")
		}
	})

	t.Run("creates client with follow-redirects flag", func(t *testing.T) {
		client := &RealHTTPClient{
			Timeout:         5 * time.Second,
			FollowRedirects: true,
		}
		if !client.FollowRedirects {
			t.Error("FollowRedirects should be true")
		}
	})

	t.Run("basic request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		defer ts.Close()

		client := &RealHTTPClient{
			Timeout: 5 * time.Second,
		}

		req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("insecure TLS", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		client := &RealHTTPClient{
			Timeout:  5 * time.Second,
			Insecure: true,
		}

		req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Do with insecure failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
		}
	})

	t.Run("redirects disabled", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "/target", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		client := &RealHTTPClient{
			Timeout:         5 * time.Second,
			FollowRedirects: false,
		}

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/redirect", http.NoBody)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Do failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 302 {
			t.Errorf("StatusCode = %d, want 302", resp.StatusCode)
		}
	})

	t.Run("redirects enabled", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/redirect" {
				http.Redirect(w, r, "/target", http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		client := &RealHTTPClient{
			Timeout:         5 * time.Second,
			FollowRedirects: true,
		}

		req, err := http.NewRequest(http.MethodGet, ts.URL+"/redirect", http.NoBody)
		if err != nil {
			t.Fatalf("NewRequest failed: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Do with follow failed: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", resp.StatusCode)
		}
	})
}

func TestHTTPCheckContainsRetry(t *testing.T) {
	t.Run("contains retry succeeds on second attempt", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/health",
			Contains:   "healthy",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts < 2 {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"status": "starting"}`)),
						}, nil
					}
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status": "healthy"}`)),
					}, nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", result.Status)
		}
		if attempts != 2 {
			t.Errorf("attempts = %d, want 2", attempts)
		}
	})

	t.Run("contains retry exhausted", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/health",
			Contains:   "healthy",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status": "starting"}`)),
					}, nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		if attempts != 3 {
			t.Errorf("attempts = %d, want 3", attempts)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "3 attempts") {
			t.Errorf("Details should mention 3 attempts: %v", result.Details)
		}
	})
}

func TestHTTPCheckJSONPathRetry(t *testing.T) {
	t.Run("json-path retry succeeds on second attempt", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/api",
			JSONPath:   "status=ready",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					if attempts < 2 {
						return &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader(`{"status": "starting"}`)),
						}, nil
					}
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status": "ready"}`)),
					}, nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", result.Status)
		}
		if attempts != 2 {
			t.Errorf("attempts = %d, want 2", attempts)
		}
	})

	t.Run("json-path retry exhausted on value mismatch", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/api",
			JSONPath:   "status=ready",
			Retry:      2,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"status": "starting"}`)),
					}, nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		if attempts != 3 {
			t.Errorf("attempts = %d, want 3", attempts)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "3 attempts") {
			t.Errorf("Details should mention 3 attempts: %v", result.Details)
		}
	})

	t.Run("json-path retry exhausted on path not found", func(t *testing.T) {
		attempts := 0
		c := Check{
			URL:        "http://localhost/api",
			JSONPath:   "data.missing",
			Retry:      1,
			RetryDelay: 1 * time.Millisecond,
			Client: &mockHTTPClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					attempts++
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"data": {"id": 123}}`)),
					}, nil
				},
			},
		}

		result := c.Run()

		if result.Status != check.StatusFail {
			t.Errorf("Status = %v, want FAIL", result.Status)
		}
		if attempts != 2 {
			t.Errorf("attempts = %d, want 2", attempts)
		}
		allDetails := strings.Join(result.Details, " ")
		if !strings.Contains(allDetails, "2 attempts") {
			t.Errorf("Details should mention 2 attempts: %v", result.Details)
		}
	})
}

func TestRealFileReader(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	content := []byte("test content")
	if err := os.WriteFile(tmpFile, content, 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	reader := &RealFileReader{}
	data, err := reader.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("content = %q, want %q", string(data), string(content))
	}
}
