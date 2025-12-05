package check

import (
	"fmt"
	"regexp"
)

// Fail sets the result to failed status with a detail message.
func (r *Result) Fail(detail string, err error) Result {
	r.Status = StatusFail
	r.Details = append(r.Details, detail)
	r.Err = err
	return *r
}

// Failf sets the result to failed status with a formatted detail message.
func (r *Result) Failf(format string, args ...interface{}) Result {
	return r.Fail(fmt.Sprintf(format, args...), fmt.Errorf(format, args...))
}

// AddDetail appends a detail line to the result.
func (r *Result) AddDetail(detail string) *Result {
	r.Details = append(r.Details, detail)
	return r
}

// AddDetailf appends a formatted detail line to the result.
func (r *Result) AddDetailf(format string, args ...interface{}) *Result {
	return r.AddDetail(fmt.Sprintf(format, args...))
}

// CompileRegex compiles a regex pattern if non-empty, returning nil if pattern is empty.
// This provides a consistent pattern for optional regex compilation across check packages.
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil
	}
	return regexp.Compile(pattern)
}
