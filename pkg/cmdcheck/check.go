package cmdcheck

import (
	"fmt"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/version"
)

// Check verifies that a command exists and can run.
type Check struct {
	Name        string           // command name to check
	VersionArgs []string         // args to get version (default: --version)
	MinVersion  *version.Version // minimum version required (inclusive)
	MaxVersion  *version.Version // maximum version allowed (exclusive)
	Runner      Runner           // injected for testing
}

// Run executes the command check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("cmd:%s", c.Name),
	}

	path, err := c.Runner.LookPath(c.Name)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("not found in PATH: %v", err))
		result.Err = err
		return result
	}

	result.Details = append(result.Details, fmt.Sprintf("path: %s", path))

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

	if c.MinVersion != nil || c.MaxVersion != nil {
		// Combine stdout and stderr for version extraction
		versionOutput := stdout
		if versionOutput == "" {
			versionOutput = stderr
		}

		parsedVersion, err := version.Extract(versionOutput)
		if err != nil {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("could not parse version from output: %v", err))
			result.Err = err
			return result
		}

		result.Details = append(result.Details, fmt.Sprintf("version: %s", parsedVersion))

		// Check min version (inclusive: version >= min)
		if c.MinVersion != nil && !parsedVersion.GreaterThanOrEqual(*c.MinVersion) {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("version %s < minimum %s", parsedVersion, c.MinVersion))
			result.Err = fmt.Errorf("version %s below minimum %s", parsedVersion, c.MinVersion)
			return result
		}

		// Check max version (exclusive: version < max)
		if c.MaxVersion != nil && !parsedVersion.LessThan(*c.MaxVersion) {
			result.Status = check.StatusFail
			result.Details = append(result.Details, fmt.Sprintf("version %s >= maximum %s", parsedVersion, c.MaxVersion))
			result.Err = fmt.Errorf("version %s at or above maximum %s", parsedVersion, c.MaxVersion)
			return result
		}
	} else if stdout != "" {
		result.Details = append(result.Details, fmt.Sprintf("version: %s", stdout))
	}

	result.Status = check.StatusOK
	return result
}
