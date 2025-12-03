package envcheck

import (
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

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
				Getter: &MockEnvGetter{Vars: map[string]string{}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "not set",
		},
		{
			name: "empty variable fails by default",
			check: Check{
				Name:   "EMPTY_VAR",
				Getter: &MockEnvGetter{Vars: map[string]string{"EMPTY_VAR": ""}},
			},
			wantStatus: check.StatusFail,
			wantDetail: "empty value",
		},
		{
			name: "non-empty variable passes",
			check: Check{
				Name:   "MY_VAR",
				Getter: &MockEnvGetter{Vars: map[string]string{"MY_VAR": "some_value"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: some_value",
		},

		// --required flag tests
		{
			name: "required allows empty value",
			check: Check{
				Name:     "EMPTY_VAR",
				Required: true,
				Getter:   &MockEnvGetter{Vars: map[string]string{"EMPTY_VAR": ""}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "required still fails if undefined",
			check: Check{
				Name:     "MISSING_VAR",
				Required: true,
				Getter:   &MockEnvGetter{Vars: map[string]string{}},
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
				Getter: &MockEnvGetter{Vars: map[string]string{"DATABASE_URL": "postgres://localhost:5432/db"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "match pattern fails",
			check: Check{
				Name:   "DATABASE_URL",
				Match:  `^postgres://`,
				Getter: &MockEnvGetter{Vars: map[string]string{"DATABASE_URL": "mysql://localhost:3306/db"}},
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
				Getter: &MockEnvGetter{Vars: map[string]string{"NODE_ENV": "production"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "exact value fails",
			check: Check{
				Name:   "NODE_ENV",
				Exact:  "production",
				Getter: &MockEnvGetter{Vars: map[string]string{"NODE_ENV": "development"}},
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
				Getter: &MockEnvGetter{Vars: map[string]string{"NODE_ENV": "production"}},
			},
			wantStatus: check.StatusOK,
		},
		{
			name: "one-of fails",
			check: Check{
				Name:   "NODE_ENV",
				OneOf:  []string{"dev", "staging", "production"},
				Getter: &MockEnvGetter{Vars: map[string]string{"NODE_ENV": "test"}},
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
				Getter:    &MockEnvGetter{Vars: map[string]string{"SECRET_ENV": "x"}},
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
				Getter:    &MockEnvGetter{Vars: map[string]string{"SECRET": "super-secret"}},
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
				Getter:    &MockEnvGetter{Vars: map[string]string{"API_KEY": "sk-secret-key-xyz"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: sk-•••xyz",
		},
		{
			name: "mask-value fully masks short value",
			check: Check{
				Name:      "SHORT",
				MaskValue: true,
				Getter:    &MockEnvGetter{Vars: map[string]string{"SHORT": "abc"}},
			},
			wantStatus: check.StatusOK,
			wantDetail: "value: •••",
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
