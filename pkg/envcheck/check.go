package envcheck

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies that an environment variable meets requirements.
type Check struct {
	Name       string    // env var name
	Required   bool      // --required: fail if undefined (allows empty)
	Match      string    // --match: regex pattern
	Exact      string    // --exact: exact value
	OneOf      []string  // --one-of: value must be one of these
	HideValue  bool      // --hide-value: don't show value in output
	MaskValue  bool      // --mask-value: show first/last 3 chars
	StartsWith string    // --starts-with: value must start with this
	EndsWith   string    // --ends-with: value must end with this
	Contains   string    // --contains: value must contain this
	IsNumeric  bool      // --is-numeric: value must be a valid number
	MinLen     int       // --min-len: minimum string length (0 = no check)
	MaxLen     int       // --max-len: maximum string length (0 = no check)
	Getter     EnvGetter // injected for testing
}

// Run executes the environment variable check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("env: %s", c.Name),
	}

	value, exists := c.Getter.LookupEnv(c.Name)

	if !exists {
		return *result.Fail("not set", fmt.Errorf("environment variable %s is not set", c.Name))
	}

	// --required flag: allow empty values
	// Default: require non-empty
	if !c.Required && value == "" {
		return *result.Fail("empty value", fmt.Errorf("environment variable %s is empty", c.Name))
	}

	// --match: regex pattern
	if c.Match != "" {
		re, err := regexp.Compile(c.Match)
		if err != nil {
			return *result.Failf("invalid regex pattern: %v", err)
		}
		if !re.MatchString(value) {
			return *result.Failf("value does not match pattern %q", c.Match)
		}
	}

	// --exact: exact value match
	if c.Exact != "" && value != c.Exact {
		return *result.Failf("value does not equal %q", c.Exact)
	}

	// --one-of: value must be one of the allowed values
	if len(c.OneOf) > 0 {
		found := false
		for _, allowed := range c.OneOf {
			if value == allowed {
				found = true
				break
			}
		}
		if !found {
			result.Fail(fmt.Sprintf("value %q not in allowed list %v", c.formatValue(value), c.OneOf),
				fmt.Errorf("value not in allowed list %v", c.OneOf))
			return result
		}
	}

	// --starts-with: value must start with prefix
	if c.StartsWith != "" && !strings.HasPrefix(value, c.StartsWith) {
		return *result.Failf("value does not start with %q", c.StartsWith)
	}

	// --ends-with: value must end with suffix
	if c.EndsWith != "" && !strings.HasSuffix(value, c.EndsWith) {
		return *result.Failf("value does not end with %q", c.EndsWith)
	}

	// --contains: value must contain substring
	if c.Contains != "" && !strings.Contains(value, c.Contains) {
		return *result.Failf("value does not contain %q", c.Contains)
	}

	// --is-numeric: value must be a valid number
	if c.IsNumeric {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return *result.Fail("value is not numeric", fmt.Errorf("value is not numeric"))
		}
	}

	// --min-len: minimum string length
	if c.MinLen > 0 && len(value) < c.MinLen {
		return *result.Failf("value length %d < minimum %d", len(value), c.MinLen)
	}

	// --max-len: maximum string length
	if c.MaxLen > 0 && len(value) > c.MaxLen {
		return *result.Failf("value length %d > maximum %d", len(value), c.MaxLen)
	}

	result.Status = check.StatusOK
	result.AddDetailf("value: %s", c.formatValue(value))
	return result
}

func (c *Check) formatValue(value string) string {
	if c.HideValue {
		return "[hidden]"
	}
	if c.MaskValue {
		return maskValue(value)
	}
	return value
}

func maskValue(value string) string {
	if len(value) <= 6 {
		return "•••"
	}
	return value[:3] + "•••" + value[len(value)-3:]
}
