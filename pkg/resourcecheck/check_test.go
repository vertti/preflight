package resourcecheck

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

// Lives here rather than in resource_test.go, which is //go:build unix: the
// mock has no platform dependency, and tagging it broke `GOOS=windows go vet`
// for every test in this package.
type mockResourceChecker struct {
	freeDiskSpace   uint64
	freeDiskErr     error
	availableMemory uint64
	availableMemErr error
	availableCPUs   float64
}

func (m *mockResourceChecker) FreeDiskSpace(path string) (uint64, error) {
	return m.freeDiskSpace, m.freeDiskErr
}

func (m *mockResourceChecker) AvailableMemory() (uint64, error) {
	return m.availableMemory, m.availableMemErr
}

func (m *mockResourceChecker) AvailableCPUs() float64 {
	return m.availableCPUs
}

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
		{"cpu check passes", Check{MinCPUs: 2, Checker: &mockResourceChecker{availableCPUs: 8}}, check.StatusOK},
		{"cpu check fails - not enough", Check{MinCPUs: 16, Checker: &mockResourceChecker{availableCPUs: 8}}, check.StatusFail},
		{"cpu check exact match", Check{MinCPUs: 4, Checker: &mockResourceChecker{availableCPUs: 4}}, check.StatusOK},
		{"all checks pass", Check{MinDisk: 10 * GB, MinMemory: 2 * GB, MinCPUs: 2, Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB, availableCPUs: 8}}, check.StatusOK},
		{"disk passes but memory fails", Check{MinDisk: 10 * GB, MinMemory: 32 * GB, Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB}}, check.StatusFail},
		{"disk and memory pass but cpu fails", Check{MinDisk: 10 * GB, MinMemory: 2 * GB, MinCPUs: 16, Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB, availableCPUs: 8}}, check.StatusFail},
		{"no checks specified", Check{Checker: &mockResourceChecker{freeDiskSpace: 50 * GB, availableMemory: 16 * GB, availableCPUs: 8}}, check.StatusOK},

		// A CPU quota buys a share of wall time, not whole cores, so a container
		// given 1.5 CPUs does not satisfy a demand for 2.
		{"cpu check fails on a fractional shortfall", Check{MinCPUs: 2, Checker: &mockResourceChecker{availableCPUs: 1.5}}, check.StatusFail},
		{"cpu check passes when the fraction clears the bar", Check{MinCPUs: 1, Checker: &mockResourceChecker{availableCPUs: 1.5}}, check.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			assert.Equal(t, "resource", result.Name)
		})
	}
}

func TestResourceCheck_ReportsFractionalCPUs(t *testing.T) {
	result := (&Check{MinCPUs: 1, Checker: &mockResourceChecker{availableCPUs: 1.5}}).Run()

	assert.Contains(t, result.Details, "cpus: 1.5")
}

func TestResourceCheck_ReportsWholeCPUsWithoutADecimalPoint(t *testing.T) {
	result := (&Check{MinCPUs: 1, Checker: &mockResourceChecker{availableCPUs: 8}}).Run()

	assert.Contains(t, result.Details, "cpus: 8")
}

// runtime.NumCPU() reflects CPU affinity but not a CFS quota, so the two have
// to be combined rather than trusted individually.
func TestAvailableCPUs(t *testing.T) {
	quota := func(n float64, limited bool) func() (float64, bool) {
		return func() (float64, bool) { return n, limited }
	}

	tests := []struct {
		name     string
		affinity int
		quota    func() (float64, bool)
		want     float64
	}{
		{"no quota leaves affinity alone", 14, quota(0, false), 14},
		{"a quota below affinity wins", 14, quota(1.5, true), 1.5},
		{"a quota above affinity does not inflate it", 2, quota(8, true), 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.InDelta(t, tt.want, availableCPUs(tt.affinity, tt.quota), 0.0001)
		})
	}
}
