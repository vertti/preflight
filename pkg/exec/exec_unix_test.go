//go:build unix

package exec

import (
	"errors"
	"testing"
)

func TestRealExecutor_Exec_Success(t *testing.T) {
	// Save original and restore after test
	originalExecFunc := execFunc
	defer func() { execFunc = originalExecFunc }()

	var capturedBinary string
	var capturedArgv []string
	var capturedEnv []string

	// Mock execFunc to capture arguments
	execFunc = func(binary string, argv []string, env []string) error {
		capturedBinary = binary
		capturedArgv = argv
		capturedEnv = env
		return nil
	}

	e := &RealExecutor{}
	err := e.Exec("echo", []string{"hello", "world"})

	if err != nil {
		t.Errorf("Exec() error = %v, want nil", err)
	}

	// Verify binary path was resolved (should be absolute path to echo)
	if capturedBinary == "" {
		t.Error("expected binary path to be set")
	}

	// Verify argv[0] is the command name
	if len(capturedArgv) < 1 || capturedArgv[0] != "echo" {
		t.Errorf("argv[0] = %v, want 'echo'", capturedArgv)
	}

	// Verify arguments were passed
	if len(capturedArgv) != 3 || capturedArgv[1] != "hello" || capturedArgv[2] != "world" {
		t.Errorf("argv = %v, want ['echo', 'hello', 'world']", capturedArgv)
	}

	// Verify environment was passed
	if len(capturedEnv) == 0 {
		t.Error("expected environment to be passed")
	}
}

func TestRealExecutor_Exec_ExecFuncError(t *testing.T) {
	// Save original and restore after test
	originalExecFunc := execFunc
	defer func() { execFunc = originalExecFunc }()

	expectedErr := errors.New("exec failed")
	execFunc = func(binary string, argv []string, env []string) error {
		return expectedErr
	}

	e := &RealExecutor{}
	err := e.Exec("echo", []string{})

	if !errors.Is(err, expectedErr) {
		t.Errorf("Exec() error = %v, want %v", err, expectedErr)
	}
}

func TestRealExecutor_Exec_EmptyArgs(t *testing.T) {
	// Save original and restore after test
	originalExecFunc := execFunc
	defer func() { execFunc = originalExecFunc }()

	var capturedArgv []string
	execFunc = func(binary string, argv []string, env []string) error {
		capturedArgv = argv
		return nil
	}

	e := &RealExecutor{}
	err := e.Exec("echo", []string{})

	if err != nil {
		t.Errorf("Exec() error = %v, want nil", err)
	}

	// argv should just be ["echo"] when no args provided
	if len(capturedArgv) != 1 || capturedArgv[0] != "echo" {
		t.Errorf("argv = %v, want ['echo']", capturedArgv)
	}
}

func TestRealExecutor_Exec_BinaryPathResolution(t *testing.T) {
	// Save original and restore after test
	originalExecFunc := execFunc
	defer func() { execFunc = originalExecFunc }()

	var capturedBinary string
	execFunc = func(binary string, argv []string, env []string) error {
		capturedBinary = binary
		return nil
	}

	e := &RealExecutor{}
	_ = e.Exec("sh", []string{"-c", "true"})

	// Binary should be resolved to absolute path
	if capturedBinary == "sh" {
		t.Error("expected binary to be resolved to absolute path")
	}
	if capturedBinary == "" {
		t.Error("expected binary path to be set")
	}
}
