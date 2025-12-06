package syscheck

import (
	"fmt"
	"runtime"

	"github.com/vertti/preflight/pkg/check"
)

// SysInfo abstracts system information for testability.
type SysInfo interface {
	OS() string
	Arch() string
}

// RealSysInfo returns actual system information.
type RealSysInfo struct{}

func (r *RealSysInfo) OS() string   { return runtime.GOOS }
func (r *RealSysInfo) Arch() string { return runtime.GOARCH }

// Check verifies the system matches expected OS and/or architecture.
type Check struct {
	ExpectedOS   string  // required OS (linux, darwin, windows)
	ExpectedArch string  // required architecture (amd64, arm64, 386)
	Info         SysInfo // injected for testing
}

func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "sys",
	}

	if c.ExpectedOS == "" && c.ExpectedArch == "" {
		return result.Failf("at least one of --os or --arch is required")
	}

	info := c.Info
	if info == nil {
		info = &RealSysInfo{}
	}

	actualOS := info.OS()
	actualArch := info.Arch()

	if c.ExpectedOS != "" && actualOS != c.ExpectedOS {
		return result.Failf("OS mismatch: expected %s, got %s", c.ExpectedOS, actualOS)
	}

	if c.ExpectedArch != "" && actualArch != c.ExpectedArch {
		return result.Failf("arch mismatch: expected %s, got %s", c.ExpectedArch, actualArch)
	}

	result.Status = check.StatusOK
	result.AddDetailf("os: %s", actualOS)
	result.AddDetailf("arch: %s", actualArch)

	if c.ExpectedOS != "" {
		result.Name = "sys: os=" + c.ExpectedOS
	}
	if c.ExpectedArch != "" {
		if c.ExpectedOS != "" {
			result.Name = fmt.Sprintf("sys: os=%s arch=%s", c.ExpectedOS, c.ExpectedArch)
		} else {
			result.Name = "sys: arch=" + c.ExpectedArch
		}
	}

	return result
}
