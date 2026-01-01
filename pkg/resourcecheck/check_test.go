package resourcecheck

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

func TestResourceCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
	}{
		{"disk check passes", Check{MinDisk: 10 * GB, Checker: &mockResourceChecker{freeDiskSpace: 20 * GB}}, check.StatusOK},
		{"disk check fails - not enough space", Check{MinDisk: 20 * GB, Checker: &mockResourceChecker{freeDiskSpace: 10 * GB}}, check.StatusFail},
		{"disk check fails - error", Check{MinDisk: 10 * GB, Checker: &mockResourceChecker{freeDiskErr: errors.New("disk error")}}, check.StatusFail},
		{"disk check with custom path", Check{MinDisk: 10 * GB, Path: "/var/lib/docker", Checker: &mockResourceChecker{freeDiskSpace: 20 * GB}}, check.StatusOK},
		{"memory check passes", Check{MinMemory: 2 * GB, Checker: &mockResourceChecker{availableMemory: 8 * GB}}, check.StatusOK},
		{"memory check fails - not enough", Check{MinMemory: 16 * GB, Checker: &mockResourceChecker{availableMemory: 8 * GB}}, check.StatusFail},
		{"memory check fails - error", Check{MinMemory: 2 * GB, Checker: &mockResourceChecker{availableMemErr: errors.New("memory error")}}, check.StatusFail},
		{"cpu check passes", Check{MinCPUs: 2, Checker: &mockResourceChecker{numCPUs: 8}}, check.StatusOK},
		{"cpu check fails - not enough", Check{MinCPUs: 16, Checker: &mockResourceChecker{numCPUs: 8}}, check.StatusFail},
		{"cpu check exact match", Check{MinCPUs: 4, Checker: &mockResourceChecker{numCPUs: 4}}, check.StatusOK},
		{"all checks pass", Check{MinDisk: 10 * GB, MinMemory: 2 * GB, MinCPUs: 2, Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB, numCPUs: 8}}, check.StatusOK},
		{"disk passes but memory fails", Check{MinDisk: 10 * GB, MinMemory: 32 * GB, Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB}}, check.StatusFail},
		{"disk and memory pass but cpu fails", Check{MinDisk: 10 * GB, MinMemory: 2 * GB, MinCPUs: 16, Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB, numCPUs: 8}}, check.StatusFail},
		{"no checks specified", Check{Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB, numCPUs: 8}}, check.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			assert.Equal(t, "resource", result.Name)
		})
	}
}
