package cmdcheck

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCmdRunner struct {
	LookPathFunc          func(file string) (string, error)
	RunCommandContextFunc func(ctx context.Context, name string, args ...string) (string, string, error)
}

func (m *mockCmdRunner) LookPath(file string) (string, error) {
	return m.LookPathFunc(file)
}

func (m *mockCmdRunner) RunCommandContext(ctx context.Context, name string, args ...string) (stdout, stderr string, err error) {
	return m.RunCommandContextFunc(ctx, name, args...)
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
	require.NoError(t, err)
	assert.Equal(t, "/usr/bin/node", path)

	_, err = mock.LookPath("nonexistent")
	assert.Error(t, err)
}

func TestMockCmdRunner_RunCommandContext(t *testing.T) {
	mock := &mockCmdRunner{
		RunCommandContextFunc: func(_ context.Context, name string, args ...string) (string, string, error) {
			if name == "node" && len(args) == 1 && args[0] == "--version" {
				return "v18.17.0", "", nil
			}
			return "", "command failed", errors.New("exit 1")
		},
	}

	ctx := context.Background()
	stdout, stderr, err := mock.RunCommandContext(ctx, "node", "--version")
	require.NoError(t, err)
	assert.Equal(t, "v18.17.0", stdout)
	assert.Empty(t, stderr)

	_, stderr, err = mock.RunCommandContext(ctx, "bad")
	assert.Error(t, err)
	assert.Equal(t, "command failed", stderr)
}
