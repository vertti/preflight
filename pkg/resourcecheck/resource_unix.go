//go:build unix

package resourcecheck

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

// FreeDiskSpace returns free disk space in bytes.
func (r *RealResourceChecker) FreeDiskSpace(path string) (uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	// Available blocks * block size
	return stat.Bavail * uint64(stat.Bsize), nil // #nosec G115 -- block size is always positive
}

// AvailableMemory returns available memory in bytes.
// First checks cgroup limits (for containers), then falls back to system memory.
func (r *RealResourceChecker) AvailableMemory() (uint64, error) {
	return availableMemory(systemCgroupPaths(), getSystemMemory)
}

// cpuQuota reports the CPU quota this process runs under, if any.
func cpuQuota() (float64, bool) {
	return systemCgroupPaths().cpuLimit()
}

func systemCgroupPaths() cgroupPaths {
	return cgroupPaths{root: "/sys/fs/cgroup", selfFile: "/proc/self/cgroup"}
}

// availableMemory reports the smaller of the machine's memory and the cgroup
// limit the process runs under. A limit larger than the machine's memory is not
// reachable, so it would overstate what a container can actually use.
func availableMemory(paths cgroupPaths, systemMemory func() (uint64, error)) (uint64, error) {
	system, err := systemMemory()

	limit, limited := paths.memoryLimit()
	switch {
	case !limited:
		return system, err
	case err != nil || limit < system:
		//nolint:nilerr // a cgroup limit answers the question on its own, so an
		// unreadable system memory is not worth failing over
		return limit, nil
	default:
		return system, nil
	}
}

// cgroupPaths locates the cgroup filesystem and the process's own membership in
// it. The fields exist so tests can point at a fixture tree.
type cgroupPaths struct {
	root     string // cgroup mount point, normally /sys/fs/cgroup
	selfFile string // normally /proc/self/cgroup
}

// memoryLimit returns the tightest memory limit applying to this process.
func (p cgroupPaths) memoryLimit() (uint64, bool) {
	var limit uint64
	var found bool
	for _, dir := range p.controllerDirs("memory") {
		mem, err := readCgroupMemoryLimit(dir.file("memory.max", "memory.limit_in_bytes"))
		if err != nil || mem == 0 {
			continue
		}
		if !found || mem < limit {
			limit, found = mem, true
		}
	}
	return limit, found
}

// cpuLimit returns the tightest CPU quota applying to this process, counted in
// CPUs. A quota is a share of wall time, so it is routinely a fraction.
func (p cgroupPaths) cpuLimit() (float64, bool) {
	var limit float64
	var found bool
	for _, dir := range p.controllerDirs("cpu") {
		cpus, ok := dir.cpuQuota()
		if !ok {
			continue
		}
		if !found || cpus < limit {
			limit, found = cpus, true
		}
	}
	return limit, found
}

// cgroupDir is a directory that may hold limits for this process, along with
// which cgroup version's file names to look for inside it.
type cgroupDir struct {
	path string
	v2   bool
}

// file picks between the two versions' names for the same limit.
func (d cgroupDir) file(v2Name, v1Name string) string {
	if d.v2 {
		return filepath.Join(d.path, v2Name)
	}
	return filepath.Join(d.path, v1Name)
}

// controllerDirs lists every directory that could hold a limit for the given
// controller, nearest first.
//
// The limit is not always at the root of the mount: a container started with
// `--cgroupns host` sees the host's whole cgroup tree, and its own limits sit
// at the path named in /proc/self/cgroup. Limits also nest, so a caller has to
// consider the whole chain from that path up to the root.
func (p cgroupPaths) controllerDirs(controller string) []cgroupDir {
	data, err := os.ReadFile(p.selfFile)
	if err != nil {
		return nil
	}

	var dirs []cgroupDir
	for line := range strings.SplitSeq(string(data), "\n") {
		// Each line is hierarchy-ID:controller-list:cgroup-path.
		fields := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(fields) != 3 {
			continue
		}
		controllers, cgroup := fields[1], fields[2]

		v2 := controllers == "" // cgroup v2 has a single unified hierarchy
		var base string
		switch {
		case v2:
			base = p.root
		case slices.Contains(strings.Split(controllers, ","), controller):
			base = filepath.Join(p.root, controller)
		default:
			continue
		}

		for _, ancestor := range ancestors(cgroup) {
			dirs = append(dirs, cgroupDir{path: filepath.Join(base, ancestor), v2: v2})
		}
	}
	return dirs
}

// ancestors lists a cgroup path and every parent above it, ending at the root.
func ancestors(cgroup string) []string {
	if !strings.HasPrefix(cgroup, "/") {
		cgroup = "/" + cgroup
	}

	paths := []string{cgroup}
	for cgroup != "/" {
		cgroup = filepath.Dir(cgroup)
		paths = append(paths, cgroup)
	}
	return paths
}

// cpuQuota reads how much CPU time this cgroup may use, as a number of CPUs.
// cgroup v2 keeps the quota and its period in one cpu.max file; v1 splits them
// across two.
func (d cgroupDir) cpuQuota() (float64, bool) {
	if d.v2 {
		data, err := os.ReadFile(filepath.Join(d.path, "cpu.max"))
		if err != nil {
			return 0, false
		}
		fields := strings.Fields(string(data))
		if len(fields) != 2 {
			return 0, false
		}
		return cpusFromQuota(fields[0], fields[1])
	}

	quota, err := os.ReadFile(filepath.Join(d.path, "cpu.cfs_quota_us"))
	if err != nil {
		return 0, false
	}
	period, err := os.ReadFile(filepath.Join(d.path, "cpu.cfs_period_us"))
	if err != nil {
		return 0, false
	}
	return cpusFromQuota(strings.TrimSpace(string(quota)), strings.TrimSpace(string(period)))
}

// cpusFromQuota divides a quota by its period to get a number of CPUs. An
// unlimited controller says "max" under v2 and "-1" under v1.
func cpusFromQuota(quota, period string) (float64, bool) {
	allowed, err := strconv.ParseFloat(quota, 64)
	if err != nil || allowed <= 0 {
		return 0, false
	}
	per, err := strconv.ParseFloat(period, 64)
	if err != nil || per <= 0 {
		return 0, false
	}
	return allowed / per, true
}

// An unlimited cgroup v1 controller reports LONG_MAX rounded down to a page
// boundary rather than a keyword, and the exact figure depends on the page
// size. Anything in that neighbourhood is a sentinel, not a memory size.
const unlimitedMemory = 1 << 62

// readCgroupMemoryLimit reads memory limit from a cgroup file.
// A zero result means the file says the memory is unlimited.
func readCgroupMemoryLimit(path string) (uint64, error) {
	data, err := os.ReadFile(path) //nolint:gosec // intentional: reading cgroup files
	if err != nil {
		return 0, err
	}

	content := strings.TrimSpace(string(data))

	// "max" means unlimited in cgroup v2
	if content == "max" {
		return 0, nil
	}

	limit, err := strconv.ParseUint(content, 10, 64)
	if err != nil {
		return 0, err
	}
	if limit >= unlimitedMemory {
		return 0, nil
	}
	return limit, nil
}
