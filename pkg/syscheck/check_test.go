package syscheck

import (
	"strings"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

type mockSysInfo struct {
	os   string
	arch string
}

func (m *mockSysInfo) OS() string   { return m.os }
func (m *mockSysInfo) Arch() string { return m.arch }

func TestSysCheck(t *testing.T) {
	tests := []struct {
		name          string
		check         Check
		wantStatus    check.Status
		wantDetailSub string
	}{
		{
			name: "OS matches",
			check: Check{
				ExpectedOS: "linux",
				Info:       &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "OS mismatch fails",
			check: Check{
				ExpectedOS: "linux",
				Info:       &mockSysInfo{os: "darwin", arch: "amd64"},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "OS mismatch",
		},
		{
			name: "arch matches",
			check: Check{
				ExpectedArch: "amd64",
				Info:         &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "arch mismatch fails",
			check: Check{
				ExpectedArch: "arm64",
				Info:         &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "arch mismatch",
		},
		{
			name: "both OS and arch match",
			check: Check{
				ExpectedOS:   "linux",
				ExpectedArch: "arm64",
				Info:         &mockSysInfo{os: "linux", arch: "arm64"},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "OS matches but arch mismatch",
			check: Check{
				ExpectedOS:   "linux",
				ExpectedArch: "arm64",
				Info:         &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "arch mismatch",
		},
		{
			name: "arch matches but OS mismatch",
			check: Check{
				ExpectedOS:   "linux",
				ExpectedArch: "amd64",
				Info:         &mockSysInfo{os: "darwin", arch: "amd64"},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "OS mismatch",
		},
		{
			name: "no flags is error",
			check: Check{
				Info: &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantStatus:    check.StatusFail,
			wantDetailSub: "at least one of --os or --arch is required",
		},
		{
			name: "windows OS",
			check: Check{
				ExpectedOS: "windows",
				Info:       &mockSysInfo{os: "windows", arch: "amd64"},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "darwin OS",
			check: Check{
				ExpectedOS: "darwin",
				Info:       &mockSysInfo{os: "darwin", arch: "arm64"},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "386 arch",
			check: Check{
				ExpectedArch: "386",
				Info:         &mockSysInfo{os: "linux", arch: "386"},
			},
			wantStatus: check.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v (details: %v)", result.Status, tt.wantStatus, result.Details)
			}
			if tt.wantDetailSub != "" {
				allDetails := strings.Join(result.Details, " ")
				if !strings.Contains(allDetails, tt.wantDetailSub) {
					t.Errorf("Details %v should contain %q", result.Details, tt.wantDetailSub)
				}
			}
		})
	}
}

func TestSysCheckResultName(t *testing.T) {
	tests := []struct {
		name     string
		check    Check
		wantName string
	}{
		{
			name: "OS only",
			check: Check{
				ExpectedOS: "linux",
				Info:       &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantName: "sys: os=linux",
		},
		{
			name: "arch only",
			check: Check{
				ExpectedArch: "amd64",
				Info:         &mockSysInfo{os: "linux", arch: "amd64"},
			},
			wantName: "sys: arch=amd64",
		},
		{
			name: "both OS and arch",
			check: Check{
				ExpectedOS:   "linux",
				ExpectedArch: "arm64",
				Info:         &mockSysInfo{os: "linux", arch: "arm64"},
			},
			wantName: "sys: os=linux arch=arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
		})
	}
}
