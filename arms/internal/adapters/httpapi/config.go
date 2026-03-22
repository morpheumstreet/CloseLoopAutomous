package httpapi

import "github.com/closeloopautomous/arms/internal/config"

// Config is the shared arms configuration (loaded via [config.LoadFromEnv] or [config.Load] with -c).
type Config = config.Config

// LoadConfig loads [Config] from the environment. Prefer [config.Load] when using a config file.
func LoadConfig() Config {
	return config.LoadFromEnv()
}
