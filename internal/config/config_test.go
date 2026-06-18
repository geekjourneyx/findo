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

func TestDefaultPathUsesUserConfigDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "findo", "config.yaml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestLoadReadsDefaultPathWhenPresent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "findo", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	err := os.WriteFile(path, []byte("search:\n  limit: 7\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(Options{DisableEnv: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Search.Limit != 7 {
		t.Fatalf("limit = %d, want 7", cfg.Search.Limit)
	}
}

func TestLoadIgnoresMissingDefaultPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load(Options{DisableEnv: true})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Search.Limit != 10 {
		t.Fatalf("limit = %d, want default 10", cfg.Search.Limit)
	}
}

func TestInitWritesDefaultConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "findo", "config.yaml")

	written, err := Init(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if written != path {
		t.Fatalf("path = %q, want %q", written, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %v, want 0600", got)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != DefaultYAML() {
		t.Fatalf("config content mismatch:\n%s", string(b))
	}
}

func TestInitDoesNotOverwriteWithoutForce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := Init(path, false)
	if !os.IsExist(err) {
		t.Fatalf("err = %v, want exist", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "existing" {
		t.Fatalf("content = %q, want existing", string(b))
	}
}

func TestInitForceResetsFileMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Init(path, true)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("mode = %v, want 0600", got)
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
