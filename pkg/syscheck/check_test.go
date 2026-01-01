package syscheck

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

type mockSysInfo struct {
	os   string
	arch string
}

func (m *mockSysInfo) OS() string   { return m.os }
func (m *mockSysInfo) Arch() string { return m.arch }

func TestSysCheck(t *testing.T) {
	linux := &mockSysInfo{os: "linux", arch: "amd64"}
	darwin := &mockSysInfo{os: "darwin", arch: "amd64"}
	linuxArm := &mockSysInfo{os: "linux", arch: "arm64"}

	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantDetailSub string
	}{
		{"OS matches", Check{ExpectedOS: "linux", Info: linux}, check.StatusOK, ""},
		{"OS mismatch fails", Check{ExpectedOS: "linux", Info: darwin}, check.StatusFail, "OS mismatch"},
		{"arch matches", Check{ExpectedArch: "amd64", Info: linux}, check.StatusOK, ""},
		{"arch mismatch fails", Check{ExpectedArch: "arm64", Info: linux}, check.StatusFail, "arch mismatch"},
		{"both OS and arch match", Check{ExpectedOS: "linux", ExpectedArch: "arm64", Info: linuxArm}, check.StatusOK, ""},
		{"OS matches but arch mismatch", Check{ExpectedOS: "linux", ExpectedArch: "arm64", Info: linux}, check.StatusFail, "arch mismatch"},
		{"arch matches but OS mismatch", Check{ExpectedOS: "linux", ExpectedArch: "amd64", Info: darwin}, check.StatusFail, "OS mismatch"},
		{"no flags is error", Check{Info: linux}, check.StatusFail, "at least one of --os or --arch is required"},
		{"windows OS", Check{ExpectedOS: "windows", Info: &mockSysInfo{os: "windows", arch: "amd64"}}, check.StatusOK, ""},
		{"darwin OS", Check{ExpectedOS: "darwin", Info: &mockSysInfo{os: "darwin", arch: "arm64"}}, check.StatusOK, ""},
		{"386 arch", Check{ExpectedArch: "386", Info: &mockSysInfo{os: "linux", arch: "386"}}, check.StatusOK, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			if tt.wantDetailSub != "" {
				assert.Contains(t, strings.Join(result.Details, " "), tt.wantDetailSub)
			}
		})
	}
}

func TestSysCheckResultName(t *testing.T) {
	tests := []struct {
		check    Check
		wantName string
	}{
		{Check{ExpectedOS: "linux", Info: &mockSysInfo{os: "linux"}}, "sys: os=linux"},
		{Check{ExpectedArch: "amd64", Info: &mockSysInfo{arch: "amd64"}}, "sys: arch=amd64"},
		{Check{ExpectedOS: "linux", ExpectedArch: "arm64", Info: &mockSysInfo{os: "linux", arch: "arm64"}}, "sys: os=linux arch=arm64"},
	}

	for _, tt := range tests {
		t.Run(tt.wantName, func(t *testing.T) {
			assert.Equal(t, tt.wantName, tt.check.Run().Name)
		})
	}
}

func TestRealSysInfo(t *testing.T) {
	info := &RealSysInfo{}
	assert.NotEmpty(t, info.OS())
	assert.NotEmpty(t, info.Arch())
}
