package jsoncheck

import (
	"strings"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

type mockFS struct {
	Content []byte
	Err     error
}

func (m *mockFS) ReadFile(_ string) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Content, nil
}

func TestJSONCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantDetail string
	}{
		// Syntax validation tests
		{
			name: "valid JSON object passes",
			check: Check{
				File: "config.json",
				FS:   &mockFS{Content: []byte(`{"name": "test"}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "syntax: valid",
		},
		{
			name: "valid JSON array passes",
			check: Check{
				File: "data.json",
				FS:   &mockFS{Content: []byte(`[1, 2, 3]`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "syntax: valid",
		},
		{
			name: "invalid JSON fails",
			check: Check{
				File: "bad.json",
				FS:   &mockFS{Content: []byte(`{invalid}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: "invalid JSON",
		},
		{
			name: "empty file is invalid JSON",
			check: Check{
				File: "empty.json",
				FS:   &mockFS{Content: []byte(``)},
			},
			wantStatus: check.StatusFail,
			wantDetail: "invalid JSON",
		},

		// --has-key tests
		{
			name: "has-key exists passes",
			check: Check{
				File:   "config.json",
				HasKey: "name",
				FS:     &mockFS{Content: []byte(`{"name": "test"}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "has key: name",
		},
		{
			name: "has-key missing fails",
			check: Check{
				File:   "config.json",
				HasKey: "missing",
				FS:     &mockFS{Content: []byte(`{"name": "test"}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: `key "missing" not found`,
		},
		{
			name: "has-key nested exists passes",
			check: Check{
				File:   "config.json",
				HasKey: "database.host",
				FS:     &mockFS{Content: []byte(`{"database": {"host": "localhost"}}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "has key: database.host",
		},
		{
			name: "has-key nested missing fails",
			check: Check{
				File:   "config.json",
				HasKey: "database.port",
				FS:     &mockFS{Content: []byte(`{"database": {"host": "localhost"}}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: `key "database.port" not found`,
		},
		{
			name: "has-key on non-object path fails",
			check: Check{
				File:   "config.json",
				HasKey: "name.nested",
				FS:     &mockFS{Content: []byte(`{"name": "test"}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: `key "name.nested" not found`,
		},

		// --key + --exact tests
		{
			name: "key exact match passes",
			check: Check{
				File:  "config.json",
				Key:   "env",
				Exact: "production",
				FS:    &mockFS{Content: []byte(`{"env": "production"}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key env: production",
		},
		{
			name: "key exact mismatch fails",
			check: Check{
				File:  "config.json",
				Key:   "env",
				Exact: "production",
				FS:    &mockFS{Content: []byte(`{"env": "development"}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value "development" does not equal "production"`,
		},
		{
			name: "key exact on nested key passes",
			check: Check{
				File:  "config.json",
				Key:   "db.host",
				Exact: "localhost",
				FS:    &mockFS{Content: []byte(`{"db": {"host": "localhost"}}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key db.host: localhost",
		},
		{
			name: "key not found fails",
			check: Check{
				File:  "config.json",
				Key:   "missing",
				Exact: "value",
				FS:    &mockFS{Content: []byte(`{"name": "test"}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: `key "missing" not found`,
		},

		// --key + --match tests
		{
			name: "key match pattern passes",
			check: Check{
				File:  "config.json",
				Key:   "version",
				Match: `^1\.`,
				FS:    &mockFS{Content: []byte(`{"version": "1.2.3"}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key version: 1.2.3",
		},
		{
			name: "key match pattern fails",
			check: Check{
				File:  "config.json",
				Key:   "version",
				Match: `^1\.`,
				FS:    &mockFS{Content: []byte(`{"version": "2.0.0"}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: `does not match pattern`,
		},
		{
			name: "key match invalid regex fails",
			check: Check{
				File:  "config.json",
				Key:   "version",
				Match: `[invalid`,
				FS:    &mockFS{Content: []byte(`{"version": "1.0.0"}`)},
			},
			wantStatus: check.StatusFail,
			wantDetail: "invalid regex pattern",
		},

		// Non-string value coercion tests
		{
			name: "key exact on number value",
			check: Check{
				File:  "config.json",
				Key:   "port",
				Exact: "8080",
				FS:    &mockFS{Content: []byte(`{"port": 8080}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key port: 8080",
		},
		{
			name: "key exact on boolean value",
			check: Check{
				File:  "config.json",
				Key:   "enabled",
				Exact: "true",
				FS:    &mockFS{Content: []byte(`{"enabled": true}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key enabled: true",
		},
		{
			name: "key exact on null value",
			check: Check{
				File:  "config.json",
				Key:   "value",
				Exact: "null",
				FS:    &mockFS{Content: []byte(`{"value": null}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key value: null",
		},
		{
			name: "key match on float value",
			check: Check{
				File:  "config.json",
				Key:   "rate",
				Match: `^0\.5`,
				FS:    &mockFS{Content: []byte(`{"rate": 0.5}`)},
			},
			wantStatus: check.StatusOK,
			wantDetail: "key rate: 0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}

			if tt.wantDetail != "" {
				found := false
				for _, d := range result.Details {
					if strings.Contains(d, tt.wantDetail) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Details %v does not contain %q", result.Details, tt.wantDetail)
				}
			}
		})
	}
}

func TestGetByDotPath(t *testing.T) {
	data := map[string]any{
		"name": "test",
		"database": map[string]any{
			"host": "localhost",
			"port": float64(5432),
		},
	}

	tests := []struct {
		path      string
		wantValue any
		wantFound bool
	}{
		{"name", "test", true},
		{"database", map[string]any{"host": "localhost", "port": float64(5432)}, true},
		{"database.host", "localhost", true},
		{"database.port", float64(5432), true},
		{"missing", nil, false},
		{"database.missing", nil, false},
		{"name.nested", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			val, found := getByDotPath(data, tt.path)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if tt.wantFound && tt.wantValue != nil {
				// Simple string comparison for scalars
				if s, ok := tt.wantValue.(string); ok {
					if val != s {
						t.Errorf("value = %v, want %v", val, tt.wantValue)
					}
				}
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		input any
		want  string
	}{
		{"hello", "hello"},
		{float64(42), "42"},
		{float64(3.14), "3.14"},
		{true, "true"},
		{false, "false"},
		{nil, "null"},
		{[]any{1, 2}, "[1,2]"},
		{map[string]any{"a": "b"}, `{"a":"b"}`},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := toString(tt.input)
			if got != tt.want {
				t.Errorf("toString(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
