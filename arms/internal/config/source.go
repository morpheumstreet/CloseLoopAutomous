package config

import "os"

// layeredSource resolves each setting with OS environment first, then optional file map
// (both keyed by canonical env names, e.g. ARMS_LISTEN). Env always wins when the variable is set in the process environment.
type layeredSource struct {
	file map[string]string // uppercase keys, same names as environment variables
}

func (s *layeredSource) lookup(key string) (value string, ok bool) {
	if v, e := os.LookupEnv(key); e {
		return v, true
	}
	if s != nil && s.file != nil {
		if v, f := s.file[key]; f {
			return v, true
		}
	}
	return "", false
}

func (s *layeredSource) getenv(key string) string {
	v, _ := s.lookup(key)
	return v
}
