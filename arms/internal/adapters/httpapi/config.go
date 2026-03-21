package httpapi

import "github.com/closeloopautomous/arms/internal/config"

// Config is the shared arms configuration (loaded via [config.LoadFromEnv]).
type Config = config.Config

// LoadConfig loads [Config] from the environment. Prefer calling [config.LoadFromEnv] from non-HTTP entrypoints.
func LoadConfig() Config {
	return config.LoadFromEnv()
}
