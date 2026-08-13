package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReal(t *testing.T) {
	t.Run("basic request", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		defer ts.Close()

		client := &Real{Timeout: 5 * time.Second}
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

		client := &Real{Timeout: 5 * time.Second, Insecure: true}
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

		client := &Real{Timeout: 5 * time.Second, FollowRedirects: false}
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

		client := &Real{Timeout: 5 * time.Second, FollowRedirects: true}
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/redirect", http.NoBody)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		assert.Equal(t, 200, resp.StatusCode)
	})
}

// Go's transport transparently decompresses gzip, so a small response can
// expand without limit once read. A 2 MB gzip bomb took preflight to 6.5 GB
// resident, which is an OOM kill on the memory-limited containers it is built
// to run in.
func TestReadBody(t *testing.T) {
	t.Run("reads a body that fits", func(t *testing.T) {
		body, err := ReadBody(strings.NewReader("hello"))

		require.NoError(t, err)
		assert.Equal(t, "hello", body)
	})

	t.Run("reads a body exactly at the limit", func(t *testing.T) {
		body, err := ReadBody(strings.NewReader(strings.Repeat("x", MaxBodySize)))

		require.NoError(t, err)
		assert.Len(t, body, MaxBodySize)
	})

	t.Run("refuses a body over the limit rather than truncating it", func(t *testing.T) {
		_, err := ReadBody(strings.NewReader(strings.Repeat("x", MaxBodySize+1)))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "too large")
	})

	// The point is to never hold the whole thing, so an endless body has to stop
	// the read rather than fill memory first.
	t.Run("stops on an endless body", func(t *testing.T) {
		endless := endlessReader{}

		_, err := ReadBody(endless)

		require.Error(t, err)
	})

	t.Run("surfaces a read failure", func(t *testing.T) {
		_, err := ReadBody(iotest.ErrReader(errors.New("connection reset")))

		require.ErrorContains(t, err, "connection reset")
	})
}

type endlessReader struct{}

func (endlessReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'x'
	}
	return len(p), nil
}
