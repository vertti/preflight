package preflightfile

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
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

func TestFindFile_StopAtGit(t *testing.T) {
	tmpDir := t.TempDir()

	projectDir := filepath.Join(tmpDir, "project")
	gitDir := filepath.Join(projectDir, ".git")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}

	preflightPath := filepath.Join(tmpDir, ".preflight")
	if err := os.WriteFile(preflightPath, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	projectPreflight := filepath.Join(projectDir, ".preflight")
	if err := os.WriteFile(projectPreflight, []byte("test"), 0o600); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	found, err := FindFile(projectDir, "")
	if err != nil {
		t.Fatalf("FindFile failed: %v", err)
	}
	if found != projectPreflight {
		t.Errorf("expected %q, got %q", projectPreflight, found)
	}
}

func TestFindFile_StopAtHome(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home directory: %v", err)
	}

	testDir := filepath.Join(homeDir, "test_preflight")
	if err := os.MkdirAll(testDir, 0o700); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(testDir); err != nil {
			t.Errorf("failed to clean up test directory: %v", err)
		}
	}()

	_, err = FindFile(testDir, "")
	if err == nil {
		t.Error("expected error when .preflight not found")
	}
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
