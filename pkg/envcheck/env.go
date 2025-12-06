package envcheck

import "os"

type EnvGetter interface {
	LookupEnv(key string) (string, bool)
}

type RealEnvGetter struct{}

func (r *RealEnvGetter) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// FileStater provides file system stat operations for testing.
type FileStater interface {
	Stat(path string) (os.FileInfo, error)
}

// RealFileStater uses actual os.Stat.
type RealFileStater struct{}

func (r *RealFileStater) Stat(path string) (os.FileInfo, error) {
	return os.Stat(path)
}
