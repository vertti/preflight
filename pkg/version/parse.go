package version

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version with major, minor, patch components.
type Version struct {
	Major int
	Minor int
	Patch int
}

// String returns the version as a string.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// versionRegex matches version patterns like 1.2.3, v1.2, 18, etc.
var versionRegex = regexp.MustCompile(`v?(\d+)(?:\.(\d+))?(?:\.(\d+))?`)

// Parse parses a version string into a Version.
func Parse(s string) (Version, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Version{}, fmt.Errorf("empty version string")
	}

	matches := versionRegex.FindStringSubmatch(s)
	if matches == nil {
		return Version{}, fmt.Errorf("invalid version format: %q", s)
	}

	if matches[0] != s {
		return Version{}, fmt.Errorf("invalid version format: %q", s)
	}

	return parseMatches(matches), nil
}

// ParseOptional parses a version string if non-empty.
// Returns nil, nil for empty strings.
func ParseOptional(s string) (*Version, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	v, err := Parse(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Extract finds and parses the first version number in a string.
func Extract(s string) (Version, error) {
	matches := versionRegex.FindStringSubmatch(s)
	if matches == nil {
		return Version{}, fmt.Errorf("no version found in: %q", s)
	}
	return parseMatches(matches), nil
}

func parseMatches(matches []string) Version {
	major, _ := strconv.Atoi(matches[1])
	var minor, patch int
	if matches[2] != "" {
		minor, _ = strconv.Atoi(matches[2])
	}
	if matches[3] != "" {
		patch, _ = strconv.Atoi(matches[3])
	}
	return Version{Major: major, Minor: minor, Patch: patch}
}
