package usercheck

import (
	"fmt"
	"os/user"

	"github.com/vertti/preflight/pkg/check"
)

// UserLookup abstracts user lookup for testability.
type UserLookup interface {
	Lookup(username string) (*user.User, error)
}

// RealUserLookup uses the real os/user package.
type RealUserLookup struct{}

// Lookup looks up a user by username.
func (r *RealUserLookup) Lookup(username string) (*user.User, error) {
	return user.Lookup(username)
}

// Check verifies a user exists and optionally validates uid/gid/home.
type Check struct {
	Username string     // username to check
	UID      string     // expected uid (empty = don't check)
	GID      string     // expected gid (empty = don't check)
	Home     string     // expected home directory (empty = don't check)
	Lookup   UserLookup // injected for testing
}

// Run executes the user check.
func (c *Check) Run() check.Result {
	result := check.Result{
		Name: fmt.Sprintf("user: %s", c.Username),
	}

	u, err := c.Lookup.Lookup(c.Username)
	if err != nil {
		return result.Failf("user not found: %v", err)
	}

	result.AddDetailf("uid: %s", u.Uid)
	result.AddDetailf("gid: %s", u.Gid)
	result.AddDetailf("home: %s", u.HomeDir)

	checks := []struct {
		name     string
		expected string
		actual   string
	}{
		{"uid", c.UID, u.Uid},
		{"gid", c.GID, u.Gid},
		{"home", c.Home, u.HomeDir},
	}

	for _, chk := range checks {
		if chk.expected != "" && chk.actual != chk.expected {
			return result.Fail(
				fmt.Sprintf("%s mismatch: expected %s, got %s", chk.name, chk.expected, chk.actual),
				fmt.Errorf("%s mismatch", chk.name))
		}
	}

	result.Status = check.StatusOK
	return result
}
