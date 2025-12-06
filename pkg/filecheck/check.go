package filecheck

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies that a file or directory meets requirements.
type Check struct {
	Path          string     // path to check
	ExpectDir     bool       // --dir: expect a directory
	ExpectSocket  bool       // --socket: expect a Unix socket
	ExpectSymlink bool       // --symlink: expect a symbolic link
	SymlinkTarget string     // --symlink-target: expected symlink target
	Writable      bool       // --writable: check write permission
	Executable    bool       // --executable: check execute permission
	NotEmpty      bool       // --not-empty: file must have size > 0
	MinSize       int64      // --min-size: minimum file size in bytes
	MaxSize       int64      // --max-size: maximum file size in bytes (0 = no limit)
	Match         string     // --match: regex pattern for content
	Contains      string     // --contains: literal string to search
	Head          int64      // --head: limit content read to first N bytes
	Mode          string     // --mode: minimum permissions (octal, e.g., "0644")
	ModeExact     string     // --mode-exact: exact permissions required
	Owner         int        // --owner: expected owner UID (-1 = don't check)
	FS            FileSystem // injected for testing
}

// Run executes the file check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("file: %s", c.Path),
	}

	// Use Lstat for symlink checks (doesn't follow symlinks), Stat otherwise
	var info fs.FileInfo
	var err error
	if c.ExpectSymlink {
		info, err = c.FS.Lstat(c.Path)
	} else {
		info, err = c.FS.Stat(c.Path)
	}
	if err != nil {
		switch {
		case os.IsNotExist(err):
			return result.Fail("not found", err)
		case os.IsPermission(err):
			return result.Fail("permission denied", err)
		default:
			return result.Failf("stat failed: %v", err)
		}
	}

	// Type check: --dir, --socket, --symlink flags
	if err := c.checkTypeConstraint(info, &result); err != nil {
		return result
	}

	// Size details and checks (only for files)
	if !info.IsDir() {
		if err := c.checkSizeConstraints(info, &result); err != nil {
			return result
		}
	}

	// Permission details
	result.AddDetailf("permissions: %s", info.Mode().Perm())

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
			return result.Fail("not writable", fmt.Errorf("file is not writable"))
		}
	}

	// --executable
	if c.Executable {
		if !isExecutable(info.Mode()) {
			return result.Fail("not executable", fmt.Errorf("file is not executable"))
		}
	}

	// --owner: check file ownership
	if c.Owner >= 0 {
		if err := c.checkOwner(&result); err != nil {
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
		result.Failf("invalid mode: %v", err)
		return err
	}

	actual := mode.Perm()
	// Check if actual permissions include at least the required permissions
	if actual&required != required {
		err := fmt.Errorf("permissions %s do not include minimum %s", actual, required)
		result.Fail(fmt.Sprintf("permissions %s do not include minimum %s", actual, required), err)
		return err
	}
	return nil
}

func (c *Check) checkModeExact(mode fs.FileMode, result *check.Result) error {
	required, err := parseOctalMode(c.ModeExact)
	if err != nil {
		result.Failf("invalid mode: %v", err)
		return err
	}

	actual := mode.Perm()
	if actual != required {
		err := fmt.Errorf("permissions %s do not match required %s", actual, required)
		result.Fail(fmt.Sprintf("permissions %s != required %s", actual, required), err)
		return err
	}
	return nil
}

