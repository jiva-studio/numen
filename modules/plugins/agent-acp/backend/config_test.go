package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfigurations(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Codex toml config
	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	codexData := "model = \"o3-mini\"\n"
	if err := os.WriteFile(filepath.Join(codexDir, "config.toml"), []byte(codexData), 0o600); err != nil {
		t.Fatal(err)
	}
	codexCfg, err := ReadCodexConfig()
	if err != nil {
		t.Fatal(err)
	}
	if codexCfg.Model != "o3-mini" {
		t.Errorf("expected model o3-mini, got %s", codexCfg.Model)
	}

	// Antigravity config
	geminiDir := filepath.Join(home, ".gemini", "antigravity-cli")
	if err := os.MkdirAll(geminiDir, 0o700); err != nil {
		t.Fatal(err)
	}
	geminiData := `{"model": "gemini-2.5-pro"}`
	if err := os.WriteFile(filepath.Join(geminiDir, "config.json"), []byte(geminiData), 0o600); err != nil {
		t.Fatal(err)
	}
	agyCfg, err := ReadAntigravityConfig()
	if err != nil {
		t.Fatal(err)
	}
	if agyCfg.Model != "gemini-2.5-pro" {
		t.Errorf("expected model gemini-2.5-pro, got %s", agyCfg.Model)
	}
}

func TestParseSimpleToml(t *testing.T) {
	cfg := parseSimpleToml("model = \"claude-3=5-sonnet\"\n")
	if cfg.Model != "claude-3=5-sonnet" {
		t.Errorf("expected model 'claude-3=5-sonnet', got %q", cfg.Model)
	}
}

func isolateWithoutHome(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("this platform finds a home without any environment")
	}
}

func TestReadCodexConfigFailsWithoutHome(t *testing.T) {
	isolateWithoutHome(t)
	cfg, err := ReadCodexConfig()
	if err == nil {
		t.Fatalf("expected error reading codex config without home directory, got %v", cfg)
	}
}

func TestReadAntigravityConfigFailsWithoutHome(t *testing.T) {
	isolateWithoutHome(t)
	cfg, err := ReadAntigravityConfig()
	if err == nil {
		t.Fatalf("expected error reading antigravity config without home directory, got %v", cfg)
	}
}

