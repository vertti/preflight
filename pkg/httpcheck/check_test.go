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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/testutil"
)

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

func clientOK() *testutil.MockHTTPClient {
	return &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) { return testutil.MockResponse(200, ""), nil }}
}

func clientStatus(code int) *testutil.MockHTTPClient {
	return &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) { return testutil.MockResponse(code, ""), nil }}
}

func clientErr(msg string) *testutil.MockHTTPClient {
	return &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) { return nil, errors.New(msg) }}
}

func clientBody(body string) *testutil.MockHTTPClient {
	return &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) { return testutil.MockResponse(200, body), nil }}
}

func TestHTTPCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantDetailSub string
	}{
		// Basic success
		{"successful GET 200", Check{URL: "http://localhost:8080/health", ExpectedStatus: 200, Client: clientOK()}, check.StatusOK, ""},
		{"default status 200", Check{URL: "http://localhost/ready", Client: clientOK()}, check.StatusOK, ""},

		// Status code matching
		{"custom expected 204", Check{URL: "http://localhost/api", ExpectedStatus: 204, Client: clientStatus(204)}, check.StatusOK, ""},
		{"status mismatch fails", Check{URL: "http://localhost/health", ExpectedStatus: 200, Client: clientStatus(503)}, check.StatusFail, "503"},
		{"redirect fails by default", Check{URL: "http://localhost/old", ExpectedStatus: 200, Client: clientStatus(302)}, check.StatusFail, "302"},
		{"can expect redirect", Check{URL: "http://localhost/redirect", ExpectedStatus: 302, Client: clientStatus(302)}, check.StatusOK, ""},

		// HTTP methods
		{"HEAD method", Check{URL: "http://localhost/health", Method: "HEAD", Client: &testutil.MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "HEAD", req.Method)
			return testutil.MockResponse(200, ""), nil
		}}}, check.StatusOK, ""},
		{"default GET method", Check{URL: "http://localhost/health", Client: &testutil.MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "GET", req.Method)
			return testutil.MockResponse(200, ""), nil
		}}}, check.StatusOK, ""},

		// Custom headers
		{"custom headers sent", Check{URL: "http://localhost/api", Headers: map[string]string{"Authorization": "Bearer token123"}, Client: &testutil.MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
			return testutil.MockResponse(200, ""), nil
		}}}, check.StatusOK, ""},

		// Request body
		{"body string sent", Check{URL: "http://localhost/api", Method: "POST", Body: `{"key": "value"}`, Client: &testutil.MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			assert.Equal(t, `{"key": "value"}`, string(body))
			return testutil.MockResponse(200, ""), nil
		}}}, check.StatusOK, ""},
		{"body-file sent", Check{URL: "http://localhost/api", Method: "POST", BodyFile: "/tmp/request.json", FileReader: &mockFileReader{Files: map[string][]byte{"/tmp/request.json": []byte(`{"from": "file"}`)}}, Client: &testutil.MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			body, _ := io.ReadAll(req.Body)
			assert.Equal(t, `{"from": "file"}`, string(body))
			return testutil.MockResponse(200, ""), nil
		}}}, check.StatusOK, ""},
		{"body-file not found", Check{URL: "http://localhost/api", Method: "POST", BodyFile: "/nonexistent/file.json", FileReader: &mockFileReader{Files: map[string][]byte{}}}, check.StatusFail, "failed to read body file"},

		// Response body contains
		{"contains passes", Check{URL: "http://localhost/health", Contains: "healthy", Client: clientBody(`{"status": "healthy"}`)}, check.StatusOK, ""},
		{"contains fails", Check{URL: "http://localhost/health", Contains: "ready", Client: clientBody(`{"status": "waiting"}`)}, check.StatusFail, "does not contain"},

		// JSON path
		{"json-path exists passes", Check{URL: "http://localhost/api", JSONPath: "data.id", Client: clientBody(`{"data": {"id": 123}}`)}, check.StatusOK, ""},
		{"json-path exists fails", Check{URL: "http://localhost/api", JSONPath: "data.missing", Client: clientBody(`{"data": {"id": 123}}`)}, check.StatusFail, "not found"},
		{"json-path value passes", Check{URL: "http://localhost/api", JSONPath: "status=healthy", Client: clientBody(`{"status": "healthy"}`)}, check.StatusOK, ""},
		{"json-path value fails", Check{URL: "http://localhost/api", JSONPath: "status=healthy", Client: clientBody(`{"status": "degraded"}`)}, check.StatusFail, "expected"},
		{"json-path nested", Check{URL: "http://localhost/api", JSONPath: "data.items.0.name=first", Client: clientBody(`{"data": {"items": [{"name": "first"}, {"name": "second"}]}}`)}, check.StatusOK, ""},

		// Connection errors
		{"connection refused", Check{URL: "http://localhost:9999/health", Client: clientErr("dial tcp: connection refused")}, check.StatusFail, "connection refused"},
		{"timeout error", Check{URL: "http://10.255.255.1/health", Timeout: time.Second, Client: clientErr("context deadline exceeded")}, check.StatusFail, "deadline"},
		{"DNS resolution failure", Check{URL: "http://nonexistent.invalid/health", Client: clientErr("no such host")}, check.StatusFail, "no such host"},
		{"TLS certificate error", Check{URL: "https://self-signed.invalid/health", Client: clientErr("x509: certificate signed by unknown authority")}, check.StatusFail, "certificate"},

		// Invalid URL
		{"invalid URL no scheme", Check{URL: "not-a-url"}, check.StatusFail, "invalid URL"},
		{"empty URL", Check{URL: ""}, check.StatusFail, "URL is required"},
		{"URL only scheme", Check{URL: "http://"}, check.StatusFail, "invalid URL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			if tt.wantDetailSub != "" {
				assert.Contains(t, strings.Join(result.Details, " "), tt.wantDetailSub)
			}
		})
	}
}

