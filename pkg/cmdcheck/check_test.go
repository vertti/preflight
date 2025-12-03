package cmdcheck

import (
	"errors"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func TestCommandCheck_NotFound(t *testing.T) {
	runner := &MockRunner{
		LookPathFunc: func(file string) (string, error) {
			return "", errors.New("executable file not found in $PATH")
		},
	}

	c := &Check{
		Name:   "nonexistent",
		Runner: runner,
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
	if result.Name != "cmd:nonexistent" {
		t.Errorf("Name = %q, want %q", result.Name, "cmd:nonexistent")
	}
}

func TestCommandCheck_VersionFails(t *testing.T) {
	runner := &MockRunner{
		LookPathFunc: func(file string) (string, error) {
			return "/usr/bin/broken", nil
		},
		RunCommandFunc: func(name string, args ...string) (string, string, error) {
			return "", "error loading shared library", errors.New("exit 1")
		},
	}

	c := &Check{
		Name:   "broken",
		Runner: runner,
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
	if result.Err == nil {
		t.Error("Err = nil, want error")
	}
}

func TestCommandCheck_Found(t *testing.T) {
	runner := &MockRunner{
		LookPathFunc: func(file string) (string, error) {
			return "/usr/bin/node", nil
		},
		RunCommandFunc: func(name string, args ...string) (string, string, error) {
			return "v18.17.0", "", nil
		},
	}

	c := &Check{
		Name:   "node",
		Runner: runner,
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusOK)
	}
	if result.Name != "cmd:node" {
		t.Errorf("Name = %q, want %q", result.Name, "cmd:node")
	}
}
