package codexcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadCodexConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

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

func TestReadCodexConfigJsonFallbackAndDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Neither config.toml nor config.json exists.
	emptyCfg, err := ReadCodexConfig()
	if err != nil || emptyCfg.Model != "" {
		t.Fatalf("ReadCodexConfig empty = (%+v, %v), want empty model", emptyCfg, err)
	}

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o700); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(codexDir, "config.json")

	// Invalid config.json falls back to empty config.
	if err := os.WriteFile(jsonPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	badJSONCfg, err := ReadCodexConfig()
	if err != nil || badJSONCfg.Model != "" {
		t.Fatalf("ReadCodexConfig invalid json = (%+v, %v), want empty model", badJSONCfg, err)
	}

	// Valid config.json returns parsed config.
	if err := os.WriteFile(jsonPath, []byte(`{"model":"gpt-4.1"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	jsonCfg, err := ReadCodexConfig()
	if err != nil || jsonCfg.Model != "gpt-4.1" {
		t.Fatalf("ReadCodexConfig valid json = (%+v, %v), want gpt-4.1", jsonCfg, err)
	}

	// parseSimpleToml trims whitespace, ignores comments/blank/non-matching lines, and unquotes values.
	tomlCfg := parseSimpleToml("  # model = \"commented\"  \n\nno_equals_line\nother = \"ignored\"\n  model  =  'o4-mini'  \n")
	if tomlCfg.Model != "o4-mini" || tomlCfg.MCPServers == nil {
		t.Fatalf("parseSimpleToml = %+v, want Model=o4-mini and non-nil MCPServers", tomlCfg)
	}
}
