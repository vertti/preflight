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

	// jsonpath.Valid answers yes or no; Unmarshal is what reports where the
	// document went wrong, which is the only thing worth telling the user.
	jsonStr := string(content)
	if !jsonpath.Valid(jsonStr) {
		var v any
		if err := json.Unmarshal(content, &v); err != nil {
			return result.Failf("invalid JSON: %v", err)
		}
		// Both are encoding/json underneath, so disagreeing would be a bug in it.
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

		valueStr := jsonResult.String()

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
