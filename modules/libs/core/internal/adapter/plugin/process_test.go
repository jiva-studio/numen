package plugin

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
)

func TestEnvironmentSanitization(t *testing.T) {
	allowed := []string{
		"PATH", "path",
		"TMPDIR", "tmpdir",
		"TMP", "tmp",
		"TEMP", "temp",
		"HOME", "home",
		"USER", "user",
		"SYSTEMROOT", "SystemRoot",
		"SYSTEMDRIVE", "SystemDrive",
		"APPDATA", "AppData",
		"LOCALAPPDATA", "LocalAppData",
		"LANG", "lang",
		"LC_ALL", "lc_ctype", "LC_MESSAGES",
		"SSL_CERT_FILE", "SSL_CERT_DIR",
	}
	for _, key := range allowed {
		if !isWhitelistedEnv(key) {
			t.Errorf("expected %s to be allowed in environment whitelist", key)
		}
	}

	disallowed := []string{
		"",
		"OPENAI_API_KEY",
		"NUMEN_EMBEDDING_KEY",
		"NUMEN_PROOFREADING_KEY",
		"AWS_SECRET_ACCESS_KEY",
		"GITHUB_TOKEN",
		"ACCESS_TOKEN",
		"AUTH_TOKEN",
		"API_KEY",
		"PRIVATE_KEY",
		"SECRET_KEY",
		"DB_PASSWORD",
		"USER_PASSWORD",
		"CREDENTIALS_FILE",
		"DATABASE_URL",
		"FOO",
		"MY_VAR",
		"XDG_DATA_HOME",
		"NIX_PATH",
	}
	for _, key := range disallowed {
		if isWhitelistedEnv(key) {
			t.Errorf("expected %s to be blocked from environment whitelist", key)
		}
	}
}

func TestSanitizeEnvironment(t *testing.T) {
	inputEnv := []string{
		"PATH=/usr/bin:/bin",
		"TMPDIR=/custom/tmp",
		"OPENAI_API_KEY=sk-1234567890",
		"GITHUB_TOKEN=ghp_secret",
		"HOME=/home/user",
		"AWS_SECRET_ACCESS_KEY=supersecret",
		"DATABASE_URL=postgres://user:pass@localhost/db",
		"LANG=en_US.UTF-8",
		"LC_ALL=C",
	}

	pluginVars := []string{
		"NUMEN_VAULT_PATH=/vault",
		"NUMEN_PLUGIN_PORT=8080",
		"NUMEN_PLUGIN_TOKEN=plugin-token-xyz",
		"NUMEN_PLUGIN_ID=test.plugin",
	}

	cleaned := sanitizeEnvironment(inputEnv, pluginVars)

	seen := make(map[string]string)
	for _, entry := range cleaned {
		k, v, ok := stringsCut(entry, "=")
		if ok {
			seen[k] = v
		}
	}

	if seen["PATH"] != "/usr/bin:/bin" {
		t.Errorf("expected PATH to be preserved, got %q", seen["PATH"])
	}
	if seen["TMPDIR"] != "/custom/tmp" {
		t.Errorf("expected TMPDIR to be preserved, got %q", seen["TMPDIR"])
	}
	if seen["HOME"] != "/home/user" {
		t.Errorf("expected HOME to be preserved, got %q", seen["HOME"])
	}
	if seen["LANG"] != "en_US.UTF-8" {
		t.Errorf("expected LANG to be preserved, got %q", seen["LANG"])
	}
	if seen["LC_ALL"] != "C" {
		t.Errorf("expected LC_ALL to be preserved, got %q", seen["LC_ALL"])
	}

	for _, secret := range []string{"OPENAI_API_KEY", "GITHUB_TOKEN", "AWS_SECRET_ACCESS_KEY", "DATABASE_URL"} {
		if _, exists := seen[secret]; exists {
			t.Errorf("expected secret %s to be scrubbed from environment", secret)
		}
	}

	if seen["NUMEN_VAULT_PATH"] != "/vault" {
		t.Errorf("expected NUMEN_VAULT_PATH to be present, got %q", seen["NUMEN_VAULT_PATH"])
	}
	if seen["NUMEN_PLUGIN_PORT"] != "8080" {
		t.Errorf("expected NUMEN_PLUGIN_PORT to be present, got %q", seen["NUMEN_PLUGIN_PORT"])
	}
	if seen["NUMEN_PLUGIN_TOKEN"] != "plugin-token-xyz" {
		t.Errorf("expected NUMEN_PLUGIN_TOKEN to be present, got %q", seen["NUMEN_PLUGIN_TOKEN"])
	}
}

func stringsCut(s, sep string) (before, after string, found bool) {
	for i := 0; i+len(sep) <= len(s); i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i], s[i+len(sep):], true
		}
	}
	return s, "", false
}

