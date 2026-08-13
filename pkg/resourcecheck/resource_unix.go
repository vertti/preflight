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
	paths := cgroupPaths{root: "/sys/fs/cgroup", selfFile: "/proc/self/cgroup"}
	return availableMemory(paths, getSystemMemory)
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
//
// The limit is not always at the root of the mount: a container started with
// `--cgroupns host` sees the host's whole cgroup tree, and its own limit sits
// at the path named in /proc/self/cgroup. Limits also nest, so the effective
// one is the smallest along the chain from that path up to the root.
func (p cgroupPaths) memoryLimit() (uint64, bool) {
	data, err := os.ReadFile(p.selfFile)
	if err != nil {
		return 0, false
	}

	var limit uint64
	var found bool
	for _, file := range p.limitFiles(string(data)) {
		mem, err := readCgroupMemoryLimit(file)
		if err != nil || mem == 0 {
			continue
		}
		if !found || mem < limit {
			limit, found = mem, true
		}
	}
	return limit, found
}

// limitFiles expands /proc/self/cgroup into every limit file that could apply,
// from the process's own cgroup up to the root of the hierarchy.
func (p cgroupPaths) limitFiles(selfCgroup string) []string {
	var files []string
	for line := range strings.SplitSeq(selfCgroup, "\n") {
		// Each line is hierarchy-ID:controller-list:cgroup-path.
		fields := strings.SplitN(strings.TrimSpace(line), ":", 3)
		if len(fields) != 3 {
			continue
		}
		controllers, cgroup := fields[1], fields[2]

		var dir, name string
		switch {
		case controllers == "": // cgroup v2 has a single unified hierarchy
			dir, name = p.root, "memory.max"
		case slices.Contains(strings.Split(controllers, ","), "memory"):
			dir, name = filepath.Join(p.root, "memory"), "memory.limit_in_bytes"
		default:
			continue
		}

		for _, ancestor := range ancestors(cgroup) {
			files = append(files, filepath.Join(dir, ancestor, name))
		}
	}
	return files
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
