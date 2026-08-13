// Package httpclient provides the HTTP client shared by the checks that make
// requests. It exists because httpcheck and promcheck each had their own copy:
// the copies drifted, promcheck's never gained redirect protection, and it
// forwarded custom auth headers to whatever host a 3xx named.
package httpclient

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client abstracts HTTP requests for testability.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// MaxBodySize caps how much of a response body a check will read. Health
// endpoints and Prometheus queries answer in kilobytes, so this is generous
// for anything preflight is pointed at on purpose.
const MaxBodySize = 10 << 20 // 10 MiB

// ReadBody reads a response body, refusing one larger than MaxBodySize.
//
// The size of the response on the wire says nothing about the size in memory:
// Go's transport transparently decompresses gzip, so a 2 MB body can expand to
// gigabytes. Reading it whole would OOM the memory-limited containers preflight
// is built to run in. Oversize bodies fail rather than truncate, so a --contains
// miss is never really a body that ran off the end.
func ReadBody(r io.Reader) (string, error) {
	body, err := io.ReadAll(io.LimitReader(r, MaxBodySize+1))
	if err != nil {
		return "", err
	}
	if len(body) > MaxBodySize {
		return "", fmt.Errorf("response body too large (over %d bytes)", MaxBodySize)
	}
	return string(body), nil
}

// Real is a Client backed by net/http.
type Real struct {
	Timeout         time.Duration
	Insecure        bool
	FollowRedirects bool
}

// Do executes an HTTP request.
//
// Redirects are not followed unless FollowRedirects is set. Go strips
// Authorization and Cookie when a redirect crosses hosts, but not custom
// headers — and tenancy headers like X-Scope-OrgID are custom. Following a
// redirect would also let the destination decide the check's verdict.
func (c *Real) Do(req *http.Request) (*http.Response, error) {
	transport := &http.Transport{}
	if c.Insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // intentional for --insecure flag
	}

	client := &http.Client{
		Timeout:   c.Timeout,
		Transport: transport,
	}

	if !c.FollowRedirects {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client.Do(req)
}
