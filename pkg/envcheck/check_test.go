package envcheck

import (
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

func TestEnvCheck_Default_Undefined(t *testing.T) {
	c := &Check{
		Name:   "MISSING_VAR",
		Getter: &MockEnvGetter{Vars: map[string]string{}},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
}

func TestEnvCheck_Default_Empty(t *testing.T) {
	c := &Check{
		Name:   "EMPTY_VAR",
		Getter: &MockEnvGetter{Vars: map[string]string{"EMPTY_VAR": ""}},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
}

func TestEnvCheck_Default_NonEmpty(t *testing.T) {
	c := &Check{
		Name:   "MY_VAR",
		Getter: &MockEnvGetter{Vars: map[string]string{"MY_VAR": "some_value"}},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusOK)
	}
}

func TestEnvCheck_Required_Empty(t *testing.T) {
	c := &Check{
		Name:     "EMPTY_VAR",
		Required: true,
		Getter:   &MockEnvGetter{Vars: map[string]string{"EMPTY_VAR": ""}},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v (--required allows empty)", result.Status, check.StatusOK)
	}
}

func TestEnvCheck_Required_Undefined(t *testing.T) {
	c := &Check{
		Name:     "MISSING_VAR",
		Required: true,
		Getter:   &MockEnvGetter{Vars: map[string]string{}},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
}

func TestEnvCheck_Match_Pass(t *testing.T) {
	c := &Check{
		Name:   "DATABASE_URL",
		Match:  `^postgres://`,
		Getter: &MockEnvGetter{Vars: map[string]string{"DATABASE_URL": "postgres://localhost:5432/db"}},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusOK)
	}
}

func TestEnvCheck_Match_Fail(t *testing.T) {
	c := &Check{
		Name:   "DATABASE_URL",
		Match:  `^postgres://`,
		Getter: &MockEnvGetter{Vars: map[string]string{"DATABASE_URL": "mysql://localhost:3306/db"}},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
}

func TestEnvCheck_Exact_Pass(t *testing.T) {
	c := &Check{
		Name:   "NODE_ENV",
		Exact:  "production",
		Getter: &MockEnvGetter{Vars: map[string]string{"NODE_ENV": "production"}},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusOK)
	}
}

func TestEnvCheck_Exact_Fail(t *testing.T) {
	c := &Check{
		Name:   "NODE_ENV",
		Exact:  "production",
		Getter: &MockEnvGetter{Vars: map[string]string{"NODE_ENV": "development"}},
	}

	result := c.Run()

	if result.Status != check.StatusFail {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusFail)
	}
}

func TestEnvCheck_MaskValue(t *testing.T) {
	c := &Check{
		Name:      "API_KEY",
		MaskValue: true,
		Getter:    &MockEnvGetter{Vars: map[string]string{"API_KEY": "sk-secret-key-xyz"}},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusOK)
	}

	// Check that value is masked in details
	found := false
	for _, d := range result.Details {
		if d == "value: sk-•••xyz" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected masked value in details, got: %v", result.Details)
	}
}

func TestEnvCheck_HideValue(t *testing.T) {
	c := &Check{
		Name:      "SECRET",
		HideValue: true,
		Getter:    &MockEnvGetter{Vars: map[string]string{"SECRET": "super-secret"}},
	}

	result := c.Run()

	if result.Status != check.StatusOK {
		t.Errorf("Status = %v, want %v", result.Status, check.StatusOK)
	}

	// Check that value is hidden in details
	found := false
	for _, d := range result.Details {
		if d == "value: [hidden]" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected hidden value in details, got: %v", result.Details)
	}
}

func TestEnvCheck_MaskValue_Short(t *testing.T) {
	c := &Check{
		Name:      "SHORT",
		MaskValue: true,
		Getter:    &MockEnvGetter{Vars: map[string]string{"SHORT": "abc"}},
	}

	result := c.Run()

	// Short values should be fully masked
	found := false
	for _, d := range result.Details {
		if d == "value: •••" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected fully masked short value, got: %v", result.Details)
	}
}