func TestValidateWorkingDirectory(t *testing.T) {
	pluginDir := filepath.Join(string(filepath.Separator), "opt", "numen", "plugins", "ocr")
	vaultRoot := filepath.Join(string(filepath.Separator), "home", "user", "documents", "vault")

	t.Run("confinement within plugin directory", func(t *testing.T) {
		got, err := ValidateWorkingDirectory(pluginDir, pluginDir, vaultRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != pluginDir {
			t.Fatalf("got %s, want %s", got, pluginDir)
		}

		subDir := filepath.Join(pluginDir, "sub", "dir")
		gotSub, err := ValidateWorkingDirectory(subDir, pluginDir, vaultRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotSub != subDir {
			t.Fatalf("got %s, want %s", gotSub, subDir)
		}
	})

	t.Run("confinement within vault directory", func(t *testing.T) {
		got, err := ValidateWorkingDirectory(vaultRoot, pluginDir, vaultRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != vaultRoot {
			t.Fatalf("got %s, want %s", got, vaultRoot)
		}

		vaultSub := filepath.Join(vaultRoot, "notes")
		gotSub, err := ValidateWorkingDirectory(vaultSub, pluginDir, vaultRoot)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gotSub != vaultSub {
			t.Fatalf("got %s, want %s", gotSub, vaultSub)
		}
	})

	t.Run("empty working directory rejected", func(t *testing.T) {
		_, err := ValidateWorkingDirectory("", pluginDir, vaultRoot)
		if !errors.Is(err, ErrEmptyWorkingDirectory) {
			t.Fatalf("expected ErrEmptyWorkingDirectory, got %v", err)
		}
	})

	t.Run("escape outside sandbox and vault rejected", func(t *testing.T) {
		escapePaths := []string{
			filepath.Join(string(filepath.Separator), "etc"),
			filepath.Join(string(filepath.Separator), "usr", "bin"),
			filepath.Join(pluginDir, "..", "..", "escape"),
			filepath.Join(vaultRoot, "..", "escape"),
		}
		for _, p := range escapePaths {
			_, err := ValidateWorkingDirectory(p, pluginDir, vaultRoot)
			if !errors.Is(err, ErrWorkingDirectoryEscape) {
				t.Errorf("path %s: expected ErrWorkingDirectoryEscape, got %v", p, err)
			}
		}
	})

	t.Run("prohibits temporary directory as working directory", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := ValidateWorkingDirectory(tempDir, tempDir, vaultRoot)
		if !errors.Is(err, errProhibitedTempDir) {
			t.Fatalf("expected errProhibitedTempDir, got %v", err)
		}
	})

	t.Run("symlink escape rejected", func(t *testing.T) {
		tempBase := filepath.Join(string(filepath.Separator), "var", "target")
		if runtime.GOOS == "windows" {
			t.Skip("skipping symlink test on windows")
		}
		realDir := t.TempDir()
		outsideDir := t.TempDir()
		symlinkPath := filepath.Join(realDir, "link_outside")
		if err := os.Symlink(outsideDir, symlinkPath); err != nil {
			t.Skipf("cannot create symlink: %v", err)
		}
		_ = tempBase

		_, err := ValidateWorkingDirectory(symlinkPath, realDir, "")
		if err == nil {
			t.Fatalf("expected error for symlink escape, got nil")
		}
	})
}

func TestProcessLimits(t *testing.T) {
	defaults := DefaultProcessLimits()
	if defaults.CPULimitSeconds == 0 {
		t.Error("expected non-zero default CPU limit")
	}
	if defaults.MemoryLimitBytes == 0 {
		t.Error("expected non-zero default memory limit")
	}

	cmd := exec.Command("sleep", "5")
	configureSysProcAttr(cmd)

	if runtime.GOOS == "linux" {
		if cmd.SysProcAttr == nil {
			t.Fatal("expected non-nil SysProcAttr on linux")
		}
		if !cmd.SysProcAttr.Setpgid {
			t.Error("expected Setpgid to be true on linux")
		}
		if cmd.SysProcAttr.Pdeathsig != syscall.SIGTERM {
			t.Errorf("expected Pdeathsig SIGTERM, got %v", cmd.SysProcAttr.Pdeathsig)
		}
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()

	t.Run("apply valid limits", func(t *testing.T) {
		limits := ProcessLimits{
			CPULimitSeconds:  10,
			MemoryLimitBytes: 1024 * 1024 * 1024,
		}
		if err := applyProcessLimits(cmd.Process.Pid, limits); err != nil {
			t.Fatalf("applyProcessLimits failed: %v", err)
		}
	})

	t.Run("apply zero limits no-op", func(t *testing.T) {
		if err := applyProcessLimits(cmd.Process.Pid, ProcessLimits{}); err != nil {
			t.Fatalf("applyProcessLimits zero failed: %v", err)
		}
	})

	if runtime.GOOS == "linux" {
		t.Run("apply limits to invalid pid", func(t *testing.T) {
			err := applyProcessLimits(-1, ProcessLimits{CPULimitSeconds: 5})
			if err == nil {
				t.Fatal("expected error applying limits to invalid pid, got nil")
			}
		})
	}
}

func TestHostProcessLimitsConfig(t *testing.T) {
	h := NewHost("test")
	if h.processLimits.CPULimitSeconds == 0 || h.processLimits.MemoryLimitBytes == 0 {
		t.Fatal("expected default limits to be initialized in NewHost")
	}

	customLimits := ProcessLimits{
		CPULimitSeconds:  60,
		MemoryLimitBytes: 512 * 1024 * 1024,
	}
	h.SetProcessLimits(customLimits)

	h.mu.RLock()
	limits := h.processLimits
	h.mu.RUnlock()

	if limits != customLimits {
		t.Fatalf("got %v, want %v", limits, customLimits)
	}
}
