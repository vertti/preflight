package usercheck

import (
	"errors"
	"os/user"
	"testing"

	"github.com/vertti/preflight/pkg/check"
)

// MockUserLookup is a mock implementation of UserLookup for testing.
type MockUserLookup struct {
	LookupFunc func(username string) (*user.User, error)
}

func (m *MockUserLookup) Lookup(username string) (*user.User, error) {
	return m.LookupFunc(username)
}

func TestUserCheck(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		uid        string
		gid        string
		home       string
		lookupFunc func(username string) (*user.User, error)
		wantStatus check.Status
		wantName   string
	}{
		{
			name:     "user exists",
			username: "appuser",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/home/appuser",
				}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "user: appuser",
		},
		{
			name:     "user not found",
			username: "nonexistent",
			lookupFunc: func(username string) (*user.User, error) {
				return nil, errors.New("user:  unknown user nonexistent")
			},
			wantStatus: check.StatusFail,
			wantName:   "user: nonexistent",
		},
		{
			name:     "uid matches",
			username: "appuser",
			uid:      "1000",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/home/appuser",
				}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "user: appuser",
		},
		{
			name:     "uid mismatch",
			username: "appuser",
			uid:      "1001",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/home/appuser",
				}, nil
			},
			wantStatus: check.StatusFail,
			wantName:   "user: appuser",
		},
		{
			name:     "gid matches",
			username: "appuser",
			gid:      "1000",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/home/appuser",
				}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "user: appuser",
		},
		{
			name:     "gid mismatch",
			username: "appuser",
			gid:      "1001",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/home/appuser",
				}, nil
			},
			wantStatus: check.StatusFail,
			wantName:   "user: appuser",
		},
		{
			name:     "home matches",
			username: "appuser",
			home:     "/app",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/app",
				}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "user: appuser",
		},
		{
			name:     "home mismatch",
			username: "appuser",
			home:     "/app",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/home/appuser",
				}, nil
			},
			wantStatus: check.StatusFail,
			wantName:   "user: appuser",
		},
		{
			name:     "all constraints match",
			username: "appuser",
			uid:      "1000",
			gid:      "1000",
			home:     "/app",
			lookupFunc: func(username string) (*user.User, error) {
				return &user.User{
					Uid:     "1000",
					Gid:     "1000",
					HomeDir: "/app",
				}, nil
			},
			wantStatus: check.StatusOK,
			wantName:   "user: appuser",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Check{
				Username: tt.username,
				UID:      tt.uid,
				GID:      tt.gid,
				Home:     tt.home,
				Lookup: &MockUserLookup{
					LookupFunc: tt.lookupFunc,
				},
			}

			result := c.Run()

			if result.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", result.Status, tt.wantStatus)
			}
			if result.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", result.Name, tt.wantName)
			}
			if tt.wantStatus == check.StatusFail && result.Err == nil {
				t.Error("expected Err to be set on failure")
			}
		})
	}
}