func (c *Check) checkTypeConstraint(info fs.FileInfo, result *check.Result) error {
	mode := info.Mode()

	if c.ExpectSymlink {
		if !isSymlink(mode) {
			err := fmt.Errorf("expected symlink, got file/directory")
			result.Fail("expected symlink, got file/directory", err)
			return err
		}
		result.AddDetail("type: symlink")
		// Check symlink target if specified
		if c.SymlinkTarget != "" {
			return c.checkSymlinkTarget(result)
		}
		return nil
	}

	if c.ExpectSocket {
		if !isSocket(mode) {
			err := fmt.Errorf("expected socket, got file/directory")
			result.Fail("expected socket, got file/directory", err)
			return err
		}
		result.AddDetail("type: socket")
		return nil
	}

	if c.ExpectDir {
		if !info.IsDir() {
			err := fmt.Errorf("expected directory, got file")
			result.Fail("expected directory, got file", err)
			return err
		}
		result.AddDetail("type: directory")
	} else {
		switch {
		case isSymlink(mode):
			result.AddDetail("type: symlink")
		case isSocket(mode):
			result.AddDetail("type: socket")
		case info.IsDir():
			result.AddDetail("type: directory")
		default:
			result.AddDetail("type: file")
		}
	}
	return nil
}

func (c *Check) checkSizeConstraints(info fs.FileInfo, result *check.Result) error {
	result.AddDetailf("size: %d", info.Size())

	// --not-empty
	if c.NotEmpty && info.Size() == 0 {
		err := fmt.Errorf("file is empty")
		result.Fail("file is empty", err)
		return err
	}

	// --min-size
	if c.MinSize > 0 && info.Size() < c.MinSize {
		err := fmt.Errorf("file size %d below minimum %d", info.Size(), c.MinSize)
		result.Fail(fmt.Sprintf("size %d < minimum %d", info.Size(), c.MinSize), err)
		return err
	}

	// --max-size
	if c.MaxSize > 0 && info.Size() > c.MaxSize {
		err := fmt.Errorf("file size %d above maximum %d", info.Size(), c.MaxSize)
		result.Fail(fmt.Sprintf("size %d > maximum %d", info.Size(), c.MaxSize), err)
		return err
	}

	return nil
}

func (c *Check) checkContent(result *check.Result) error {
	content, err := c.FS.ReadFile(c.Path, c.Head)
	if err != nil {
		result.Failf("failed to read file: %v", err)
		return err
	}

	// --contains: literal string search
	if c.Contains != "" {
		if !strings.Contains(string(content), c.Contains) {
			err := fmt.Errorf("content does not contain %q", c.Contains)
			result.Fail(fmt.Sprintf("content does not contain %q", c.Contains), err)
			return err
		}
	}

	// --match: regex pattern
	if c.Match != "" {
		re, err := check.CompileRegex(c.Match)
		if err != nil {
			result.Failf("invalid regex pattern: %v", err)
			return err
		}
		if !re.Match(content) {
			err := fmt.Errorf("content does not match pattern %q", c.Match)
			result.Fail(fmt.Sprintf("content does not match pattern %q", c.Match), err)
			return err
		}
	}

	return nil
}

func (c *Check) checkOwner(result *check.Result) error {
	uid, _, err := c.FS.GetOwner(c.Path)
	if err != nil {
		result.Failf("failed to get owner: %v", err)
		return err
	}

	result.AddDetailf("owner: %d", uid)

	if int(uid) != c.Owner {
		err := fmt.Errorf("owner %d != expected %d", uid, c.Owner)
		result.Fail(fmt.Sprintf("owner %d != expected %d", uid, c.Owner), err)
		return err
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

// isSocket checks if the mode indicates a Unix socket
func isSocket(mode fs.FileMode) bool {
	return mode&fs.ModeSocket != 0
}

// isSymlink checks if the mode indicates a symbolic link
func isSymlink(mode fs.FileMode) bool {
	return mode&fs.ModeSymlink != 0
}

func (c *Check) checkSymlinkTarget(result *check.Result) error {
	target, err := c.FS.Readlink(c.Path)
	if err != nil {
		result.Failf("failed to read symlink target: %v", err)
		return err
	}

	result.AddDetailf("target: %s", target)

	if target != c.SymlinkTarget {
		err := fmt.Errorf("symlink target %s != expected %s", target, c.SymlinkTarget)
		result.Fail(fmt.Sprintf("symlink target %s != expected %s", target, c.SymlinkTarget), err)
		return err
	}

	return nil
}
