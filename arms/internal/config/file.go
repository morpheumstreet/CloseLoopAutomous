package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// ReadConfigFileMap reads a flat JSON or TOML file into uppercase env-style keys.
// Values may be strings, numbers, or booleans; they are normalized to strings for the shared loader.
func ReadConfigFileMap(path string) (map[string]string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}
	ext := strings.ToLower(filepath.Ext(path))
	var raw map[string]interface{}
	switch ext {
	case ".json":
		if err := json.Unmarshal(b, &raw); err != nil {
			return nil, fmt.Errorf("config json: %w", err)
		}
	case ".toml":
		if err := toml.Unmarshal(b, &raw); err != nil {
			return nil, fmt.Errorf("config toml: %w", err)
	}
	default:
		return nil, fmt.Errorf("config file: use extension .json or .toml (got %q)", ext)
	}
	return normalizeFileMap(raw), nil
}

func normalizeFileMap(m map[string]interface{}) map[string]string {
	out := make(map[string]string)
	for k, v := range m {
		key := strings.ToUpper(strings.TrimSpace(k))
		key = strings.ReplaceAll(key, "-", "_")
		if key == "" || v == nil {
			continue
		}
		out[key] = stringifyFileValue(v)
	}
	return out
}

func stringifyFileValue(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(x))
	}
}

// Load reads optional config from path (JSON or TOML), then applies the process environment on top
// (environment variables override file values). An empty path is equivalent to [LoadFromEnv].
func Load(configPath string) (Config, error) {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return buildConfig(nil), nil
	}
	m, err := ReadConfigFileMap(configPath)
	if err != nil {
		return Config{}, err
	}
	return buildConfig(m), nil
}
