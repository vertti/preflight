//go:build unix

package resourcecheck

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealResourceChecker_FreeDiskSpace(t *testing.T) {
	r := &RealResourceChecker{}

	space, err := r.FreeDiskSpace(".")
	require.NoError(t, err)
	assert.Positive(t, space)

	_, err = r.FreeDiskSpace("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestRealResourceChecker_FreeDiskSpace_TempDir(t *testing.T) {
	r := &RealResourceChecker{}

	tmpDir := t.TempDir()
	space, err := r.FreeDiskSpace(tmpDir)
	require.NoError(t, err)
	assert.Positive(t, space)
}

func TestRealResourceChecker_AvailableMemory(t *testing.T) {
	r := &RealResourceChecker{}

	mem, err := r.AvailableMemory()
	require.NoError(t, err)
	assert.GreaterOrEqual(t, mem, 100*MB, "expected at least 100MB")
}

func TestRealResourceChecker_AvailableCPUs(t *testing.T) {
	r := &RealResourceChecker{}
	assert.GreaterOrEqual(t, r.AvailableCPUs(), 1.0)
}

func TestReadCgroupMemoryLimit(t *testing.T) {
	_, err := readCgroupMemoryLimit("/nonexistent/path")
	require.Error(t, err)

	t.Run("numeric value", func(t *testing.T) {
		tmpFile := writeTempFile(t, "8589934592")
		mem, err := readCgroupMemoryLimit(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, uint64(8589934592), mem)
	})

	t.Run("max value", func(t *testing.T) {
		tmpFile := writeTempFile(t, "max")
		mem, err := readCgroupMemoryLimit(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), mem)
	})

	t.Run("value with newline", func(t *testing.T) {
		tmpFile := writeTempFile(t, "4294967296\n")
		mem, err := readCgroupMemoryLimit(tmpFile)
		require.NoError(t, err)
		assert.Equal(t, uint64(4294967296), mem)
	})

	// cgroup v1 has no "max" keyword: an unlimited controller reports LONG_MAX
	// rounded down to a page boundary, which is not a memory size anybody has.
	t.Run("cgroup v1 unlimited sentinel is not a limit", func(t *testing.T) {
		for _, sentinel := range []string{
			"9223372036854771712", // 4 KiB pages
			"9223372036854710272", // 64 KiB pages
		} {
			tmpFile := writeTempFile(t, sentinel)
			mem, err := readCgroupMemoryLimit(tmpFile)
			require.NoError(t, err)
			assert.Equal(t, uint64(0), mem, "sentinel %s should read as unlimited", sentinel)
		}
	})
}

// Where a container's memory limit lives depends on its cgroup namespace: with
// a private one it is at the root of /sys/fs/cgroup, but under `--cgroupns
// host` the root belongs to the host and the container's own limit sits
// further down the tree at the path named in /proc/self/cgroup.
func TestCgroupPaths_MemoryLimit(t *testing.T) {
	const gib = 1 << 30

	tests := []struct {
		name      string
		self      string
		files     map[string]string
		want      uint64
		wantFound bool
	}{
		{
			name:      "v2 private namespace reads the root",
			self:      "0::/\n",
			files:     map[string]string{"memory.max": "2147483648"},
			want:      2 * gib,
			wantFound: true,
		},
		{
			name: "v2 host namespace follows the path from /proc/self/cgroup",
			self: "0::/system.slice/docker-abc123.scope\n",
			files: map[string]string{
				"memory.max": "max",
				"system.slice/docker-abc123.scope/memory.max": "1073741824",
			},
			want:      gib,
			wantFound: true,
		},
		{
			name: "v2 takes the tightest limit in the chain",
			self: "0::/system.slice/docker-abc123.scope\n",
			files: map[string]string{
				"memory.max":              "max",
				"system.slice/memory.max": "1073741824",
				"system.slice/docker-abc123.scope/memory.max": "4294967296",
			},
			want:      gib,
			wantFound: true,
		},
		{
			name: "v1 reads under the memory controller",
			self: "9:memory:/docker/abc123\n8:cpu,cpuacct:/docker/abc123\n",
			files: map[string]string{
				"memory/docker/abc123/memory.limit_in_bytes": "536870912",
			},
			want:      536870912,
			wantFound: true,
		},
		{
			name: "v1 unlimited is not a limit",
			self: "9:memory:/\n",
			files: map[string]string{
				"memory/memory.limit_in_bytes": "9223372036854771712",
			},
			wantFound: false,
		},
		{
			name:      "no cgroup files means no limit",
			self:      "0::/\n",
			files:     nil,
			wantFound: false,
		},
		{
			name:      "an unreadable /proc/self/cgroup means no limit",
			self:      "",
			files:     map[string]string{"memory.max": "2147483648"},
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := buildCgroupTree(t, tt.self, tt.files)

			limit, found := paths.memoryLimit()

			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.want, limit)
			}
		})
	}
}

