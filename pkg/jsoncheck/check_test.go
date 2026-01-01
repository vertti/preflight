package jsoncheck

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/testutil"
)

type mockFS struct {
	Content []byte
	Err     error
}

func (m *mockFS) ReadFile(string) ([]byte, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Content, nil
}

func fs(content string) *mockFS { return &mockFS{Content: []byte(content)} }

func TestJSONCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantDetail string
	}{
		// Syntax validation
		{"valid JSON object", Check{File: "config.json", FS: fs(`{"name": "test"}`)}, check.StatusOK, "syntax: valid"},
		{"valid JSON array", Check{File: "data.json", FS: fs(`[1, 2, 3]`)}, check.StatusOK, "syntax: valid"},
		{"invalid JSON", Check{File: "bad.json", FS: fs(`{invalid}`)}, check.StatusFail, "invalid JSON"},
		{"empty file invalid", Check{File: "empty.json", FS: fs(``)}, check.StatusFail, "invalid JSON"},

		// --has-key
		{"has-key exists", Check{File: "f.json", HasKey: "name", FS: fs(`{"name": "test"}`)}, check.StatusOK, "has key: name"},
		{"has-key missing", Check{File: "f.json", HasKey: "missing", FS: fs(`{"name": "test"}`)}, check.StatusFail, `key "missing" not found`},
		{"has-key nested exists", Check{File: "f.json", HasKey: "database.host", FS: fs(`{"database": {"host": "localhost"}}`)}, check.StatusOK, "has key: database.host"},
		{"has-key nested missing", Check{File: "f.json", HasKey: "database.port", FS: fs(`{"database": {"host": "localhost"}}`)}, check.StatusFail, `key "database.port" not found`},
		{"has-key on non-object", Check{File: "f.json", HasKey: "name.nested", FS: fs(`{"name": "test"}`)}, check.StatusFail, `key "name.nested" not found`},

		// --key + --exact
		{"key exact match", Check{File: "f.json", Key: "env", Exact: "production", FS: fs(`{"env": "production"}`)}, check.StatusOK, "key env: production"},
		{"key exact mismatch", Check{File: "f.json", Key: "env", Exact: "production", FS: fs(`{"env": "development"}`)}, check.StatusFail, `value "development" does not equal "production"`},
		{"key exact nested", Check{File: "f.json", Key: "db.host", Exact: "localhost", FS: fs(`{"db": {"host": "localhost"}}`)}, check.StatusOK, "key db.host: localhost"},
		{"key not found", Check{File: "f.json", Key: "missing", Exact: "value", FS: fs(`{"name": "test"}`)}, check.StatusFail, `key "missing" not found`},

		// --key + --match
		{"key match pattern", Check{File: "f.json", Key: "version", Match: `^1\.`, FS: fs(`{"version": "1.2.3"}`)}, check.StatusOK, "key version: 1.2.3"},
		{"key match fails", Check{File: "f.json", Key: "version", Match: `^1\.`, FS: fs(`{"version": "2.0.0"}`)}, check.StatusFail, `does not match pattern`},
		{"key match invalid regex", Check{File: "f.json", Key: "version", Match: `[invalid`, FS: fs(`{"version": "1.0.0"}`)}, check.StatusFail, "invalid regex pattern"},

		// Non-string value coercion
		{"key exact number", Check{File: "f.json", Key: "port", Exact: "8080", FS: fs(`{"port": 8080}`)}, check.StatusOK, "key port: 8080"},
		{"key exact boolean", Check{File: "f.json", Key: "enabled", Exact: "true", FS: fs(`{"enabled": true}`)}, check.StatusOK, "key enabled: true"},
		{"key exact null", Check{File: "f.json", Key: "value", Exact: "null", FS: fs(`{"value": null}`)}, check.StatusOK, "key value: null"},
		{"key match float", Check{File: "f.json", Key: "rate", Match: `^0\.5`, FS: fs(`{"rate": 0.5}`)}, check.StatusOK, "key rate: 0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status)
			if tt.wantDetail != "" {
				assert.True(t, testutil.ContainsDetail(result.Details, tt.wantDetail), "details %v should contain %q", result.Details, tt.wantDetail)
			}
		})
	}
}