func TestHTTPCheckRetry(t *testing.T) {
	t.Run("succeeds on second attempt", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 2 {
				return nil, errors.New("connection refused")
			}
			return testutil.MockResponse(200, ""), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status)
		assert.Equal(t, 2, attempts)
	})

	t.Run("exhausted after max attempts", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", Retry: 2, RetryDelay: time.Millisecond, Client: clientErr("connection refused")}
		c.Client = &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			return nil, errors.New("connection refused")
		}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Equal(t, 3, attempts)
		assert.Contains(t, strings.Join(result.Details, " "), "3 attempts")
	})

	t.Run("retry on status mismatch", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", ExpectedStatus: 200, Retry: 1, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 2 {
				return testutil.MockResponse(503, ""), nil
			}
			return testutil.MockResponse(200, ""), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status)
		assert.Equal(t, 2, attempts)
	})

	t.Run("no retry message on single attempt", func(t *testing.T) {
		c := Check{URL: "http://localhost/health", Retry: 0, Client: clientErr("connection refused")}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.NotContains(t, strings.Join(result.Details, " "), "attempts")
	})

	t.Run("exhausted on status mismatch", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", ExpectedStatus: 200, Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			return testutil.MockResponse(503, ""), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Equal(t, 3, attempts)
		details := strings.Join(result.Details, " ")
		assert.Contains(t, details, "3 attempts")
		assert.Contains(t, details, "503")
	})

	t.Run("success includes attempt info", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return testutil.MockResponse(503, ""), nil
			}
			return testutil.MockResponse(200, ""), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status)
		assert.Contains(t, strings.Join(result.Details, " "), "attempt 3 of 3")
	})
}

func TestHTTPCheckContainsRetry(t *testing.T) {
	t.Run("succeeds on second attempt", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", Contains: "healthy", Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 2 {
				return testutil.MockResponse(200, `{"status": "starting"}`), nil
			}
			return testutil.MockResponse(200, `{"status": "healthy"}`), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status)
		assert.Equal(t, 2, attempts)
	})

	t.Run("exhausted", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/health", Contains: "healthy", Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			return testutil.MockResponse(200, `{"status": "starting"}`), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Equal(t, 3, attempts)
		assert.Contains(t, strings.Join(result.Details, " "), "3 attempts")
	})
}

func TestHTTPCheckJSONPathRetry(t *testing.T) {
	t.Run("succeeds on second attempt", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/api", JSONPath: "status=ready", Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 2 {
				return testutil.MockResponse(200, `{"status": "starting"}`), nil
			}
			return testutil.MockResponse(200, `{"status": "ready"}`), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status)
		assert.Equal(t, 2, attempts)
	})

	t.Run("exhausted on value mismatch", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/api", JSONPath: "status=ready", Retry: 2, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			return testutil.MockResponse(200, `{"status": "starting"}`), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Equal(t, 3, attempts)
		assert.Contains(t, strings.Join(result.Details, " "), "3 attempts")
	})

	t.Run("exhausted on path not found", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://localhost/api", JSONPath: "data.missing", Retry: 1, RetryDelay: time.Millisecond, Client: &testutil.MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			return testutil.MockResponse(200, `{"data": {"id": 123}}`), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Equal(t, 2, attempts)
		assert.Contains(t, strings.Join(result.Details, " "), "2 attempts")
	})
}

func TestRealHTTPClient(t *testing.T) {
	t.Run("basic request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		defer ts.Close()

		client := &RealHTTPClient{Timeout: 5 * time.Second}
		req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)
	})

	t.Run("insecure TLS", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		client := &RealHTTPClient{Timeout: 5 * time.Second, Insecure: true}
		req, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)
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

		client := &RealHTTPClient{Timeout: 5 * time.Second, FollowRedirects: false}
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/redirect", http.NoBody)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 302, resp.StatusCode)
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

		client := &RealHTTPClient{Timeout: 5 * time.Second, FollowRedirects: true}
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/redirect", http.NoBody)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)
	})
}

func TestRealFileReader(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "testfile")
	content := []byte("test content")
	require.NoError(t, os.WriteFile(tmpFile, content, 0o600))

	reader := &RealFileReader{}
	data, err := reader.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.True(t, bytes.Equal(data, content))
}
