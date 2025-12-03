package cmdcheck

import (
	"fmt"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies that a command exists and can run.
type Check struct {
	Name        string   // command name to check
	VersionArgs []string // args to get version (default: --version)
	Runner      Runner   // injected for testing
}

// Run executes the command check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("cmd:%s", c.Name),
	}

	// Look up the command in PATH
	path, err := c.Runner.LookPath(c.Name)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("not found in PATH: %v", err))
		result.Err = err
		return result
	}

	result.Details = append(result.Details, fmt.Sprintf("path: %s", path))

	// Run version command
	args := c.VersionArgs
	if len(args) == 0 {
		args = []string{"--version"}
	}

	stdout, stderr, err := c.Runner.RunCommand(c.Name, args...)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("version command failed: %v", err))
		if stderr != "" {
			result.Details = append(result.Details, fmt.Sprintf("stderr: %s", stderr))
		}
		result.Err = err
		return result
	}

	// Success
	result.Status = check.StatusOK
	if stdout != "" {
		result.Details = append(result.Details, fmt.Sprintf("version: %s", stdout))
	}

	return result
}
