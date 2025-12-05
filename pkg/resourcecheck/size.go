package resourcecheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	_         = iota
	KB uint64 = 1 << (10 * iota)
	MB
	GB
	TB
)

// ParseSize parses a human-readable size string into bytes.
// Supports: B, K/KB, M/MB, G/GB, T/TB (case-insensitive)
// Examples: "10G", "500M", "1TB", "1024", "1.5GB"
func ParseSize(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Match number (with optional decimal) and optional unit
	re := regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)\s*([KMGT]?B?)?$`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0, fmt.Errorf("invalid size format: %q", s)
	}

	numStr := matches[1]
	unit := strings.ToUpper(matches[2])

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %q", numStr)
	}

	var multiplier uint64
	switch unit {
	case "", "B":
		multiplier = 1
	case "K", "KB":
		multiplier = KB
	case "M", "MB":
		multiplier = MB
	case "G", "GB":
		multiplier = GB
	case "T", "TB":
		multiplier = TB
	default:
		return 0, fmt.Errorf("unknown unit: %q", unit)
	}

	return uint64(num * float64(multiplier)), nil
}

// FormatSize formats bytes into a human-readable string.
func FormatSize(bytes uint64) string {
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fTB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1fGB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1fMB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1fKB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
