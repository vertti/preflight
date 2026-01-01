package envcheck

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/testutil"
)

type mockEnvGetter struct{ Vars map[string]string }

func (m *mockEnvGetter) LookupEnv(key string) (string, bool) {
	val, ok := m.Vars[key]
	return val, ok
}

type mockFileInfo struct{ isDir bool }

func (m *mockFileInfo) Name() string       { return "mock" }
func (m *mockFileInfo) Size() int64        { return 0 }
func (m *mockFileInfo) Mode() os.FileMode  { return 0 }
func (m *mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() any           { return nil }

type mockFileStater struct{ Files map[string]*mockFileInfo }

func (m *mockFileStater) Stat(path string) (os.FileInfo, error) {
	if info, ok := m.Files[path]; ok && info != nil {
		return info, nil
	}
	return nil, errors.New("file not found")
}

func env(vars map[string]string) *mockEnvGetter { return &mockEnvGetter{Vars: vars} }

func TestEnvCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantDetail string
	}{
		// Default behavior
		{"undefined variable fails", Check{Name: "MISSING_VAR", Getter: env(map[string]string{})}, check.StatusFail, "not set"},
		{"empty variable fails by default", Check{Name: "EMPTY_VAR", Getter: env(map[string]string{"EMPTY_VAR": ""})}, check.StatusFail, "empty value"},
		{"non-empty variable passes", Check{Name: "MY_VAR", Getter: env(map[string]string{"MY_VAR": "some_value"})}, check.StatusOK, "value: some_value"},

		// --allow-empty
		{"allow-empty allows empty value", Check{Name: "EMPTY_VAR", AllowEmpty: true, Getter: env(map[string]string{"EMPTY_VAR": ""})}, check.StatusOK, ""},
		{"allow-empty still fails if undefined", Check{Name: "MISSING_VAR", AllowEmpty: true, Getter: env(map[string]string{})}, check.StatusFail, "not set"},

		// --match
		{"match pattern passes", Check{Name: "DATABASE_URL", Match: `^postgres://`, Getter: env(map[string]string{"DATABASE_URL": "postgres://localhost:5432/db"})}, check.StatusOK, ""},
		{"match pattern fails", Check{Name: "DATABASE_URL", Match: `^postgres://`, Getter: env(map[string]string{"DATABASE_URL": "mysql://localhost:3306/db"})}, check.StatusFail, `does not match pattern`},

		// --exact
		{"exact value passes", Check{Name: "NODE_ENV", Exact: "production", Getter: env(map[string]string{"NODE_ENV": "production"})}, check.StatusOK, ""},
		{"exact value fails", Check{Name: "NODE_ENV", Exact: "production", Getter: env(map[string]string{"NODE_ENV": "development"})}, check.StatusFail, `does not equal "production"`},

		// --one-of
		{"one-of passes", Check{Name: "NODE_ENV", OneOf: []string{"dev", "staging", "production"}, Getter: env(map[string]string{"NODE_ENV": "production"})}, check.StatusOK, ""},
		{"one-of fails", Check{Name: "NODE_ENV", OneOf: []string{"dev", "staging", "production"}, Getter: env(map[string]string{"NODE_ENV": "test"})}, check.StatusFail, "not in allowed list"},
		{"one-of with hide-value", Check{Name: "SECRET_ENV", OneOf: []string{"a", "b", "c"}, HideValue: true, Getter: env(map[string]string{"SECRET_ENV": "x"})}, check.StatusFail, `"[hidden]" not in allowed list`},

		// --hide-value
		{"hide-value hides value", Check{Name: "SECRET", HideValue: true, Getter: env(map[string]string{"SECRET": "super-secret"})}, check.StatusOK, "value: [hidden]"},

		// --mask-value
		{"mask-value masks long value", Check{Name: "API_KEY", MaskValue: true, Getter: env(map[string]string{"API_KEY": "sk-secret-key-xyz"})}, check.StatusOK, "sk-"},
		{"mask-value fully masks short value", Check{Name: "SHORT", MaskValue: true, Getter: env(map[string]string{"SHORT": "abc"})}, check.StatusOK, ""},

		// --starts-with
		{"starts-with passes", Check{Name: "PATH", StartsWith: "/usr", Getter: env(map[string]string{"PATH": "/usr/local/bin"})}, check.StatusOK, ""},
		{"starts-with fails", Check{Name: "PATH", StartsWith: "/usr", Getter: env(map[string]string{"PATH": "/opt/bin"})}, check.StatusFail, `does not start with "/usr"`},

		// --ends-with
		{"ends-with passes", Check{Name: "CONFIG", EndsWith: ".yaml", Getter: env(map[string]string{"CONFIG": "/app/config.yaml"})}, check.StatusOK, ""},
		{"ends-with fails", Check{Name: "CONFIG", EndsWith: ".yaml", Getter: env(map[string]string{"CONFIG": "/app/config.json"})}, check.StatusFail, `does not end with ".yaml"`},

		// --contains
		{"contains passes", Check{Name: "URL", Contains: "example", Getter: env(map[string]string{"URL": "https://example.com/api"})}, check.StatusOK, ""},
		{"contains fails", Check{Name: "URL", Contains: "example", Getter: env(map[string]string{"URL": "https://other.com/api"})}, check.StatusFail, `does not contain "example"`},

		// --is-numeric
		{"is-numeric passes with integer", Check{Name: "PORT", IsNumeric: true, Getter: env(map[string]string{"PORT": "8080"})}, check.StatusOK, ""},
		{"is-numeric passes with float", Check{Name: "RATE", IsNumeric: true, Getter: env(map[string]string{"RATE": "3.14"})}, check.StatusOK, ""},
		{"is-numeric fails", Check{Name: "PORT", IsNumeric: true, Getter: env(map[string]string{"PORT": "not-a-number"})}, check.StatusFail, "is not numeric"},

		// --min-len
		{"min-len passes", Check{Name: "API_KEY", MinLen: 10, Getter: env(map[string]string{"API_KEY": "abcdefghijk"})}, check.StatusOK, ""},
		{"min-len fails", Check{Name: "API_KEY", MinLen: 10, Getter: env(map[string]string{"API_KEY": "short"})}, check.StatusFail, "< minimum 10"},

		// --max-len
		{"max-len passes", Check{Name: "CODE", MaxLen: 10, Getter: env(map[string]string{"CODE": "abc"})}, check.StatusOK, ""},
		{"max-len fails", Check{Name: "CODE", MaxLen: 5, Getter: env(map[string]string{"CODE": "toolongvalue"})}, check.StatusFail, "> maximum 5"},

		// --not-set
		{"not-set passes when undefined", Check{Name: "SHOULD_NOT_EXIST", NotSet: true, Getter: env(map[string]string{})}, check.StatusOK, "not set (as expected)"},
		{"not-set fails when defined", Check{Name: "EXISTS", NotSet: true, Getter: env(map[string]string{"EXISTS": "some-value"})}, check.StatusFail, "variable is set (expected not set)"},
		{"not-set fails when empty", Check{Name: "EMPTY", NotSet: true, Getter: env(map[string]string{"EMPTY": ""})}, check.StatusFail, "variable is set (expected not set)"},

		// --is-port
		{"is-port passes valid port", Check{Name: "PORT", IsPort: true, Getter: env(map[string]string{"PORT": "8080"})}, check.StatusOK, ""},
		{"is-port passes port 1", Check{Name: "PORT", IsPort: true, Getter: env(map[string]string{"PORT": "1"})}, check.StatusOK, ""},
		{"is-port passes port 65535", Check{Name: "PORT", IsPort: true, Getter: env(map[string]string{"PORT": "65535"})}, check.StatusOK, ""},
		{"is-port fails port 0", Check{Name: "PORT", IsPort: true, Getter: env(map[string]string{"PORT": "0"})}, check.StatusFail, "not a valid port"},
		{"is-port fails port 65536", Check{Name: "PORT", IsPort: true, Getter: env(map[string]string{"PORT": "65536"})}, check.StatusFail, "not a valid port"},
		{"is-port fails non-numeric", Check{Name: "PORT", IsPort: true, Getter: env(map[string]string{"PORT": "http"})}, check.StatusFail, "not a valid port"},

		// --is-url
		{"is-url passes valid URL", Check{Name: "URL", IsURL: true, Getter: env(map[string]string{"URL": "https://example.com/path"})}, check.StatusOK, ""},
		{"is-url fails missing scheme", Check{Name: "URL", IsURL: true, Getter: env(map[string]string{"URL": "example.com"})}, check.StatusFail, "not a valid URL"},
		{"is-url fails path only", Check{Name: "URL", IsURL: true, Getter: env(map[string]string{"URL": "/path/to/file"})}, check.StatusFail, "not a valid URL"},

		// --is-json
		{"is-json passes object", Check{Name: "CONFIG", IsJSON: true, Getter: env(map[string]string{"CONFIG": `{"key": "value"}`})}, check.StatusOK, ""},
		{"is-json passes array", Check{Name: "LIST", IsJSON: true, Getter: env(map[string]string{"LIST": `[1, 2, 3]`})}, check.StatusOK, ""},
		{"is-json passes string", Check{Name: "STR", IsJSON: true, Getter: env(map[string]string{"STR": `"hello"`})}, check.StatusOK, ""},
		{"is-json fails invalid", Check{Name: "BAD", IsJSON: true, Getter: env(map[string]string{"BAD": `{key: value}`})}, check.StatusFail, "not valid JSON"},

		// --is-bool
		{"is-bool passes true", Check{Name: "FLAG", IsBool: true, Getter: env(map[string]string{"FLAG": "true"})}, check.StatusOK, ""},
		{"is-bool passes false", Check{Name: "FLAG", IsBool: true, Getter: env(map[string]string{"FLAG": "false"})}, check.StatusOK, ""},
		{"is-bool passes 1", Check{Name: "FLAG", IsBool: true, Getter: env(map[string]string{"FLAG": "1"})}, check.StatusOK, ""},
		{"is-bool passes yes", Check{Name: "FLAG", IsBool: true, Getter: env(map[string]string{"FLAG": "yes"})}, check.StatusOK, ""},
		{"is-bool passes ON (case insensitive)", Check{Name: "FLAG", IsBool: true, Getter: env(map[string]string{"FLAG": "ON"})}, check.StatusOK, ""},
		{"is-bool fails invalid", Check{Name: "FLAG", IsBool: true, Getter: env(map[string]string{"FLAG": "maybe"})}, check.StatusFail, "not a valid boolean"},

		// --min-value
		{"min-value passes", Check{Name: "COUNT", MinValue: testutil.Ptr(10.0), Getter: env(map[string]string{"COUNT": "15"})}, check.StatusOK, ""},
		{"min-value passes at boundary", Check{Name: "COUNT", MinValue: testutil.Ptr(10.0), Getter: env(map[string]string{"COUNT": "10"})}, check.StatusOK, ""},
		{"min-value fails below minimum", Check{Name: "COUNT", MinValue: testutil.Ptr(10.0), Getter: env(map[string]string{"COUNT": "5"})}, check.StatusFail, "< minimum 10"},
		{"min-value fails with non-numeric", Check{Name: "COUNT", MinValue: testutil.Ptr(10.0), Getter: env(map[string]string{"COUNT": "abc"})}, check.StatusFail, "not numeric"},

		// --max-value
		{"max-value passes", Check{Name: "COUNT", MaxValue: testutil.Ptr(100.0), Getter: env(map[string]string{"COUNT": "50"})}, check.StatusOK, ""},
		{"max-value passes at boundary", Check{Name: "COUNT", MaxValue: testutil.Ptr(100.0), Getter: env(map[string]string{"COUNT": "100"})}, check.StatusOK, ""},
		{"max-value fails above maximum", Check{Name: "COUNT", MaxValue: testutil.Ptr(100.0), Getter: env(map[string]string{"COUNT": "150"})}, check.StatusFail, "> maximum 100"},

		// combined min-value and max-value
		{"min and max value in range", Check{Name: "PORT", MinValue: testutil.Ptr(1024.0), MaxValue: testutil.Ptr(65535.0), Getter: env(map[string]string{"PORT": "8080"})}, check.StatusOK, ""},

		// --is-file
		{"is-file passes for existing file", Check{Name: "CONFIG_PATH", IsFile: true, Getter: env(map[string]string{"CONFIG_PATH": "/etc/config.yaml"}), Stater: &mockFileStater{Files: map[string]*mockFileInfo{"/etc/config.yaml": {isDir: false}}}}, check.StatusOK, ""},
		{"is-file fails for non-existent path", Check{Name: "CONFIG_PATH", IsFile: true, Getter: env(map[string]string{"CONFIG_PATH": "/nonexistent/file"}), Stater: &mockFileStater{Files: map[string]*mockFileInfo{}}}, check.StatusFail, "does not exist"},
		{"is-file fails for directory", Check{Name: "CONFIG_PATH", IsFile: true, Getter: env(map[string]string{"CONFIG_PATH": "/etc"}), Stater: &mockFileStater{Files: map[string]*mockFileInfo{"/etc": {isDir: true}}}}, check.StatusFail, "is a directory"},

		// --is-dir
		{"is-dir passes for existing directory", Check{Name: "DATA_DIR", IsDir: true, Getter: env(map[string]string{"DATA_DIR": "/var/data"}), Stater: &mockFileStater{Files: map[string]*mockFileInfo{"/var/data": {isDir: true}}}}, check.StatusOK, ""},
		{"is-dir fails for non-existent path", Check{Name: "DATA_DIR", IsDir: true, Getter: env(map[string]string{"DATA_DIR": "/nonexistent/dir"}), Stater: &mockFileStater{Files: map[string]*mockFileInfo{}}}, check.StatusFail, "does not exist"},
		{"is-dir fails for file", Check{Name: "DATA_DIR", IsDir: true, Getter: env(map[string]string{"DATA_DIR": "/var/data.txt"}), Stater: &mockFileStater{Files: map[string]*mockFileInfo{"/var/data.txt": {isDir: false}}}}, check.StatusFail, "is a file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()
			assert.Equal(t, tt.wantStatus, result.Status, "details: %v", result.Details)
			if tt.wantDetail != "" {
				assert.True(t, testutil.ContainsDetail(result.Details, tt.wantDetail), "details %v should contain %q", result.Details, tt.wantDetail)
			}
		})
	}
}
