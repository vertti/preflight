// Package httpclient provides the HTTP client shared by the checks that make
// requests. It exists because httpcheck and promcheck each had their own copy:
// the copies drifted, promcheck's never gained redirect protection, and it
// forwarded custom auth headers to whatever host a 3xx named.
package httpclient

import (
	"crypto/tls"
	"net/http"
	"time"
)

// Client abstracts HTTP requests for testability.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
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
