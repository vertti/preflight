package tcpcheck

import (
	"net"
	"time"

	"github.com/vertti/preflight/pkg/check"
)

// TCPDialer abstracts network dialing for testability.
type TCPDialer interface {
	DialTimeout(network, address string, timeout time.Duration) (net.Conn, error)
}

// RealTCPDialer uses the real net package.
type RealTCPDialer struct{}

// DialTimeout dials the network address with a timeout.
func (d *RealTCPDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout(network, address, timeout)
}

// Check verifies TCP connectivity to a host:port.
type Check struct {
	Address string        // host:port to connect to
	Timeout time.Duration // connection timeout (default 5s)
	Dialer  TCPDialer     // injected for testing
}

// Run executes the TCP connectivity check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "tcp: " + c.Address,
	}

	timeout := c.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	conn, err := c.Dialer.DialTimeout("tcp", c.Address, timeout)
	if err != nil {
		return result.Failf("connection failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	result.Status = check.StatusOK
	result.AddDetailf("connected to %s", c.Address)
	return result
}
