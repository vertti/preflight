package filecheck

import (
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies that a file or directory meets requirements.
type Check struct {
	Path       string     // path to check
	ExpectDir  bool       // --dir: expect a directory
	Writable   bool       // --writable: check write permission
	Executable bool       // --executable: check execute permission
	NotEmpty   bool       // --not-empty: file must have size > 0
	MinSize    int64      // --min-size: minimum file size in bytes
	MaxSize    int64      // --max-size: maximum file size in bytes (0 = no limit)
	Match      string     // --match: regex pattern for content
	Contains   string     // --contains: literal string to search
	Head       int64      // --head: limit content read to first N bytes
	Mode       string     // --mode: minimum permissions (octal, e.g., "0644")
	ModeExact  string     // --mode-exact: exact permissions required
	FS         FileSystem // injected for testing
}

// Run executes the file check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("file:%s", c.Path),
	}

	info, err := c.FS.Stat(c.Path)
	if err != nil {
		result.Status = check.StatusFail
		switch {
		case os.IsNotExist(err):
			result.Details = append(result.Details, "not found")
		case os.IsPermission(err):
			result.Details = append(result.Details, "permission denied")
		default:
			result.Details = append(result.Details, fmt.Sprintf("stat failed: %v", err))
		}
		result.Err = err
		return result
	}

	// Type check: --dir flag
	if c.ExpectDir {
		if !info.IsDir() {
			result.Status = check.StatusFail
			result.Details = append(result.Details, "expected directory, got file")
			result.Err = fmt.Errorf("expected directory, got file")
			return result
		}
		result.Details = append(result.Details, "type: directory")
	} else {
		if info.IsDir() {
			result.Details = append(result.Details, "type: directory")
		} else {
			result.Details = append(result.Details, "type: file")
		}
	}

	// Size details and checks (only for files)
	if !info.IsDir() {
		result.Details = append(result.Details, fmt.Sprintf("size: %d", info.Size()))

		// --not-empty
		if c.NotEmpty && info.Size() == 0 {
			result.Status = check.StatusFail
			result.Details = append(result.Details, "file is empty")
			result.Err = fmt.Errorf("file is empty")
			return result
		}

		// --min-size
		if c.MinSize > 0 && info.Size() < c.MinSize {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("size %d < minimum %d", info.Size(), c.MinSize))
			result.Err = fmt.Errorf("file size %d below minimum %d", info.Size(), c.MinSize)
			return result
		}

		// --max-size
		if c.MaxSize > 0 && info.Size() > c.MaxSize {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("size %d > maximum %d", info.Size(), c.MaxSize))
			result.Err = fmt.Errorf("file size %d above maximum %d", info.Size(), c.MaxSize)
			return result
		}
	}

	// Permission details
	result.Details = append(result.Details, fmt.Sprintf("permissions: %s", info.Mode().Perm()))

	// --mode: minimum permissions check
	if c.Mode != "" {
		if err := c.checkModeMinimum(info.Mode(), &result); err != nil {
			return result
		}
	}

	// --mode-exact: exact permissions check
	if c.ModeExact != "" {
		if err := c.checkModeExact(info.Mode(), &result); err != nil {
			return result
		}
	}

	// --writable
	if c.Writable {
		if !isWritable(info.Mode()) {
			result.Status = check.StatusFail
			result.Details = append(result.Details, "not writable")
			result.Err = fmt.Errorf("file is not writable")
			return result
		}
	}

	// --executable
	if c.Executable {
		if !isExecutable(info.Mode()) {
			result.Status = check.StatusFail
			result.Details = append(result.Details, "not executable")
			result.Err = fmt.Errorf("file is not executable")
			return result
		}
	}

	// Content checks (only for files, not directories)
	if !info.IsDir() && (c.Contains != "" || c.Match != "") {
		if err := c.checkContent(&result); err != nil {
			return result
		}
	}

	result.Status = check.StatusOK
	return result
}

func (c *Check) checkModeMinimum(mode fs.FileMode, result *check.Result) error {
	required, err := parseOctalMode(c.Mode)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("invalid mode: %v", err))
		result.Err = err
		return err
	}

	actual := mode.Perm()
	// Check if actual permissions include at least the required permissions
	if actual&required != required {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("permissions %s do not include minimum %s", actual, required))
		result.Err = fmt.Errorf("permissions %s do not include minimum %s", actual, required)
		return result.Err
	}
	return nil
}

func (c *Check) checkModeExact(mode fs.FileMode, result *check.Result) error {
	required, err := parseOctalMode(c.ModeExact)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("invalid mode: %v", err))
		result.Err = err
		return err
	}

	actual := mode.Perm()
	if actual != required {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("permissions %s != required %s", actual, required))
		result.Err = fmt.Errorf("permissions %s do not match required %s", actual, required)
		return result.Err
	}
	return nil
}

func (c *Check) checkContent(result *check.Result) error {
	content, err := c.FS.ReadFile(c.Path, c.Head)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("failed to read file: %v", err))
		result.Err = err
		return err
	}

	// --contains: literal string search
	if c.Contains != "" {
		if !strings.Contains(string(content), c.Contains) {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("content does not contain %q", c.Contains))
			result.Err = fmt.Errorf("content does not contain %q", c.Contains)
			return result.Err
		}
	}

	// --match: regex pattern
	if c.Match != "" {
		re, err := regexp.Compile(c.Match)
		if err != nil {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("invalid regex pattern: %v", err))
			result.Err = err
			return err
		}
		if !re.Match(content) {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("content does not match pattern %q", c.Match))
			result.Err = fmt.Errorf("content does not match pattern %q", c.Match)
			return result.Err
		}
	}

	return nil
}

// parseOctalMode parses an octal permission string like "0644" or "644"
func parseOctalMode(s string) (fs.FileMode, error) {
	var mode uint32
	_, err := fmt.Sscanf(s, "%o", &mode)
	if err != nil {
		return 0, fmt.Errorf("invalid octal mode %q: %w", s, err)
	}
	return fs.FileMode(mode), nil
}

// isWritable checks if the mode has any write bit set (owner, group, or other)
func isWritable(mode fs.FileMode) bool {
	return mode&0o222 != 0
}

// isExecutable checks if the mode has any execute bit set (owner, group, or other)
func isExecutable(mode fs.FileMode) bool {
	return mode&0o111 != 0
}
