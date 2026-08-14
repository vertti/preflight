package version

import (
	"errors"
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
		return Version{}, errors.New("empty version string")
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
		return nil, nil //nolint:nilnil // intentional: empty string means "no version constraint"
	}
	v, err := Parse(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// Extract finds and parses the most version-like number in a string.
//
// Taking the leftmost match is not good enough, because program names carry
// digits: read left to right, "s3cmd version 2.3.0" yields 3.0.0 and
// "x264 0.164.3108" yields 264.0.0. Every candidate is scored instead, and the
// best one wins.
func Extract(s string) (Version, error) {
	candidates := versionRegex.FindAllStringSubmatchIndex(s, -1)
	if candidates == nil {
		return Version{}, fmt.Errorf("no version found in: %q", s)
	}

	best := candidates[0]
	bestScore := score(s, best)
	for _, c := range candidates[1:] {
		// Strictly greater keeps the leftmost of equally good candidates.
		if cScore := score(s, c); cScore > bestScore {
			best, bestScore = c, cScore
		}
	}

	return parseMatches(submatches(s, best)), nil
}

// score ranks a candidate by how much it looks like a version rather than a
// number that happens to sit in a name.
//
// Both halves are load-bearing. Component count alone cannot separate the "7"
// from the "23.01" in "7-Zip 23.01", since both start a word. Standing alone,
// used on its own, would reject every candidate in "go version go1.25.0", where
// the real version is glued to the name — so it only breaks ties, and a
// two-component candidate still outranks a lone standalone digit.
func score(s string, loc []int) int {
	components := 0
	for group := 1; group <= 3; group++ {
		if loc[2*group] >= 0 {
			components++
		}
	}

	standalone := 0
	if loc[0] == 0 || !isWordByte(s[loc[0]-1]) {
		standalone = 1
	}

	return components*2 + standalone
}

// isWordByte reports whether b would glue a digit to a surrounding name. A
// multi-byte rune's bytes are all >= 0x80 and so count as a separator, which is
// what we want: a version after a non-ASCII character stands on its own.
func isWordByte(b byte) bool {
	return b >= '0' && b <= '9' || b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
}

// submatches converts an index pair list from FindAllStringSubmatchIndex into
// the group strings that parseMatches expects.
func submatches(s string, loc []int) []string {
	groups := make([]string, len(loc)/2)
	for i := range groups {
		if start := loc[2*i]; start >= 0 {
			groups[i] = s[start:loc[2*i+1]]
		}
	}
	return groups
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
