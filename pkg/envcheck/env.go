package envcheck

import "os"

type EnvGetter interface {
	LookupEnv(key string) (string, bool)
}

type RealEnvGetter struct{}

func (r *RealEnvGetter) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}
