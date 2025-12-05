package cmdcheck

import (
	"errors"
	"testing"
)

type mockCmdRunner struct {
	LookPathFunc   func(file string) (string, error)
	RunCommandFunc func(name string, args ...string) (string, string, error)
}

func (m *mockCmdRunner) LookPath(file string) (string, error) {
	return m.LookPathFunc(file)
}

func (m *mockCmdRunner) RunCommand(name string, args ...string) (stdout, stderr string, err error) {
	return m.RunCommandFunc(name, args...)
}

func TestMockCmdRunner_LookPath(t *testing.T) {
	mock := &mockCmdRunner{
		LookPathFunc: func(file string) (string, error) {
			if file == "node" {
				return "/usr/bin/node", nil
			}
			return "", errors.New("not found")
		},
	}

	path, err := mock.LookPath("node")
	if err != nil {
		t.Fatalf("LookPath(node) error = %v", err)
	}
	if path != "/usr/bin/node" {
		t.Errorf("LookPath(node) = %q, want %q", path, "/usr/bin/node")
	}

	_, err = mock.LookPath("nonexistent")
	if err == nil {
		t.Error("LookPath(nonexistent) error = nil, want error")
	}
}

func TestMockCmdRunner_RunCommand(t *testing.T) {
	mock := &mockCmdRunner{
		RunCommandFunc: func(name string, args ...string) (string, string, error) {
			if name == "node" && len(args) == 1 && args[0] == "--version" {
				return "v18.17.0", "", nil
			}
			return "", "command failed", errors.New("exit 1")
		},
	}

	stdout, stderr, err := mock.RunCommand("node", "--version")
	if err != nil {
		t.Fatalf("RunCommand error = %v", err)
	}
	if stdout != "v18.17.0" {
		t.Errorf("stdout = %q, want %q", stdout, "v18.17.0")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}

	_, stderr, err = mock.RunCommand("bad")
	if err == nil {
		t.Error("RunCommand(bad) error = nil, want error")
	}
	if stderr != "command failed" {
		t.Errorf("stderr = %q, want %q", stderr, "command failed")
	}
}
