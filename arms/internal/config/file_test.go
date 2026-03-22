package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigFileMapJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(p, []byte(`{"ARMS_LISTEN": ":7070", "DATABASE_PATH": "/tmp/x.db"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := ReadConfigFileMap(p)
	if err != nil {
		t.Fatal(err)
	}
	if m["ARMS_LISTEN"] != ":7070" || m["DATABASE_PATH"] != "/tmp/x.db" {
		t.Fatalf("%v", m)
	}
}

func TestReadConfigFileMapTOML(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.toml")
	if err := os.WriteFile(p, []byte("ARMS_LISTEN = ':7070'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := ReadConfigFileMap(p)
	if err != nil {
		t.Fatal(err)
	}
	if m["ARMS_LISTEN"] != ":7070" {
		t.Fatalf("%v", m)
	}
}

func TestLoadEnvOverridesFile(t *testing.T) {
	t.Setenv("ARMS_LISTEN", ":6060")
	t.Setenv("DATABASE_PATH", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.toml")
	if err := os.WriteFile(p, []byte(`ARMS_LISTEN = ":7070"`), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddr != ":6060" {
		t.Fatalf("env should win: got %q", c.ListenAddr)
	}
}

func TestLoadEmptyPathSameAsEnvOnly(t *testing.T) {
	t.Setenv("ARMS_LISTEN", ":5555")
	c, err := Load("")
	if err != nil {
		t.Fatal(err)
	}
	if c.ListenAddr != ":5555" {
		t.Fatalf("%q", c.ListenAddr)
	}
}

func TestLoadUnsupportedExt(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	if err := os.WriteFile(p, []byte(`a: 1`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	if err == nil {
		t.Fatal("want error for .yaml")
	}
}
