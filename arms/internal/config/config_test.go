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
	if c.GatewayDispatchTimeout != 30*time.Second {
		t.Fatalf("timeout %v", c.GatewayDispatchTimeout)
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
	if c.ListenAddr != ":9999" || c.GatewayDispatchTimeout != 60*time.Second || !c.DatabaseBackupBeforeMigrate {
		t.Fatalf("%+v", c)
	}
	if !c.LogJSON || c.AccessLog {
		t.Fatalf("log flags %+v", c)
	}
}

func TestLoadFromEnvUseAsynqScheduler(t *testing.T) {
	t.Setenv("ARMS_USE_ASYNQ_SCHEDULER", "true")
	c := LoadFromEnv()
	if !c.UseAsynqScheduler {
		t.Fatalf("UseAsynqScheduler want true got %+v", c)
	}
	t.Setenv("ARMS_USE_ASYNQ_SCHEDULER", "0")
	c = LoadFromEnv()
	if c.UseAsynqScheduler {
		t.Fatalf("UseAsynqScheduler want false got %+v", c)
	}
}

func TestLoadFromEnvARMSACL(t *testing.T) {
	t.Setenv("ARMS_ACL", "alice|s1|admin;bob|s2|read")
	c := LoadFromEnv()
	if len(c.ACLUsers) != 2 {
		t.Fatalf("want 2 users got %+v", c.ACLUsers)
	}
	if c.ACLUsers[0].UserID != "alice" || c.ACLUsers[0].Role != "admin" {
		t.Fatalf("user0 %+v", c.ACLUsers[0])
	}
	if c.ACLUsers[1].UserID != "bob" || c.ACLUsers[1].Role != "read" {
		t.Fatalf("user1 %+v", c.ACLUsers[1])
	}
}
