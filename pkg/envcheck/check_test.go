package envcheck

import (
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

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
