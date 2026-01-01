package usercheck

import (
	"errors"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vertti/preflight/pkg/check"
)

type mockUserLookup struct {
	LookupFunc func(username string) (*user.User, error)
}

func (m *mockUserLookup) Lookup(username string) (*user.User, error) {
	return m.LookupFunc(username)
}

func TestUserCheck(t *testing.T) {
	appUser := &user.User{Uid: "1000", Gid: "1000", HomeDir: "/home/appuser"}
	appUserCustomHome := &user.User{Uid: "1000", Gid: "1000", HomeDir: "/app"}

	tests := []struct {
		name       string
		username   string
		uid        string
		gid        string
		home       string
		lookupFunc func(string) (*user.User, error)
		wantStatus check.Status
	}{
		{"user exists", "appuser", "", "", "", func(string) (*user.User, error) { return appUser, nil }, check.StatusOK},
		{"user not found", "nonexistent", "", "", "", func(string) (*user.User, error) { return nil, errors.New("user: unknown") }, check.StatusFail},
		{"uid matches", "appuser", "1000", "", "", func(string) (*user.User, error) { return appUser, nil }, check.StatusOK},
		{"uid mismatch", "appuser", "1001", "", "", func(string) (*user.User, error) { return appUser, nil }, check.StatusFail},
		{"gid matches", "appuser", "", "1000", "", func(string) (*user.User, error) { return appUser, nil }, check.StatusOK},
		{"gid mismatch", "appuser", "", "1001", "", func(string) (*user.User, error) { return appUser, nil }, check.StatusFail},
		{"home matches", "appuser", "", "", "/app", func(string) (*user.User, error) { return appUserCustomHome, nil }, check.StatusOK},
		{"home mismatch", "appuser", "", "", "/app", func(string) (*user.User, error) { return appUser, nil }, check.StatusFail},
		{"all constraints match", "appuser", "1000", "1000", "/app", func(string) (*user.User, error) { return appUserCustomHome, nil }, check.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Check{
				Username: tt.username,
				UID:      tt.uid,
				GID:      tt.gid,
				Home:     tt.home,
				Lookup:   &mockUserLookup{LookupFunc: tt.lookupFunc},
			}

			result := c.Run()

			assert.Equal(t, tt.wantStatus, result.Status)
			assert.Equal(t, "user: "+tt.username, result.Name)
			if tt.wantStatus == check.StatusFail {
				assert.Error(t, result.Err)
			}
		})
	}
}
