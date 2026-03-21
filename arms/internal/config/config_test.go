package config

import (
	"testing"
	"time"
)

func TestLoadFromEnvDefaults(t *testing.T) {
	t.Setenv("ARMS_LISTEN", "")
	t.Setenv("MC_API_TOKEN", "")
	t.Setenv("OPENCLAW_DISPATCH_TIMEOUT_SEC", "")
	c := LoadFromEnv()
	if c.ListenAddr != ":8080" {
		t.Fatalf("ListenAddr %q", c.ListenAddr)
	}
	if c.OpenClawDispatchTimeout != 30*time.Second {
		t.Fatalf("timeout %v", c.OpenClawDispatchTimeout)
	}
	if !c.AccessLog || c.LogJSON {
		t.Fatalf("default log flags %+v", c)
	}
}

func TestLoadFromEnvOverrides(t *testing.T) {
	t.Setenv("ARMS_LISTEN", ":9999")
	t.Setenv("OPENCLAW_DISPATCH_TIMEOUT_SEC", "60")
	t.Setenv("ARMS_DB_BACKUP", "1")
	t.Setenv("ARMS_LOG_JSON", "1")
	t.Setenv("ARMS_ACCESS_LOG", "0")
	c := LoadFromEnv()
	if c.ListenAddr != ":9999" || c.OpenClawDispatchTimeout != 60*time.Second || !c.DatabaseBackupBeforeMigrate {
		t.Fatalf("%+v", c)
	}
	if !c.LogJSON || c.AccessLog {
		t.Fatalf("log flags %+v", c)
	}
}
