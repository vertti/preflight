package cmdcheck

import (
	"fmt"
	"regexp"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/version"
)

// Check verifies that a command exists and can run.
type Check struct {
	Name         string           // command name to check
	VersionArgs  []string         // args to get version (default: --version)
	MinVersion   *version.Version // minimum version required (inclusive)
	MaxVersion   *version.Version // maximum version allowed (exclusive)
	ExactVersion *version.Version // exact version required
	MatchPattern string           // regex pattern to match against version output
	Runner       Runner           // injected for testing
}

// Run executes the command check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("cmd: %s", c.Name),
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

	versionOutput := stdout
	if versionOutput == "" {
		versionOutput = stderr
	}

	switch {
	case c.MatchPattern != "":
		if err := c.checkMatchPattern(versionOutput, &result); err != nil {
			return result
		}
	case c.MinVersion != nil || c.MaxVersion != nil || c.ExactVersion != nil:
		if err := c.checkVersionConstraints(versionOutput, &result); err != nil {
			return result
		}
	default:
		if versionOutput != "" {
			result.Details = append(result.Details, fmt.Sprintf("version: %s", versionOutput))
		}
	}

	result.Status = check.StatusOK
	return result
}

func (c *Check) checkMatchPattern(output string, result *check.Result) error {
	re, err := regexp.Compile(c.MatchPattern)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("invalid regex pattern: %v", err))
		result.Err = err
		return err
	}
	if !re.MatchString(output) {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("version output %q does not match pattern %q", output, c.MatchPattern))
		result.Err = fmt.Errorf("version output does not match pattern %q", c.MatchPattern)
		return result.Err
	}
	result.Details = append(result.Details, fmt.Sprintf("version: %s", output))
	return nil
}

func (c *Check) checkVersionConstraints(output string, result *check.Result) error {
	parsedVersion, err := version.Extract(output)
	if err != nil {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("could not parse version from output: %v", err))
		result.Err = err
		return err
	}

	result.Details = append(result.Details, fmt.Sprintf("version: %s", parsedVersion))

	// Check exact version
	if c.ExactVersion != nil && parsedVersion.Compare(*c.ExactVersion) != 0 {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("version %s != required %s", parsedVersion, c.ExactVersion))
		result.Err = fmt.Errorf("version %s does not match required %s", parsedVersion, c.ExactVersion)
		return result.Err
	}

	// Check min version (inclusive: version >= min)
	if c.MinVersion != nil && !parsedVersion.GreaterThanOrEqual(*c.MinVersion) {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("version %s < minimum %s", parsedVersion, c.MinVersion))
		result.Err = fmt.Errorf("version %s below minimum %s", parsedVersion, c.MinVersion)
		return result.Err
	}

	// Check max version (exclusive: version < max)
	if c.MaxVersion != nil && !parsedVersion.LessThan(*c.MaxVersion) {
		result.Status = check.StatusFail
		result.Details = append(result.Details, fmt.Sprintf("version %s >= maximum %s", parsedVersion, c.MaxVersion))
		result.Err = fmt.Errorf("version %s at or above maximum %s", parsedVersion, c.MaxVersion)
		return result.Err
	}

	return nil
}
