package tcpcheck

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

type mockTCPDialer struct {
	DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (m *mockTCPDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return m.DialFunc(network, address, timeout)
}

type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *mockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestTCPCheck(t *testing.T) {
	tests := []struct {
		name       string
		address    string
		timeout    time.Duration
		dialFunc   func(network, address string, timeout time.Duration) (net.Conn, error)
		wantStatus check.Status
		wantName   string
	}{
		{
			name:    "successful connection",
			address: "localhost:5432",
			dialFunc: func(_, _ string, _ time.Duration) (net.Conn, error) {
				return &mockConn{}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "tcp: localhost:5432",
		},
		{
			name:    "connection refused",
			address: "localhost:9999",
			dialFunc: func(_, _ string, _ time.Duration) (net.Conn, error) {
				return nil, errors.New("connection refused")
			},
			wantStatus: check.StatusFail,
			wantName:   "tcp: localhost:9999",
		},
		{
			name:    "timeout",
			address: "10.255.255.1:80",
			timeout: 1 * time.Second,
			dialFunc: func(_, _ string, _ time.Duration) (net.Conn, error) {
				return nil, errors.New("i/o timeout")
			},
			wantStatus: check.StatusFail,
			wantName:   "tcp: 10.255.255.1:80",
		},
		{
			name:    "dns resolution failure",
			address: "nonexistent.invalid:80",
			dialFunc: func(_, _ string, _ time.Duration) (net.Conn, error) {
				return nil, errors.New("no such host")
			},
			wantStatus: check.StatusFail,
			wantName:   "tcp: nonexistent.invalid:80",
		},
		{
			name:    "custom timeout used",
			address: "localhost:8080",
			timeout: 10 * time.Second,
			dialFunc: func(_, _ string, timeout time.Duration) (net.Conn, error) {
				assert.Equal(t, 10*time.Second, timeout)
				return &mockConn{}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "tcp: localhost:8080",
		},
		{
			name:    "default timeout when zero",
			address: "localhost:3000",
			timeout: 0,
			dialFunc: func(_, _ string, timeout time.Duration) (net.Conn, error) {
				assert.Equal(t, 5*time.Second, timeout)
				return &mockConn{}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "tcp: localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Check{
				Address: tt.address,
				Timeout: tt.timeout,
				Dialer:  &mockTCPDialer{DialFunc: tt.dialFunc},
			}

			result := c.Run()

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, tt.wantName, result.Name)
			if tt.wantStatus == check.StatusFail {
				assert.NotNil(t, result.Err)
			}
		})
	}
}
