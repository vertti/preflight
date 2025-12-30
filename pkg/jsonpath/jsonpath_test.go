package jsonpath

import (
	"testing"
)

func TestValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"empty object", "{}", true},
		{"simple object", `{"key": "value"}`, true},
		{"nested object", `{"data": {"result": 42}}`, true},
		{"array", `[1, 2, 3]`, true},
		{"string", `"hello"`, true},
		{"number", `42`, true},
		{"boolean true", `true`, true},
		{"boolean false", `false`, true},
		{"null", `null`, true},
		{"empty string", "", false},
		{"invalid json", `{invalid}`, false},
		{"unclosed brace", `{"key": "value"`, false},
		{"trailing comma", `{"key": "value",}`, false},
		{"single quotes", `{'key': 'value'}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Valid(tt.input)
			if got != tt.want {
				t.Errorf("Valid(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGet_SimpleKeys(t *testing.T) {
	json := `{"status": "success", "error": "", "count": 42, "active": true, "data": null}`

	tests := []struct {
		name       string
		path       string
		wantExists bool
		wantString string
		wantNull   bool
	}{
		{"existing string key", "status", true, "success", false},
		{"empty string value", "error", true, "", false},
		{"number as string", "count", true, "42", false},
		{"boolean as string", "active", true, "true", false},
		{"null value", "data", true, "null", true},
		{"missing key", "missing", false, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Get(json, tt.path)
			if result.Exists() != tt.wantExists {
				t.Errorf("Get(%q).Exists() = %v, want %v", tt.path, result.Exists(), tt.wantExists)
			}
			if result.String() != tt.wantString {
				t.Errorf("Get(%q).String() = %q, want %q", tt.path, result.String(), tt.wantString)
			}
			if result.IsNull() != tt.wantNull {
				t.Errorf("Get(%q).IsNull() = %v, want %v", tt.path, result.IsNull(), tt.wantNull)
			}
		})
	}
}

func TestGet_NestedPaths(t *testing.T) {
	json := `{
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {"job": "prometheus"},
					"value": [1234567890, "42.5"]
				}
			]
		}
	}`

	tests := []struct {
		name       string
		path       string
		wantExists bool
		wantString string
	}{
		{"one level nested", "data.resultType", true, "vector"},
		{"deep nested object", "data.result.0.metric.job", true, "prometheus"},
		{"array index", "data.result.0.value.1", true, "42.5"},
		{"missing nested key", "data.missing", false, ""},
		{"missing deep key", "data.result.0.missing", false, ""},
		{"array index out of bounds", "data.result.99", false, ""},
		{"negative array index", "data.result.-1", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Get(json, tt.path)
			if result.Exists() != tt.wantExists {
				t.Errorf("Get(%q).Exists() = %v, want %v", tt.path, result.Exists(), tt.wantExists)
			}
			if result.String() != tt.wantString {
				t.Errorf("Get(%q).String() = %q, want %q", tt.path, result.String(), tt.wantString)
			}
		})
	}
}

func TestGet_Array(t *testing.T) {
	json := `{"data": {"result": [{"id": 1}, {"id": 2}, {"id": 3}]}}`

	result := Get(json, "data.result")
	if !result.Exists() {
		t.Fatal("Get(data.result).Exists() = false, want true")
	}

	arr := result.Array()
	if len(arr) != 3 {
		t.Fatalf("len(Array()) = %d, want 3", len(arr))
	}

	// Each element should be accessible
	for i, elem := range arr {
		if !elem.Exists() {
			t.Errorf("Array()[%d].Exists() = false, want true", i)
		}
	}
}

func TestGet_EmptyArray(t *testing.T) {
	json := `{"data": {"result": []}}`

	result := Get(json, "data.result")
	if !result.Exists() {
		t.Fatal("Get(data.result).Exists() = false, want true")
	}

	arr := result.Array()
	if len(arr) != 0 {
		t.Errorf("len(Array()) = %d, want 0", len(arr))
	}
}

func TestGet_NonArrayAsArray(t *testing.T) {
	json := `{"value": "string"}`

	result := Get(json, "value")
	arr := result.Array()
	if arr != nil {
		t.Errorf("Array() on non-array = %v, want nil", arr)
	}
}

func TestGet_InvalidJSON(t *testing.T) {
	result := Get("{invalid}", "key")
	if result.Exists() {
		t.Error("Get on invalid JSON should return non-existing result")
	}
	if result.String() != "" {
		t.Error("String() on invalid JSON should return empty string")
	}
}

func TestGet_ObjectAsString(t *testing.T) {
	// When getting an object, String() should return JSON representation
	json := `{"metric": {"job": "prometheus", "instance": "localhost:9090"}}`

	result := Get(json, "metric")
	if !result.Exists() {
		t.Fatal("Get(metric).Exists() = false, want true")
	}

	// The string representation should be valid JSON
	s := result.String()
	if !Valid(s) {
		t.Errorf("Get(metric).String() = %q is not valid JSON", s)
	}
}

func TestGet_ArrayAsString(t *testing.T) {
	json := `{"items": [1, 2, 3]}`

	result := Get(json, "items")
	if !result.Exists() {
		t.Fatal("Get(items).Exists() = false, want true")
	}

	// The string representation should be valid JSON
	s := result.String()
	if !Valid(s) {
		t.Errorf("Get(items).String() = %q is not valid JSON", s)
	}
}

// Test cases from actual usage in promcheck
func TestGet_PrometheusResponse(t *testing.T) {
	// Simulated Prometheus API response
	json := `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {"__name__": "up", "instance": "localhost:9090", "job": "prometheus"},
					"value": [1703862000, "1"]
				}
			]
		}
	}`

	// Test status extraction
	status := Get(json, "status").String()
	if status != "success" {
		t.Errorf("status = %q, want %q", status, "success")
	}

	// Test resultType extraction
	resultType := Get(json, "data.resultType").String()
	if resultType != "vector" {
		t.Errorf("resultType = %q, want %q", resultType, "vector")
	}

	// Test results array
	results := Get(json, "data.result")
	if !results.Exists() {
		t.Fatal("data.result should exist")
	}
	if len(results.Array()) != 1 {
		t.Errorf("len(results) = %d, want 1", len(results.Array()))
	}

	// Test value extraction (vector format: [timestamp, "value"])
	value := Get(json, "data.result.0.value.1").String()
	if value != "1" {
		t.Errorf("value = %q, want %q", value, "1")
	}

	// Test metric labels (should be JSON object as string)
	metric := Get(json, "data.result.0.metric").String()
	if !Valid(metric) {
		t.Errorf("metric = %q is not valid JSON", metric)
	}
}

// Test scalar result type from Prometheus
func TestGet_PrometheusScalarResponse(t *testing.T) {
	json := `{
		"status": "success",
		"data": {
			"resultType": "scalar",
			"result": [1703862000, "42"]
		}
	}`

	resultType := Get(json, "data.resultType").String()
	if resultType != "scalar" {
		t.Errorf("resultType = %q, want %q", resultType, "scalar")
	}

	// Scalar result is array [timestamp, "value"]
	value := Get(json, "data.result.1").String()
	if value != "42" {
		t.Errorf("value = %q, want %q", value, "42")
	}
}

func TestGet_EmptyPath(t *testing.T) {
	json := `{"key": "value"}`

	result := Get(json, "")
	// Empty path should return the whole document
	if !result.Exists() {
		t.Error("Empty path should return existing result for valid JSON")
	}
}

func TestGet_FloatValues(t *testing.T) {
	json := `{"pi": 3.14159, "integer": 42, "zero": 0}`

	// Non-integer float should keep decimal representation
	pi := Get(json, "pi")
	if pi.String() != "3.14159" {
		t.Errorf("pi.String() = %q, want %q", pi.String(), "3.14159")
	}

	// Integer-like float should format without decimal
	integer := Get(json, "integer")
	if integer.String() != "42" {
		t.Errorf("integer.String() = %q, want %q", integer.String(), "42")
	}

	// Zero should format without decimal
	zero := Get(json, "zero")
	if zero.String() != "0" {
		t.Errorf("zero.String() = %q, want %q", zero.String(), "0")
	}
}

func TestResult_ArrayOnNonExisting(t *testing.T) {
	json := `{"key": "value"}`

	// Get a path that doesn't exist
	result := Get(json, "nonexistent")
	if result.Exists() {
		t.Fatal("result should not exist")
	}

	// Array() on non-existing should return nil
	arr := result.Array()
	if arr != nil {
		t.Errorf("Array() on non-existing = %v, want nil", arr)
	}
}
