package main

import (
	"strings"
	"testing"
)

func TestRequireExactlyOne(t *testing.T) {
	tests := []struct {
		name      string
		flags     []flagValue
		wantErr   bool
		errPrefix string
	}{
		{
			name: "no flags set",
			flags: []flagValue{
				{"--foo", ""},
				{"--bar", ""},
			},
			wantErr:   true,
			errPrefix: "one of",
		},
		{
			name: "one flag set",
			flags: []flagValue{
				{"--foo", "value"},
				{"--bar", ""},
			},
			wantErr: false,
		},
		{
			name: "multiple flags set",
			flags: []flagValue{
				{"--foo", "value1"},
				{"--bar", "value2"},
			},
			wantErr:   true,
			errPrefix: "only one of",
		},
		{
			name: "all flags set",
			flags: []flagValue{
				{"--foo", "a"},
				{"--bar", "b"},
				{"--baz", "c"},
			},
			wantErr:   true,
			errPrefix: "only one of",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireExactlyOne(tt.flags...)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errPrefix != "" && !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("error = %q, want prefix %q", err.Error(), tt.errPrefix)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRequireAtLeastOne(t *testing.T) {
	tests := []struct {
		name      string
		flags     []flagSet
		wantErr   bool
		errPrefix string
	}{
		{
			name: "no flags set",
			flags: []flagSet{
				{"--foo", false},
				{"--bar", false},
			},
			wantErr:   true,
			errPrefix: "at least one of",
		},
		{
			name: "one flag set",
			flags: []flagSet{
				{"--foo", true},
				{"--bar", false},
			},
			wantErr: false,
		},
		{
			name: "multiple flags set",
			flags: []flagSet{
				{"--foo", true},
				{"--bar", true},
			},
			wantErr: false,
		},
		{
			name: "last flag set",
			flags: []flagSet{
				{"--foo", false},
				{"--bar", false},
				{"--baz", true},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireAtLeastOne(tt.flags...)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errPrefix != "" && !strings.HasPrefix(err.Error(), tt.errPrefix) {
					t.Errorf("error = %q, want prefix %q", err.Error(), tt.errPrefix)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
