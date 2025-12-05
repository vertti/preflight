package resourcecheck

import (
	"testing"
)

func TestParseSize(t *testing.T) {
	tests := []struct {
		input   string
		want    uint64
		wantErr bool
	}{
		// Basic units
		{"1024", 1024, false},
		{"1024B", 1024, false},
		{"1K", KB, false},
		{"1KB", KB, false},
		{"1M", MB, false},
		{"1MB", MB, false},
		{"1G", GB, false},
		{"1GB", GB, false},
		{"1T", TB, false},
		{"1TB", TB, false},

		// Case insensitivity
		{"1g", GB, false},
		{"1gb", GB, false},
		{"1Gb", GB, false},

		// Multipliers
		{"10G", 10 * GB, false},
		{"500M", 500 * MB, false},
		{"2T", 2 * TB, false},
		{"100K", 100 * KB, false},

		// Decimals
		{"1.5G", uint64(1.5 * float64(GB)), false},
		{"2.5MB", uint64(2.5 * float64(MB)), false},

		// Whitespace
		{" 10G ", 10 * GB, false},
		{"10 G", 10 * GB, false},

		// Errors
		{"", 0, true},
		{"abc", 0, true},
		{"10X", 0, true},
		{"-10G", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseSize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSize(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0B"},
		{500, "500B"},
		{1023, "1023B"},
		{KB, "1.0KB"},
		{MB, "1.0MB"},
		{GB, "1.0GB"},
		{TB, "1.0TB"},
		{10 * GB, "10.0GB"},
		{uint64(1.5 * float64(GB)), "1.5GB"},
		{500 * MB, "500.0MB"},
		{2 * TB, "2.0TB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatSize(tt.input)
			if got != tt.want {
				t.Errorf("FormatSize(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