func TestAvailableMemory(t *testing.T) {
	const gib = 1 << 30
	system := func(n uint64, err error) func() (uint64, error) {
		return func() (uint64, error) { return n, err }
	}

	t.Run("reports the cgroup limit when it is the binding one", func(t *testing.T) {
		paths := buildCgroupTree(t, "0::/\n", map[string]string{"memory.max": "1073741824"})

		mem, err := availableMemory(paths, system(64*gib, nil))

		require.NoError(t, err)
		assert.Equal(t, uint64(gib), mem)
	})

	// A limit above the machine's own memory cannot be reached, so reporting it
	// would overstate what is actually available.
	t.Run("reports system memory when the cgroup limit exceeds it", func(t *testing.T) {
		paths := buildCgroupTree(t, "0::/\n", map[string]string{"memory.max": "137438953472"})

		mem, err := availableMemory(paths, system(64*gib, nil))

		require.NoError(t, err)
		assert.Equal(t, uint64(64*gib), mem)
	})

	t.Run("falls back to system memory when there is no cgroup", func(t *testing.T) {
		paths := buildCgroupTree(t, "", nil)

		mem, err := availableMemory(paths, system(64*gib, nil))

		require.NoError(t, err)
		assert.Equal(t, uint64(64*gib), mem)
	})

	t.Run("uses the cgroup limit when system memory is unavailable", func(t *testing.T) {
		paths := buildCgroupTree(t, "0::/\n", map[string]string{"memory.max": "1073741824"})

		mem, err := availableMemory(paths, system(0, errors.New("sysinfo failed")))

		require.NoError(t, err)
		assert.Equal(t, uint64(gib), mem)
	})

	t.Run("surfaces the error when neither source works", func(t *testing.T) {
		paths := buildCgroupTree(t, "", nil)

		_, err := availableMemory(paths, system(0, errors.New("sysinfo failed")))

		require.Error(t, err)
	})
}

// A CPU quota is a share of wall time per period, so it has to be read from the
// cgroup and divided out — unlike CPU affinity, the Go runtime never sees it.
func TestCgroupPaths_CPULimit(t *testing.T) {
	tests := []struct {
		name      string
		self      string
		files     map[string]string
		want      float64
		wantFound bool
	}{
		{
			name:      "v2 quota below one whole CPU",
			self:      "0::/\n",
			files:     map[string]string{"cpu.max": "50000 100000"},
			want:      0.5,
			wantFound: true,
		},
		{
			name:      "v2 quota spanning more than one CPU",
			self:      "0::/\n",
			files:     map[string]string{"cpu.max": "150000 100000"},
			want:      1.5,
			wantFound: true,
		},
		{
			name:      "v2 unlimited",
			self:      "0::/\n",
			files:     map[string]string{"cpu.max": "max 100000"},
			wantFound: false,
		},
		{
			name: "v2 host namespace follows the path from /proc/self/cgroup",
			self: "0::/system.slice/docker-abc123.scope\n",
			files: map[string]string{
				"cpu.max": "max 100000",
				"system.slice/docker-abc123.scope/cpu.max": "200000 100000",
			},
			want:      2,
			wantFound: true,
		},
		{
			name: "v2 takes the tightest quota in the chain",
			self: "0::/system.slice/docker-abc123.scope\n",
			files: map[string]string{
				"cpu.max":              "max 100000",
				"system.slice/cpu.max": "100000 100000",
				"system.slice/docker-abc123.scope/cpu.max": "400000 100000",
			},
			want:      1,
			wantFound: true,
		},
		{
			name: "v1 keeps quota and period in separate files",
			self: "5:cpu,cpuacct:/docker/abc123\n",
			files: map[string]string{
				"cpu/docker/abc123/cpu.cfs_quota_us":  "50000",
				"cpu/docker/abc123/cpu.cfs_period_us": "100000",
			},
			want:      0.5,
			wantFound: true,
		},
		{
			name: "v1 unlimited is -1, not a keyword",
			self: "5:cpu,cpuacct:/\n",
			files: map[string]string{
				"cpu/cpu.cfs_quota_us":  "-1",
				"cpu/cpu.cfs_period_us": "100000",
			},
			wantFound: false,
		},
		{
			name:      "no cgroup files means no quota",
			self:      "0::/\n",
			files:     nil,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths := buildCgroupTree(t, tt.self, tt.files)

			limit, found := paths.cpuLimit()

			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.InDelta(t, tt.want, limit, 0.0001)
			}
		})
	}
}

// buildCgroupTree writes a fake /sys/fs/cgroup and /proc/self/cgroup so the
// lookup can be exercised without a container. An empty self makes
// /proc/self/cgroup unreadable.
func buildCgroupTree(t *testing.T, self string, files map[string]string) cgroupPaths {
	t.Helper()

	dir := t.TempDir()
	root := filepath.Join(dir, "cgroup")
	for name, content := range files {
		full := filepath.Join(root, name)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}

	selfFile := filepath.Join(dir, "self-cgroup")
	if self != "" {
		require.NoError(t, os.WriteFile(selfFile, []byte(self), 0o600))
	}

	return cgroupPaths{root: root, selfFile: selfFile}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "memory_limit")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(tmpFile.Name()) })
	_, err = tmpFile.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name()
}
