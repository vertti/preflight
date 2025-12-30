package jsoncheck

import (
	"encoding/json"

	"github.com/vertti/preflight/pkg/check"
	"github.com/vertti/preflight/pkg/jsonpath"
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
		Name: "json: " + c.File,
	}

	// Read file
	content, err := c.FS.ReadFile(c.File)
	if err != nil {
		return result.Failf("failed to read file: %v", err)
	}

	// Validate JSON syntax using encoding/json for detailed errors
	jsonStr := string(content)
	if !jsonpath.Valid(jsonStr) {
		// Use encoding/json to get detailed parse error
		var v any
		if err := json.Unmarshal(content, &v); err != nil {
			return result.Failf("invalid JSON: %v", err)
		}
		// Fallback if gjson says invalid but encoding/json passes (shouldn't happen)
		return result.Failf("invalid JSON")
	}

	result.AddDetail("syntax: valid")

	// --has-key: check key exists
	if c.HasKey != "" {
		if !jsonpath.Get(jsonStr, c.HasKey).Exists() {
			return result.Failf("key %q not found", c.HasKey)
		}
		result.AddDetailf("has key: %s", c.HasKey)
	}

	// --key: check value
	if c.Key != "" {
		jsonResult := jsonpath.Get(jsonStr, c.Key)
		if !jsonResult.Exists() {
			return result.Failf("key %q not found", c.Key)
		}

		// Use String() for most values, but handle null specially
		valueStr := jsonResult.String()
		if jsonResult.IsNull() {
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
