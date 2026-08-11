package preflightfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindFile_ExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	preflightPath := filepath.Join(tmpDir, ".preflight")
	if err := os.WriteFile(preflightPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	found, err := FindFile(tmpDir, preflightPath)
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}
	if found != preflightPath {
		t.Errorf("expected %q, got %q", preflightPath, found)
	}

	_, err = FindFile(tmpDir, filepath.Join(tmpDir, "nonexistent"))
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFindFile_TraverseUp(t *testing.T) {
	tmpDir := t.TempDir()

	subdir1 := filepath.Join(tmpDir, "subdir1")
	subdir2 := filepath.Join(subdir1, "subdir2")
	if err := os.MkdirAll(subdir2, 0o700); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	preflightPath := filepath.Join(tmpDir, ".preflight")
	if err := os.WriteFile(preflightPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	found, err := FindFile(subdir2, "")
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}
	if found != preflightPath {
		t.Errorf("expected %q, got %q", preflightPath, found)
	}
}

// A repo root is a boundary: a .preflight above it belongs to something else.
// The .preflight must live only in the parent, or FindFile returns on its first
// iteration and the .git logic is never reached — which is what the previous
// version of this test did, so it passed with that logic deleted.
func TestFindFile_StopAtGit(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(filepath.Join(projectDir, ".git"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".preflight"), []byte("env HOME\n"), 0o600))

	_, err := FindFile(projectDir, "")
	require.Error(t, err, ".git is a boundary: the parent's .preflight is not ours")
}

// The mirror image: without a .git boundary the parent's file is found, which
// pins that the error above comes from the boundary and not from the walk
// failing generally.
func TestFindFile_WalksUpWithoutGitBoundary(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	require.NoError(t, os.MkdirAll(projectDir, 0o700))
	parentPreflight := filepath.Join(tmpDir, ".preflight")
	require.NoError(t, os.WriteFile(parentPreflight, []byte("env HOME\n"), 0o600))

	found, err := FindFile(projectDir, "")
	require.NoError(t, err)
	require.Equal(t, parentPreflight, found)
}

// The search stops at $HOME. This uses a temporary HOME rather than the real
// one: the previous version created and RemoveAll'd ~/test_preflight on the
// developer's machine, and failed outright for anyone who has a global
// ~/.preflight — which is a supported location this package exists to find.
func TestFindFile_StopAtHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows

	testDir := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(testDir, 0o700))

	_, err := FindFile(testDir, "")
	require.Error(t, err, "no .preflight anywhere up to HOME")
}

func TestFindFile_StopsBeforeLeavingHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	// A .preflight above HOME must not be picked up.
	require.NoError(t, os.WriteFile(filepath.Join(filepath.Dir(home), ".preflight"), []byte("env HOME\n"), 0o600))
	t.Cleanup(func() { _ = os.Remove(filepath.Join(filepath.Dir(home), ".preflight")) })

	testDir := filepath.Join(home, "project")
	require.NoError(t, os.MkdirAll(testDir, 0o700))

	_, err := FindFile(testDir, "")
	require.Error(t, err, "search must stop at HOME, not walk past it")
}

func TestParseFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name: "basic commands",
			content: `file /models/bert.onnx --not-empty
cmd myapp
cmd java
preflight env PATH`,
			expected: []string{
				"preflight file /models/bert.onnx --not-empty",
				"preflight cmd myapp",
				"preflight cmd java",
				"preflight env PATH",
			},
		},
		{
			name: "with comments and empty lines",
			content: `# This is a comment
file /models/bert.onnx --not-empty

# Another comment
cmd myapp
preflight env PATH
`,
			expected: []string{
				"preflight file /models/bert.onnx --not-empty",
				"preflight cmd myapp",
				"preflight env PATH",
			},
		},
		{
			name: "all lines already have preflight",
			content: `preflight file /models/bert.onnx --not-empty
preflight cmd myapp
preflight env PATH`,
			expected: []string{
				"preflight file /models/bert.onnx --not-empty",
				"preflight cmd myapp",
				"preflight env PATH",
			},
		},
		{
			name:     "empty file",
			content:  ``,
			expected: []string{},
		},
		{
			name: "only comments and empty lines",
			content: `# Comment 1

# Comment 2
`,
			expected: []string{},
		},
		{
			name: "inline comment handling",
			content: `file /models/bert.onnx --not-empty # inline comment
cmd myapp`,
			expected: []string{
				"preflight file /models/bert.onnx --not-empty # inline comment",
				"preflight cmd myapp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			preflightPath := filepath.Join(tmpDir, ".preflight")
			if err := os.WriteFile(preflightPath, []byte(tt.content), 0o600); err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}

			commands, err := ParseFile(preflightPath)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			if !reflect.DeepEqual(commands, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, commands)
			}
		})
	}
}

func TestParseFile_Nonexistent(t *testing.T) {
	_, err := ParseFile("/nonexistent/file")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestFindFile_ReachFilesystemRoot(t *testing.T) {
	// Create a deep temp directory structure outside home dir
	// TempDir is typically in /tmp or similar, outside home
	tmpDir := t.TempDir()
	deepPath := filepath.Join(tmpDir, "a", "b", "c", "d")
	if err := os.MkdirAll(deepPath, 0o700); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	// No .preflight and no .git anywhere in this hierarchy
	// The search should traverse up until it hits filesystem root
	_, err := FindFile(deepPath, "")
	if err == nil {
		t.Error("expected error when .preflight not found")
	}
}
