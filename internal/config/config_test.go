package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.Search.Limit != 10 {
		t.Fatalf("limit = %d, want 10", cfg.Search.Limit)
	}
	if cfg.Search.Timeout != "45s" {
		t.Fatalf("timeout = %q, want 45s", cfg.Search.Timeout)
	}
	if len(cfg.Search.DefaultSourceIDs) != 3 {
		t.Fatalf("default sources = %#v", cfg.Search.DefaultSourceIDs)
	}
}

func TestEnvOverridesConfig(t *testing.T) {
	t.Setenv("BOCHA_API_KEY", "env-bocha")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("bocha:\n  api_key: file-bocha\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(Options{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bocha.APIKey != "env-bocha" {
		t.Fatalf("bocha key = %q", cfg.Bocha.APIKey)
	}
}

func TestConfigDoesNotExpandEnvPlaceholders(t *testing.T) {
	t.Setenv("BOCHA_API_KEY", "env-bocha")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("bocha:\n  api_key: ${BOCHA_API_KEY}\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(Options{Path: path, DisableEnv: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bocha.APIKey != "${BOCHA_API_KEY}" {
		t.Fatalf("placeholder expanded unexpectedly: %q", cfg.Bocha.APIKey)
	}
}

func TestRedactedConfig(t *testing.T) {
	cfg := Defaults()
	cfg.Bocha.APIKey = "secret"
	redacted := cfg.Redacted()
	if redacted.Bocha.APIKey != "***" {
		t.Fatalf("redacted key = %q", redacted.Bocha.APIKey)
	}
}
