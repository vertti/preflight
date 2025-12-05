package main

import (
	"fmt"
	"strings"
)

// flagValue represents a flag name and its current value for validation.
type flagValue struct {
	name  string
	value string
}

// flagSet represents a flag that is either set (true) or not set (false).
type flagSet struct {
	name  string
	isSet bool
}

// requireExactlyOne returns an error if not exactly one of the given flags is set (non-empty).
func requireExactlyOne(flags ...flagValue) error {
	var set []string
	for _, f := range flags {
		if f.value != "" {
			set = append(set, f.name)
		}
	}

	names := make([]string, len(flags))
	for i, f := range flags {
		names[i] = f.name
	}
	flagList := strings.Join(names, ", ")

	if len(set) == 0 {
		return fmt.Errorf("one of %s is required", flagList)
	}
	if len(set) > 1 {
		return fmt.Errorf("only one of %s can be specified", flagList)
	}
	return nil
}

// requireAtLeastOne returns an error if none of the given flags are set.
func requireAtLeastOne(flags ...flagSet) error {
	for _, f := range flags {
		if f.isSet {
			return nil
		}
	}

	names := make([]string, len(flags))
	for i, f := range flags {
		names[i] = f.name
	}
	flagList := strings.Join(names, ", ")
	return fmt.Errorf("at least one of %s is required", flagList)
}
