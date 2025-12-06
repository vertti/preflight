package cmdcheck

import (
	"context"
	"fmt"
	"time"

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
	Timeout      time.Duration    // timeout for version command (default: 30s)
	Runner       CmdRunner        // injected for testing
}

// Run executes the command check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("cmd: %s", c.Name),
	}

	path, err := c.Runner.LookPath(c.Name)
	if err != nil {
		return result.Failf("not found in PATH: %v", err)
	}

	result.AddDetailf("path: %s", path)

	args := c.VersionArgs
	if len(args) == 0 {
		args = []string{"--version"}
	}

	// Set up timeout context
	timeout := c.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	stdout, stderr, err := c.Runner.RunCommandContext(ctx, c.Name, args...)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return result.Failf("version command timed out after %s", timeout)
		}
		result.AddDetailf("version command failed: %v", err)
		if stderr != "" {
			result.AddDetailf("stderr: %s", stderr)
		}
		result.Status = check.StatusFail
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
			result.AddDetailf("version: %s", versionOutput)
		}
	}

	result.Status = check.StatusOK
	return result
}

func (c *Check) checkMatchPattern(output string, result *check.Result) error {
	re, err := check.CompileRegex(c.MatchPattern)
	if err != nil {
		result.Failf("invalid regex pattern: %v", err)
		return err
	}
	if !re.MatchString(output) {
		err := fmt.Errorf("version output does not match pattern %q", c.MatchPattern)
		result.Fail(fmt.Sprintf("version output %q does not match pattern %q", output, c.MatchPattern), err)
		return err
	}
	result.AddDetailf("version: %s", output)
	return nil
}

func (c *Check) checkVersionConstraints(output string, result *check.Result) error {
	parsedVersion, err := version.Extract(output)
	if err != nil {
		result.Failf("could not parse version from output: %v", err)
		return err
	}

	result.AddDetailf("version: %s", parsedVersion)

	// Check exact version
	if c.ExactVersion != nil && parsedVersion.Compare(*c.ExactVersion) != 0 {
		err := fmt.Errorf("version %s does not match required %s", parsedVersion, c.ExactVersion)
		result.Fail(fmt.Sprintf("version %s != required %s", parsedVersion, c.ExactVersion), err)
		return err
	}

	// Check min version (inclusive: version >= min)
	if c.MinVersion != nil && !parsedVersion.GreaterThanOrEqual(*c.MinVersion) {
		err := fmt.Errorf("version %s below minimum %s", parsedVersion, c.MinVersion)
		result.Fail(fmt.Sprintf("version %s < minimum %s", parsedVersion, c.MinVersion), err)
		return err
	}

	// Check max version (exclusive: version < max)
	if c.MaxVersion != nil && !parsedVersion.LessThan(*c.MaxVersion) {
		err := fmt.Errorf("version %s at or above maximum %s", parsedVersion, c.MaxVersion)
		result.Fail(fmt.Sprintf("version %s >= maximum %s", parsedVersion, c.MaxVersion), err)
		return err
	}

	return nil
}
