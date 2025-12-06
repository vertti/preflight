package jsoncheck

import (
	"fmt"

	"github.com/tidwall/gjson"

	"github.com/vertti/preflight/pkg/check"
)

// Check verifies that a JSON file is valid and optionally checks key/value assertions.
type Check struct {
	File   string     // path to JSON file
	HasKey string     // --has-key: check key exists (dot notation)
	Key    string     // --key: key to check value of
	Exact  string     // --exact: expected exact value (requires --key)
	Match  string     // --match: regex pattern for value (requires --key)
	FS     FileSystem // injected for testing
}

// Run executes the JSON check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("json: %s", c.File),
	}

	// Read file
	content, err := c.FS.ReadFile(c.File)
	if err != nil {
		return result.Failf("failed to read file: %v", err)
	}

	// Validate JSON syntax
	jsonStr := string(content)
	if !gjson.Valid(jsonStr) {
		return result.Fail("invalid JSON", fmt.Errorf("invalid JSON syntax"))
	}

	result.AddDetail("syntax: valid")

	// --has-key: check key exists
	if c.HasKey != "" {
		if !gjson.Get(jsonStr, c.HasKey).Exists() {
			return result.Failf("key %q not found", c.HasKey)
		}
		result.AddDetailf("has key: %s", c.HasKey)
	}

	// --key: check value
	if c.Key != "" {
		jsonResult := gjson.Get(jsonStr, c.Key)
		if !jsonResult.Exists() {
			return result.Failf("key %q not found", c.Key)
		}

		// Use String() for most values, but handle null specially
		valueStr := jsonResult.String()
		if jsonResult.Type == gjson.Null {
			valueStr = "null"
		}

		// --exact: exact value match
		if c.Exact != "" && valueStr != c.Exact {
			return result.Failf("value %q does not equal %q", valueStr, c.Exact)
		}

		// --match: regex pattern
		if c.Match != "" {
			re, err := check.CompileRegex(c.Match)
			if err != nil {
				return result.Failf("invalid regex pattern: %v", err)
			}
			if !re.MatchString(valueStr) {
				return result.Failf("value %q does not match pattern %q", valueStr, c.Match)
			}
		}

		result.AddDetailf("key %s: %s", c.Key, valueStr)
	}

	result.Status = check.StatusOK
	return result
}
