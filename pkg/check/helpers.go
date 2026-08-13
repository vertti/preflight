package check

import (
	"fmt"
	"regexp"
	"slices"
)

// Fail sets the result to failed status with a detail message.
func (r *Result) Fail(detail string, err error) Result {
	r.Status = StatusFail
	r.Details = append(r.Details, detail)
	r.Err = err
	return *r
}

// Failf sets the result to failed status with a formatted detail message.
func (r *Result) Failf(format string, args ...any) Result {
	return r.Fail(fmt.Sprintf(format, args...), fmt.Errorf(format, args...))
}

// FailAfter is Failf for a check that was allowed to retry: it records how many
// attempts were made, when more than one was.
//
// The wording lives here rather than at each call site because it used to be
// spelled out at eleven of them across two files, where nothing kept the count
// and the message together — and two had drifted into a different phrasing.
func (r *Result) FailAfter(attempts int, format string, args ...any) Result {
	if attempts <= 1 {
		return r.Failf(format, args...)
	}
	return r.Failf(format+" (after %d attempts)", append(slices.Clone(args), attempts)...)
}

// AddDetail appends a detail line to the result.
func (r *Result) AddDetail(detail string) *Result {
	r.Details = append(r.Details, detail)
	return r
}

// AddDetailf appends a formatted detail line to the result.
func (r *Result) AddDetailf(format string, args ...any) *Result {
	return r.AddDetail(fmt.Sprintf(format, args...))
}

// CompileRegex compiles a regex pattern if non-empty, returning nil if pattern is empty.
// This provides a consistent pattern for optional regex compilation across check packages.
func CompileRegex(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, nil //nolint:nilnil // intentional: empty pattern means "no regex"
	}
	return regexp.Compile(pattern)
}
