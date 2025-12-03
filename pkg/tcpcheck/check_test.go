package tcpcheck

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/vertti/preflight/pkg/check"
)

// MockDialer is a mock implementation of Dialer for testing.
type MockDialer struct {
	DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)
}

func (m *MockDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	return m.DialFunc(network, address, timeout)
}

// MockConn is a minimal net.Conn implementation for testing.
type MockConn struct{}

func (m *MockConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (m *MockConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (m *MockConn) Close() error                       { return nil }
func (m *MockConn) LocalAddr() net.Addr                { return nil }
func (m *MockConn) RemoteAddr() net.Addr               { return nil }
func (m *MockConn) SetDeadline(t time.Time) error      { return nil }
func (m *MockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *MockConn) SetWriteDeadline(t time.Time) error { return nil }

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
			dialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return &MockConn{}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "tcp:localhost:5432",
		},
		{
			name:    "connection refused",
			address: "localhost:9999",
			dialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, errors.New("connection refused")
			},
			wantStatus: check.StatusFail,
			wantName:   "tcp:localhost:9999",
		},
		{
			name:    "timeout",
			address: "10.255.255.1:80",
			timeout: 1 * time.Second,
			dialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, errors.New("i/o timeout")
			},
			wantStatus: check.StatusFail,
			wantName:   "tcp:10.255.255.1:80",
		},
		{
			name:    "dns resolution failure",
			address: "nonexistent.invalid:80",
			dialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, errors.New("no such host")
			},
			wantStatus: check.StatusFail,
			wantName:   "tcp:nonexistent.invalid:80",
		},
		{
			name:    "custom timeout used",
			address: "localhost:8080",
			timeout: 10 * time.Second,
			dialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				if timeout != 10*time.Second {
					t.Errorf("expected timeout 10s, got %v", timeout)
				}
				return &MockConn{}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "tcp:localhost:8080",
		},
		{
			name:    "default timeout when zero",
			address: "localhost:3000",
			timeout: 0,
			dialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				if timeout != 5*time.Second {
					t.Errorf("expected default timeout 5s, got %v", timeout)
				}
				return &MockConn{}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "tcp:localhost:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Check{
				Address: tt.address,
				Timeout: tt.timeout,
				Dialer: &MockDialer{
					DialFunc: tt.dialFunc,
				},
			}

			result := c.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			if tt.wantStatus == check.StatusFail && result.Err == nil {
				t.Error("expected Err to be set on failure")
			}
		})
	}
}
