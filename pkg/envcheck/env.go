package envcheck

import "os"

// EnvGetter abstracts environment variable access for testability.
type EnvGetter interface {
	LookupEnv(key string) (string, bool)
}

// RealEnvGetter implements EnvGetter using the actual environment.
type RealEnvGetter struct{}

func (r *RealEnvGetter) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// MockEnvGetter is a test double for EnvGetter.
type MockEnvGetter struct {
	Vars map[string]string
}

func (m *MockEnvGetter) LookupEnv(key string) (string, bool) {
	val, ok := m.Vars[key]
	return val, ok
}
