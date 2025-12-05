package main

import (
	"reflect"
	"testing"
)

func TestParseVersionArgs(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"--version", []string{"--version"}},
		{"-v", []string{"-v"}},
		{"-version", []string{"-version"}},
		{"version --short", []string{"version", "--short"}},
		{"", []string{}},
		{"  --version  ", []string{"--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseVersionArgs(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseVersionArgs(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
