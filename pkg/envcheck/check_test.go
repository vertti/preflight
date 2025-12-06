package envcheck

import (
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func float64Ptr(f float64) *float64 {
	return &f
}

type mockEnvGetter struct {
	Vars map[string]string
}

func (m *mockEnvGetter) LookupEnv(key string) (string, bool) {
	val, ok := m.Vars[key]
	return val, ok
}

func TestEnvCheck_Run(t *testing.T) {
	tests := []struct {
		name       string
		check      Check
		wantStatus check.Status
		wantDetail string
	}{
		// Default behavior tests
		{
			name: "undefined variable fails",
			check: Check{
				Name:   "MISSING_VAR",
				Getter: &mockEnvGetter{Vars: map[string]string{}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "not set",
		},
		{
			name: "empty variable fails by default",
			check: Check{
				Name:   "EMPTY_VAR",
				Getter: &mockEnvGetter{Vars: map[string]string{"EMPTY_VAR": ""}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "empty value",
		},
		{
			name: "non-empty variable passes",
			check: Check{
				Name:   "MY_VAR",
				Getter: &mockEnvGetter{Vars: map[string]string{"MY_VAR": "some_value"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: some_value",
		},

		// --allow-empty flag tests
		{
			name: "allow-empty allows empty value",
			check: Check{
				Name:       "EMPTY_VAR",
				AllowEmpty: true,
				Getter:     &mockEnvGetter{Vars: map[string]string{"EMPTY_VAR": ""}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "allow-empty still fails if undefined",
			check: Check{
				Name:       "MISSING_VAR",
				AllowEmpty: true,
				Getter:     &mockEnvGetter{Vars: map[string]string{}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "not set",
		},

		// --match flag tests
		{
			name: "match pattern passes",
			check: Check{
				Name:   "DATABASE_URL",
				Match:  `^postgres://`,
				Getter: &mockEnvGetter{Vars: map[string]string{"DATABASE_URL": "postgres://localhost:5432/db"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "match pattern fails",
			check: Check{
				Name:   "DATABASE_URL",
				Match:  `^postgres://`,
				Getter: &mockEnvGetter{Vars: map[string]string{"DATABASE_URL": "mysql://localhost:3306/db"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value does not match pattern "^postgres://"`,
		},

		// --exact flag tests
		{
			name: "exact value passes",
			check: Check{
				Name:   "NODE_ENV",
				Exact:  "production",
				Getter: &mockEnvGetter{Vars: map[string]string{"NODE_ENV": "production"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "exact value fails",
			check: Check{
				Name:   "NODE_ENV",
				Exact:  "production",
				Getter: &mockEnvGetter{Vars: map[string]string{"NODE_ENV": "development"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value does not equal "production"`,
		},

		// --one-of flag tests
		{
			name: "one-of passes",
			check: Check{
				Name:   "NODE_ENV",
				OneOf:  []string{"dev", "staging", "production"},
				Getter: &mockEnvGetter{Vars: map[string]string{"NODE_ENV": "production"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "one-of fails",
			check: Check{
				Name:   "NODE_ENV",
				OneOf:  []string{"dev", "staging", "production"},
				Getter: &mockEnvGetter{Vars: map[string]string{"NODE_ENV": "test"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value "test" not in allowed list [dev staging production]`,
		},
		{
			name: "one-of with hide-value hides value in error",
			check: Check{
				Name:      "SECRET_ENV",
				OneOf:     []string{"a", "b", "c"},
				HideValue: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"SECRET_ENV": "x"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value "[hidden]" not in allowed list [a b c]`,
		},

		// --hide-value flag tests
		{
			name: "hide-value hides value in output",
			check: Check{
				Name:      "SECRET",
				HideValue: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"SECRET": "super-secret"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: [hidden]",
		},

		// --mask-value flag tests
		{
			name: "mask-value masks long value",
			check: Check{
				Name:      "API_KEY",
				MaskValue: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"API_KEY": "sk-secret-key-xyz"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: sk-•••xyz",
		},
		{
			name: "mask-value fully masks short value",
			check: Check{
				Name:      "SHORT",
				MaskValue: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"SHORT": "abc"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: •••",
		},

		// --starts-with flag tests
		{
			name: "starts-with passes",
			check: Check{
				Name:       "PATH",
				StartsWith: "/usr",
				Getter:     &mockEnvGetter{Vars: map[string]string{"PATH": "/usr/local/bin"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "starts-with fails",
			check: Check{
				Name:       "PATH",
				StartsWith: "/usr",
				Getter:     &mockEnvGetter{Vars: map[string]string{"PATH": "/opt/bin"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value does not start with "/usr"`,
		},

		// --ends-with flag tests
		{
			name: "ends-with passes",
			check: Check{
				Name:     "CONFIG",
				EndsWith: ".yaml",
				Getter:   &mockEnvGetter{Vars: map[string]string{"CONFIG": "/app/config.yaml"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "ends-with fails",
			check: Check{
				Name:     "CONFIG",
				EndsWith: ".yaml",
				Getter:   &mockEnvGetter{Vars: map[string]string{"CONFIG": "/app/config.json"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value does not end with ".yaml"`,
		},

		// --contains flag tests
		{
			name: "contains passes",
			check: Check{
				Name:     "URL",
				Contains: "example",
				Getter:   &mockEnvGetter{Vars: map[string]string{"URL": "https://example.com/api"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "contains fails",
			check: Check{
				Name:     "URL",
				Contains: "example",
				Getter:   &mockEnvGetter{Vars: map[string]string{"URL": "https://other.com/api"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: `value does not contain "example"`,
		},

		// --is-numeric flag tests
		{
			name: "is-numeric passes with integer",
			check: Check{
				Name:      "PORT",
				IsNumeric: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"PORT": "8080"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-numeric passes with float",
			check: Check{
				Name:      "RATE",
				IsNumeric: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"RATE": "3.14"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-numeric fails",
			check: Check{
				Name:      "PORT",
				IsNumeric: true,
				Getter:    &mockEnvGetter{Vars: map[string]string{"PORT": "not-a-number"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not numeric",
		},

		// --min-len flag tests
		{
			name: "min-len passes",
			check: Check{
				Name:   "API_KEY",
				MinLen: 10,
				Getter: &mockEnvGetter{Vars: map[string]string{"API_KEY": "abcdefghijk"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "min-len fails",
			check: Check{
				Name:   "API_KEY",
				MinLen: 10,
				Getter: &mockEnvGetter{Vars: map[string]string{"API_KEY": "short"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value length 5 < minimum 10",
		},

		// --max-len flag tests
		{
			name: "max-len passes",
			check: Check{
				Name:   "CODE",
				MaxLen: 10,
				Getter: &mockEnvGetter{Vars: map[string]string{"CODE": "abc"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "max-len fails",
			check: Check{
				Name:   "CODE",
				MaxLen: 5,
				Getter: &mockEnvGetter{Vars: map[string]string{"CODE": "toolongvalue"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value length 12 > maximum 5",
		},

		// --not-set flag tests
		{
			name: "not-set passes when variable undefined",
			check: Check{
				Name:   "SHOULD_NOT_EXIST",
				NotSet: true,
				Getter: &mockEnvGetter{Vars: map[string]string{}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "not set (as expected)",
		},
		{
			name: "not-set fails when variable is defined",
			check: Check{
				Name:   "EXISTS",
				NotSet: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"EXISTS": "some-value"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "variable is set (expected not set)",
		},
		{
			name: "not-set fails when variable is defined but empty",
			check: Check{
				Name:   "EMPTY",
				NotSet: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"EMPTY": ""}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "variable is set (expected not set)",
		},

		// --is-port flag tests
		{
			name: "is-port passes with valid port",
			check: Check{
				Name:   "PORT",
				IsPort: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"PORT": "8080"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-port passes with port 1",
			check: Check{
				Name:   "PORT",
				IsPort: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"PORT": "1"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-port passes with port 65535",
			check: Check{
				Name:   "PORT",
				IsPort: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"PORT": "65535"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-port fails with port 0",
			check: Check{
				Name:   "PORT",
				IsPort: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"PORT": "0"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not a valid port (1-65535)",
		},
		{
			name: "is-port fails with port 65536",
			check: Check{
				Name:   "PORT",
				IsPort: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"PORT": "65536"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not a valid port (1-65535)",
		},
		{
			name: "is-port fails with non-numeric",
			check: Check{
				Name:   "PORT",
				IsPort: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"PORT": "http"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not a valid port (1-65535)",
		},

		// --is-url flag tests
		{
			name: "is-url passes with valid URL",
			check: Check{
				Name:   "URL",
				IsURL:  true,
				Getter: &mockEnvGetter{Vars: map[string]string{"URL": "https://example.com/path"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-url fails with missing scheme",
			check: Check{
				Name:   "URL",
				IsURL:  true,
				Getter: &mockEnvGetter{Vars: map[string]string{"URL": "example.com"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not a valid URL",
		},
		{
			name: "is-url fails with path only",
			check: Check{
				Name:   "URL",
				IsURL:  true,
				Getter: &mockEnvGetter{Vars: map[string]string{"URL": "/path/to/file"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not a valid URL",
		},

		// --is-json flag tests
		{
			name: "is-json passes with valid JSON object",
			check: Check{
				Name:   "CONFIG",
				IsJSON: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"CONFIG": `{"key": "value"}`}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-json passes with valid JSON array",
			check: Check{
				Name:   "LIST",
				IsJSON: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"LIST": `[1, 2, 3]`}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-json passes with JSON string",
			check: Check{
				Name:   "STR",
				IsJSON: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"STR": `"hello"`}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-json fails with invalid JSON",
			check: Check{
				Name:   "BAD",
				IsJSON: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"BAD": `{key: value}`}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not valid JSON",
		},

		// --is-bool flag tests
		{
			name: "is-bool passes with true",
			check: Check{
				Name:   "FLAG",
				IsBool: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"FLAG": "true"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-bool passes with false",
			check: Check{
				Name:   "FLAG",
				IsBool: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"FLAG": "false"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-bool passes with 1",
			check: Check{
				Name:   "FLAG",
				IsBool: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"FLAG": "1"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-bool passes with yes",
			check: Check{
				Name:   "FLAG",
				IsBool: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"FLAG": "yes"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-bool passes with on (case insensitive)",
			check: Check{
				Name:   "FLAG",
				IsBool: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"FLAG": "ON"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "is-bool fails with invalid value",
			check: Check{
				Name:   "FLAG",
				IsBool: true,
				Getter: &mockEnvGetter{Vars: map[string]string{"FLAG": "maybe"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not a valid boolean (true/false/1/0/yes/no/on/off)",
		},

		// --min-value flag tests
		{
			name: "min-value passes",
			check: Check{
				Name:     "COUNT",
				MinValue: float64Ptr(10),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "15"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "min-value passes at boundary",
			check: Check{
				Name:     "COUNT",
				MinValue: float64Ptr(10),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "10"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "min-value fails below minimum",
			check: Check{
				Name:     "COUNT",
				MinValue: float64Ptr(10),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "5"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value 5 < minimum 10",
		},
		{
			name: "min-value fails with non-numeric",
			check: Check{
				Name:     "COUNT",
				MinValue: float64Ptr(10),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "abc"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value is not numeric (required for --min-value)",
		},

		// --max-value flag tests
		{
			name: "max-value passes",
			check: Check{
				Name:     "COUNT",
				MaxValue: float64Ptr(100),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "50"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "max-value passes at boundary",
			check: Check{
				Name:     "COUNT",
				MaxValue: float64Ptr(100),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "100"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "max-value fails above maximum",
			check: Check{
				Name:     "COUNT",
				MaxValue: float64Ptr(100),
				Getter:   &mockEnvGetter{Vars: map[string]string{"COUNT": "150"}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "value 150 > maximum 100",
		},

		// combined min-value and max-value
		{
			name: "min-value and max-value passes in range",
			check: Check{
				Name:     "PORT",
				MinValue: float64Ptr(1024),
				MaxValue: float64Ptr(65535),
				Getter:   &mockEnvGetter{Vars: map[string]string{"PORT": "8080"}},
			},
			wantStatus: check.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.check.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", result.Status, tt.wantStatus)
			}

			if tt.wantDetail != "" {
				found := false
				for _, d := range result.Details {
					if d == tt.wantDetail {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected detail %q not found in %v", tt.wantDetail, result.Details)
				}
			}
		})
	}
}
