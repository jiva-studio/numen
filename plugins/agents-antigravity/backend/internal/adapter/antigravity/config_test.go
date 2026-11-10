package antigravity

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

func TestReadAntigravityConfigFallbackCandidatesAndEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// No config files exist -> returns empty AntigravityConfig.
	emptyCfg, err := ReadAntigravityConfig()
	if err != nil || emptyCfg.Model != "" {
		t.Fatalf("ReadAntigravityConfig with no files = (%+v, %v), want empty model", emptyCfg, err)
	}

	// Invalid JSON in first candidate falls through to third candidate (~/.antigravity/config.json).
	firstDir := filepath.Join(home, ".gemini", "antigravity-cli")
	thirdDir := filepath.Join(home, ".antigravity")
	if err := os.MkdirAll(firstDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(thirdDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firstDir, "config.json"), []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(thirdDir, "config.json"), []byte(`{"model":"gemini-fallback"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	fallbackCfg, err := ReadAntigravityConfig()
	if err != nil || fallbackCfg.Model != "gemini-fallback" {
		t.Fatalf("ReadAntigravityConfig fallback = (%+v, %v), want gemini-fallback", fallbackCfg, err)
	}
}
