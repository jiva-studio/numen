package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadAntigravityConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

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

func TestReadAntigravityConfigFailsWithoutHome(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")
	t.Setenv("HOMEDRIVE", "")
	t.Setenv("HOMEPATH", "")
	if _, err := os.UserHomeDir(); err == nil {
		t.Skip("this platform finds a home without any environment")
	}
	cfg, err := ReadAntigravityConfig()
	if err == nil {
		t.Fatalf("expected error reading antigravity config without home directory, got %v", cfg)
	}
}
