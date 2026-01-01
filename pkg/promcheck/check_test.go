package promcheck

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

type mockHTTPClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}

func mockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

const (
	promSuccessVector   = `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"__name__":"up","job":"myapp"},"value":[1702000000,"1"]}]}}`
	promSuccessScalar   = `{"status":"success","data":{"resultType":"scalar","result":[1702000000,"42.5"]}}`
	promEmptyResult     = `{"status":"success","data":{"resultType":"vector","result":[]}}`
	promMultipleResults = `{"status":"success","data":{"resultType":"vector","result":[{"metric":{"job":"a"},"value":[1702000000,"1"]},{"metric":{"job":"b"},"value":[1702000000,"2"]}]}}`
	promError           = `{"status":"error","errorType":"bad_data","error":"invalid expression"}`
	promValueFloat      = `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1702000000,"0.05"]}]}}`
	promValueHigh       = `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1702000000,"100"]}]}}`
	promValueLow        = `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1702000000,"5"]}]}}`
)

func ptr(f float64) *float64 { return &f }

func resp(body string) func(*http.Request) (*http.Response, error) {
	return func(*http.Request) (*http.Response, error) { return mockResponse(200, body), nil }
}

func respStatus(code int) func(*http.Request) (*http.Response, error) {
	return func(*http.Request) (*http.Response, error) { return mockResponse(code, ""), nil }
}

func respErr(msg string) func(*http.Request) (*http.Response, error) {
	return func(*http.Request) (*http.Response, error) { return nil, errors.New(msg) }
}

func TestPrometheusCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantDetailSub string
	}{
		// Validation
		{"missing URL", Check{Query: "up"}, check.StatusFail, "URL is required"},
		{"missing query", Check{URL: "http://prom:9090"}, check.StatusFail, "query is required"},
		{"invalid URL", Check{URL: "not-a-url", Query: "up"}, check.StatusFail, "invalid URL"},

		// Successful queries
		{"vector query", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}, check.StatusOK, "value: 1"},
		{"scalar query", Check{URL: "http://prom:9090", Query: "42.5", Client: &mockHTTPClient{DoFunc: resp(promSuccessScalar)}}, check.StatusOK, "value: 42.5"},
		{"query shown", Check{URL: "http://prom:9090", Query: "error_rate", Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}, check.StatusOK, "query: error_rate"},

		// Errors
		{"connection error", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: respErr("connection refused")}}, check.StatusFail, "connection refused"},
		{"non-200 status", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: respStatus(500)}}, check.StatusFail, "prometheus returned status 500"},
		{"prom error response", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(promError)}}, check.StatusFail, "invalid expression"},
		{"empty result", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(promEmptyResult)}}, check.StatusFail, "returned no data"},
		{"multiple results", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(promMultipleResults)}}, check.StatusFail, "returned 2 results"},

		// Thresholds
		{"exact match passes", Check{URL: "http://prom:9090", Query: "up", Exact: ptr(1), Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}, check.StatusOK, ""},
		{"exact match fails", Check{URL: "http://prom:9090", Query: "up", Exact: ptr(0), Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}, check.StatusFail, "does not equal 0"},
		{"min passes", Check{URL: "http://prom:9090", Query: "up", Min: ptr(50), Client: &mockHTTPClient{DoFunc: resp(promValueHigh)}}, check.StatusOK, ""},
		{"min fails", Check{URL: "http://prom:9090", Query: "up", Min: ptr(50), Client: &mockHTTPClient{DoFunc: resp(promValueLow)}}, check.StatusFail, "< minimum 50"},
		{"max passes", Check{URL: "http://prom:9090", Query: "up", Max: ptr(0.1), Client: &mockHTTPClient{DoFunc: resp(promValueFloat)}}, check.StatusOK, ""},
		{"max fails", Check{URL: "http://prom:9090", Query: "up", Max: ptr(0.01), Client: &mockHTTPClient{DoFunc: resp(promValueFloat)}}, check.StatusFail, "> maximum 0.01"},

		// Headers and URL
		{"custom headers", Check{URL: "http://prom:9090", Query: "up", Headers: map[string]string{"Authorization": "Bearer token123"}, Client: &mockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Equal(t, "Bearer token123", req.Header.Get("Authorization"))
			return mockResponse(200, promSuccessVector), nil
		}}}, check.StatusOK, ""},
		{"query URL encoded", Check{URL: "http://prom:9090", Query: `up{job="myapp"}`, Client: &mockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
			assert.Contains(t, req.URL.RawQuery, "up%7Bjob%3D%22myapp%22%7D")
			return mockResponse(200, promSuccessVector), nil
		}}}, check.StatusOK, ""},

		// Edge cases
		{"unsupported result type", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(`{"status":"success","data":{"resultType":"matrix","result":[]}}`)}}, check.StatusFail, "unsupported result type"},
		{"non-numeric value", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1702000000,"not-a-number"]}]}}`)}}, check.StatusFail, "failed to parse metric value"},
		{"prom status no error msg", Check{URL: "http://prom:9090", Query: "up", Client: &mockHTTPClient{DoFunc: resp(`{"status":"error"}`)}}, check.StatusFail, "prometheus returned status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			if tt.wantDetailSub != "" {
				assert.True(t, containsDetail(result.Details, tt.wantDetailSub), "details %v should contain %q", result.Details, tt.wantDetailSub)
			}
		})
	}
}

func TestPrometheusCheckRetry(t *testing.T) {
	t.Run("succeeds after retries", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://prom:9090", Query: "up", Retry: 2, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			if attempts < 3 {
				return nil, errors.New("connection refused")
			}
			return mockResponse(200, promSuccessVector), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusOK, result.Status)
		assert.Equal(t, 3, attempts)
	})

	t.Run("fails after max retries", func(t *testing.T) {
		attempts := 0
		c := Check{URL: "http://prom:9090", Query: "up", Retry: 2, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
			attempts++
			return mockResponse(500, ""), nil
		}}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.Equal(t, 3, attempts)
		assert.True(t, containsDetail(result.Details, "after 3 attempts"))
	})

	t.Run("connection failure exhausts retries", func(t *testing.T) {
		c := Check{URL: "http://prom:9090", Query: "up", Retry: 1, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: respErr("connection refused")}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.True(t, containsDetail(result.Details, "after 2 attempts"))
	})

	t.Run("empty result exhausts retries", func(t *testing.T) {
		c := Check{URL: "http://prom:9090", Query: "up", Retry: 1, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: resp(promEmptyResult)}}
		result := c.Run()
		assert.Equal(t, check.StatusFail, result.Status)
		assert.True(t, containsDetail(result.Details, "after 2 attempts"))
	})

	t.Run("threshold failures exhaust retries", func(t *testing.T) {
		for _, tc := range []struct {
			name  string
			check Check
		}{
			{"exact", Check{URL: "http://prom:9090", Query: "up", Exact: ptr(0), Retry: 1, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}},
			{"min", Check{URL: "http://prom:9090", Query: "up", Min: ptr(10), Retry: 1, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}},
			{"max", Check{URL: "http://prom:9090", Query: "up", Max: ptr(0.5), Retry: 1, RetryDelay: 1, Client: &mockHTTPClient{DoFunc: resp(promSuccessVector)}}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.check.Run()
				assert.Equal(t, check.StatusFail, result.Status)
				assert.True(t, containsDetail(result.Details, "after 2 attempts"))
			})
		}
	})
}

func containsDetail(details []string, substr string) bool {
	for _, d := range details {
		if strings.Contains(d, substr) {
			return true
		}
	}
	return false
}
