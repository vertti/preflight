package jsoncheck

import (
	"encoding/json"
	"fmt"
	"strings"

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

	// Parse JSON
	var data any
	if err := json.Unmarshal(content, &data); err != nil {
		return result.Fail("invalid JSON", err)
	}

	result.AddDetail("syntax: valid")

	// --has-key: check key exists
	if c.HasKey != "" {
		if _, found := getByDotPath(data, c.HasKey); !found {
			return result.Failf("key %q not found", c.HasKey)
		}
		result.AddDetailf("has key: %s", c.HasKey)
	}

	// --key: check value
	if c.Key != "" {
		value, found := getByDotPath(data, c.Key)
		if !found {
			return result.Failf("key %q not found", c.Key)
		}

		valueStr := toString(value)

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

// getByDotPath traverses a nested structure using dot notation.
// Returns the value and whether it was found.
func getByDotPath(data any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		val, exists := m[part]
		if !exists {
			return nil, false
		}
		current = val
	}

	return current, true
}

// toString converts a value to its string representation.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		// JSON numbers are float64
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case nil:
		return "null"
	default:
		// For arrays/objects, use JSON representation
		b, _ := json.Marshal(val)
		return string(b)
	}
}
