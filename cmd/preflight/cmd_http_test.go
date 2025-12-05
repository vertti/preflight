package main

import (
	"reflect"
	"testing"
)

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    map[string]string
	}{
		{
			name:    "empty input",
			headers: nil,
			want:    map[string]string{},
		},
		{
			name:    "single header",
			headers: []string{"Authorization:Bearer token123"},
			want:    map[string]string{"Authorization": "Bearer token123"},
		},
		{
			name:    "multiple headers",
			headers: []string{"X-API-Key:secret", "Content-Type:application/json"},
			want: map[string]string{
				"X-API-Key":    "secret",
				"Content-Type": "application/json",
			},
		},
		{
			name:    "header with spaces around colon",
			headers: []string{"Authorization : Bearer token"},
			want:    map[string]string{"Authorization": "Bearer token"},
		},
		{
			name:    "header with multiple colons in value",
			headers: []string{"X-URL:http://example.com:8080"},
			want:    map[string]string{"X-URL": "http://example.com:8080"},
		},
		{
			name:    "invalid header without colon ignored",
			headers: []string{"InvalidHeader", "Valid:Header"},
			want:    map[string]string{"Valid": "Header"},
		},
		{
			name:    "empty value",
			headers: []string{"X-Empty:"},
			want:    map[string]string{"X-Empty": ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHeaders(tt.headers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}
