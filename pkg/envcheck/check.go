package envcheck

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies that an environment variable meets requirements.
type Check struct {
	Name       string     // env var name
	NotSet     bool       // --not-set: verify variable is NOT defined
	AllowEmpty bool       // --allow-empty: pass if defined but empty
	Match      string     // --match: regex pattern
	Exact      string     // --exact: exact value
	OneOf      []string   // --one-of: value must be one of these
	HideValue  bool       // --hide-value: don't show value in output
	MaskValue  bool       // --mask-value: show first/last 3 chars
	StartsWith string     // --starts-with: value must start with this
	EndsWith   string     // --ends-with: value must end with this
	Contains   string     // --contains: value must contain this
	IsNumeric  bool       // --is-numeric: value must be a valid number
	IsPort     bool       // --is-port: value must be valid TCP port (1-65535)
	IsURL      bool       // --is-url: value must be valid URL
	IsJSON     bool       // --is-json: value must be valid JSON
	IsBool     bool       // --is-bool: value must be boolean (true/false/1/0/yes/no/on/off)
	IsFile     bool       // --is-file: value must be path to existing file
	IsDir      bool       // --is-dir: value must be path to existing directory
	MinLen     int        // --min-len: minimum string length (0 = no check)
	MaxLen     int        // --max-len: maximum string length (0 = no check)
	MinValue   *float64   // --min-value: minimum numeric value
	MaxValue   *float64   // --max-value: maximum numeric value
	Getter     EnvGetter  // injected for testing
	Stater     FileStater // injected for testing
}

// Run executes the environment variable check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: "env: " + c.Name,
	}

	value, exists := c.Getter.LookupEnv(c.Name)

	// --not-set: verify variable is NOT defined
	if c.NotSet {
		if exists {
			return result.Fail("variable is set (expected not set)", fmt.Errorf("environment variable %s is set", c.Name))
		}
		result.Status = check.StatusOK
		result.AddDetail("not set (as expected)")
		return result
	}

	if !exists {
		return result.Fail("not set", fmt.Errorf("environment variable %s is not set", c.Name))
	}

	// --allow-empty flag: pass if defined but empty
	// Default: require non-empty
	if !c.AllowEmpty && value == "" {
		return result.Fail("empty value", fmt.Errorf("environment variable %s is empty", c.Name))
	}

	// --match: regex pattern
	if c.Match != "" {
		re, err := check.CompileRegex(c.Match)
		if err != nil {
			return result.Failf("invalid regex pattern: %v", err)
		}
		if !re.MatchString(value) {
			return result.Failf("value does not match pattern %q", c.Match)
		}
	}

	// --exact: exact value match
	if c.Exact != "" && value != c.Exact {
		return result.Failf("value does not equal %q", c.Exact)
	}

	// --one-of: value must be one of the allowed values
	if len(c.OneOf) > 0 {
		if err := c.validateOneOf(value, &result); err != nil {
			return result
		}
	}

	// --starts-with: value must start with prefix
	if c.StartsWith != "" && !strings.HasPrefix(value, c.StartsWith) {
		return result.Failf("value does not start with %q", c.StartsWith)
	}

	// --ends-with: value must end with suffix
	if c.EndsWith != "" && !strings.HasSuffix(value, c.EndsWith) {
		return result.Failf("value does not end with %q", c.EndsWith)
	}

	// --contains: value must contain substring
	if c.Contains != "" && !strings.Contains(value, c.Contains) {
		return result.Failf("value does not contain %q", c.Contains)
	}

	// --is-numeric: value must be a valid number
	if c.IsNumeric {
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return result.Fail("value is not numeric", errors.New("value is not numeric"))
		}
	}

	// --is-port: value must be valid TCP port (1-65535)
	if c.IsPort {
		port, err := strconv.Atoi(value)
		if err != nil || port < 1 || port > 65535 {
			return result.Fail("value is not a valid port (1-65535)", fmt.Errorf("invalid port: %s", value))
		}
	}

	// --is-url: value must be valid URL
	if c.IsURL {
		u, err := url.Parse(value)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return result.Fail("value is not a valid URL", fmt.Errorf("invalid URL: %s", value))
		}
	}

	// --is-json: value must be valid JSON
	if c.IsJSON {
		if !gjson.Valid(value) {
			return result.Fail("value is not valid JSON", errors.New("invalid JSON"))
		}
	}

	// --is-bool: value must be boolean truthy value
	if c.IsBool {
		if !isValidBool(value) {
			return result.Fail("value is not a valid boolean (true/false/1/0/yes/no/on/off)", fmt.Errorf("invalid boolean: %s", value))
		}
	}

	// --is-file: value must be path to existing file
	if c.IsFile {
		info, err := c.Stater.Stat(value)
		if err != nil {
			return result.Fail("path does not exist", fmt.Errorf("path does not exist: %s", value))
		}
		if info.IsDir() {
			return result.Fail("path is a directory, not a file", fmt.Errorf("path is a directory: %s", value))
		}
	}

	// --is-dir: value must be path to existing directory
	if c.IsDir {
		info, err := c.Stater.Stat(value)
		if err != nil {
			return result.Fail("path does not exist", fmt.Errorf("path does not exist: %s", value))
		}
		if !info.IsDir() {
			return result.Fail("path is a file, not a directory", fmt.Errorf("path is not a directory: %s", value))
		}
	}

	// --min-value: minimum numeric value
	if c.MinValue != nil {
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return result.Fail("value is not numeric (required for --min-value)", errors.New("not numeric"))
		}
		if num < *c.MinValue {
			return result.Failf("value %v < minimum %v", num, *c.MinValue)
		}
	}

	// --max-value: maximum numeric value
	if c.MaxValue != nil {
		num, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return result.Fail("value is not numeric (required for --max-value)", errors.New("not numeric"))
		}
		if num > *c.MaxValue {
			return result.Failf("value %v > maximum %v", num, *c.MaxValue)
		}
	}

	// --min-len: minimum string length
	if c.MinLen > 0 && len(value) < c.MinLen {
		return result.Failf("value length %d < minimum %d", len(value), c.MinLen)
	}

	// --max-len: maximum string length
	if c.MaxLen > 0 && len(value) > c.MaxLen {
		return result.Failf("value length %d > maximum %d", len(value), c.MaxLen)
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

func (c *Check) validateOneOf(value string, result *check.Result) error {
	if slices.Contains(c.OneOf, value) {
		return nil
	}
	err := fmt.Errorf("value not in allowed list %v", c.OneOf)
	result.Fail(fmt.Sprintf("value %q not in allowed list %v", c.formatValue(value), c.OneOf), err)
	return err
}

// isValidBool checks if value is a valid boolean truthy/falsy string
func isValidBool(value string) bool {
	v := strings.ToLower(value)
	switch v {
	case "true", "false", "1", "0", "yes", "no", "on", "off":
		return true
	default:
		return false
	}
}
