// Package jsonpath provides simple JSON path querying.
// It supports dot-notation paths with array indices (e.g., "data.result.0.value.1").
package jsonpath

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Valid reports whether s is valid JSON.
func Valid(s string) bool {
	return json.Valid([]byte(s))
}

// Result represents a JSON value lookup result.
type Result struct {
	value  any
	exists bool
}

// Get retrieves a value at a dot-notation path.
// Supports object keys and array indices (e.g., "data.result.0.value").
func Get(jsonStr, path string) Result {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return Result{exists: false}
	}

	// Empty path returns the whole document
	if path == "" {
		return Result{value: data, exists: true}
	}

	parts := strings.Split(path, ".")
	current := data

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]any:
			val, ok := v[part]
			if !ok {
				return Result{exists: false}
			}
			current = val
		case []any:
			idx, err := strconv.Atoi(part)
			if err != nil || idx < 0 || idx >= len(v) {
				return Result{exists: false}
			}
			current = v[idx]
		default:
			return Result{exists: false}
		}
	}

	return Result{value: current, exists: true}
}

// Exists reports whether the path was found.
func (r Result) Exists() bool {
	return r.exists
}

// IsNull reports whether the value is JSON null.
func (r Result) IsNull() bool {
	return r.exists && r.value == nil
}

// String returns the value as a string.
// For objects and arrays, returns the JSON representation.
// For null, returns "null".
// For non-existing paths, returns "".
func (r Result) String() string {
	if !r.exists {
		return ""
	}

	switch v := r.value.(type) {
	case nil:
		return "null"
	case string:
		return v
	case float64:
		// Format without trailing zeros for integers
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case map[string]any, []any:
		// Return JSON representation for objects and arrays
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	default:
		return ""
	}
}

// Array returns the value as a slice of Results.
// Returns nil if the value is not an array.
func (r Result) Array() []Result {
	if !r.exists {
		return nil
	}

	arr, ok := r.value.([]any)
	if !ok {
		return nil
	}

	results := make([]Result, len(arr))
	for i, v := range arr {
		results[i] = Result{value: v, exists: true}
	}
	return results
}
