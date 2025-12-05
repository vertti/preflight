package resourcecheck

import (
	"errors"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func TestResourceCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
	}{
		// Disk checks
		{
			name: "disk check passes",
			check: Check{
				MinDisk: 10 * GB,
				Checker: &mockResourceChecker{
					freeDiskSpace: 20 * GB,
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "disk check fails - not enough space",
			check: Check{
				MinDisk: 20 * GB,
				Checker: &mockResourceChecker{
					freeDiskSpace: 10 * GB,
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "disk check fails - error",
			check: Check{
				MinDisk: 10 * GB,
				Checker: &mockResourceChecker{
					freeDiskErr: errors.New("disk error"),
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "disk check with custom path",
			check: Check{
				MinDisk: 10 * GB,
				Path:    "/var/lib/docker",
				Checker: &mockResourceChecker{
					freeDiskSpace: 20 * GB,
				},
			},
			wantStatus: check.StatusOK,
		},

		// Memory checks
		{
			name: "memory check passes",
			check: Check{
				MinMemory: 2 * GB,
				Checker: &mockResourceChecker{
					availableMemory: 8 * GB,
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "memory check fails - not enough memory",
			check: Check{
				MinMemory: 16 * GB,
				Checker: &mockResourceChecker{
					availableMemory: 8 * GB,
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "memory check fails - error",
			check: Check{
				MinMemory: 2 * GB,
				Checker: &mockResourceChecker{
					availableMemErr: errors.New("memory error"),
				},
			},
			wantStatus: check.StatusFail,
		},

		// CPU checks
		{
			name: "cpu check passes",
			check: Check{
				MinCPUs: 2,
				Checker: &mockResourceChecker{
					numCPUs: 8,
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "cpu check fails - not enough cpus",
			check: Check{
				MinCPUs: 16,
				Checker: &mockResourceChecker{
					numCPUs: 8,
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "cpu check exact match",
			check: Check{
				MinCPUs: 4,
				Checker: &mockResourceChecker{
					numCPUs: 4,
				},
			},
			wantStatus: check.StatusOK,
		},

		// Combined checks
		{
			name: "all checks pass",
			check: Check{
				MinDisk:   10 * GB,
				MinMemory: 2 * GB,
				MinCPUs:   2,
				Checker: &mockResourceChecker{
					freeDiskSpace:   50 * GB,
					availableMemory: 16 * GB,
					numCPUs:         8,
				},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "disk passes but memory fails",
			check: Check{
				MinDisk:   10 * GB,
				MinMemory: 32 * GB,
				Checker: &mockResourceChecker{
					freeDiskSpace:   50 * GB,
					availableMemory: 16 * GB,
				},
			},
			wantStatus: check.StatusFail,
		},
		{
			name: "disk and memory pass but cpu fails",
			check: Check{
				MinDisk:   10 * GB,
				MinMemory: 2 * GB,
				MinCPUs:   16,
				Checker: &mockResourceChecker{
					freeDiskSpace:   50 * GB,
					availableMemory: 16 * GB,
					numCPUs:         8,
				},
			},
			wantStatus: check.StatusFail,
		},

		// No checks specified (should pass)
		{
			name: "no checks specified",
			check: Check{
				Checker: &mockResourceChecker{
					freeDiskSpace:   50 * GB,
					availableMemory: 16 * GB,
					numCPUs:         8,
				},
			},
			wantStatus: check.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v (details: %v)", result.Status, tt.wantStatus, result.Details)
			}

			if result.Name != "resource" {
				t.Errorf("name = %q, want %q", result.Name, "resource")
			}
		})
	}
}
