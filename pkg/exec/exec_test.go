package exec

import (
	"errors"
	"os"
	"strings"
	"testing"
)

// MockExecutor is a test implementation of Executor.
type MockExecutor struct {
	ExecFunc func(name string, args []string) error
}

func (m *MockExecutor) Exec(name string, args []string) error {
	if m.ExecFunc != nil {
		return m.ExecFunc(name, args)
	}
	return nil
}

func TestExecutorInterface(t *testing.T) {
	// Verify MockExecutor implements Executor interface
	var _ Executor = &MockExecutor{}
	var _ Executor = &RealExecutor{}
}

func TestMockExecutor(t *testing.T) {
	tests := []struct {
		name     string
		execFunc func(string, []string) error
		wantErr  bool
	}{
		{
			name: "successful exec",
			execFunc: func(name string, args []string) error {
				return nil
			},
			wantErr: false,
		},
		{
			name: "exec returns error",
			execFunc: func(name string, args []string) error {
				return errors.New("exec failed")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MockExecutor{ExecFunc: tt.execFunc}
			err := m.Exec("test", []string{"arg1", "arg2"})
			if (err != nil) != tt.wantErr {
				t.Errorf("Exec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockExecutor_NilFunc(t *testing.T) {
	m := &MockExecutor{}
	err := m.Exec("test", []string{"arg1"})
	if err != nil {
		t.Errorf("expected nil error when ExecFunc is nil, got %v", err)
	}
}

func TestRealExecutor_CommandNotFound(t *testing.T) {
	e := &RealExecutor{}
	err := e.Exec("nonexistent-command-that-does-not-exist-12345", []string{})
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestLookPath(t *testing.T) {
	// Test with a command that should exist on all systems
	path, err := lookPath("echo")
	if err != nil {
		t.Skipf("echo not found in PATH, skipping: %v", err)
	}
	if path == "" {
		t.Error("expected non-empty path for echo")
	}
}

func TestLookPath_NotFound(t *testing.T) {
	_, err := lookPath("nonexistent-command-xyz-12345")
	if err == nil {
		t.Error("expected error for nonexistent command")
	}
}

func TestEnviron(t *testing.T) {
	env := environ()
	if len(env) == 0 {
		t.Error("expected non-empty environment")
	}

	// Check that PATH is in the environment
	hasPath := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			hasPath = true
			break
		}
	}
	if !hasPath && os.Getenv("PATH") != "" {
		t.Error("expected PATH in environment")
	}
}
