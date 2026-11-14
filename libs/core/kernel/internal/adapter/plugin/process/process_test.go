package process

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
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
		"HTTP_PROXY", "https_proxy", "NO_PROXY", "no_proxy",
		"NUMEN_TEST_MOCK_ML", "NUMEN_TEST_MOCK_EMBED_LOG", "NUMEN_TEST_MOCK_EMBED_HOLD_FILE", "NUMEN_TEST_LANTERN",
	}
	for _, key := range allowed {
		if !IsWhitelistedEnv(key) {
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
		if IsWhitelistedEnv(key) {
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

	cleaned := SanitizeEnvironment(inputEnv, pluginVars, nil)

	observed := make(map[string]string)
	for _, entry := range cleaned {
		k, v, ok := stringsCut(entry, "=")
		if ok {
			observed[k] = v
		}
	}

	if observed["PATH"] != "/usr/bin:/bin" {
		t.Errorf("expected PATH to be preserved, got %q", observed["PATH"])
	}
	if observed["TMPDIR"] != "/custom/tmp" {
		t.Errorf("expected TMPDIR to be preserved, got %q", observed["TMPDIR"])
	}
	if observed["HOME"] != "/home/user" {
		t.Errorf("expected HOME to be preserved, got %q", observed["HOME"])
	}
	if observed["LANG"] != "en_US.UTF-8" {
		t.Errorf("expected LANG to be preserved, got %q", observed["LANG"])
	}
	if observed["LC_ALL"] != "C" {
		t.Errorf("expected LC_ALL to be preserved, got %q", observed["LC_ALL"])
	}

	for _, secret := range []string{"OPENAI_API_KEY", "GITHUB_TOKEN", "AWS_SECRET_ACCESS_KEY", "DATABASE_URL"} {
		if _, exists := observed[secret]; exists {
			t.Errorf("expected secret %s to be scrubbed from environment", secret)
		}
	}

	if observed["NUMEN_VAULT_PATH"] != "/vault" {
		t.Errorf("expected NUMEN_VAULT_PATH to be present, got %q", observed["NUMEN_VAULT_PATH"])
	}
	if observed["NUMEN_PLUGIN_PORT"] != "8080" {
		t.Errorf("expected NUMEN_PLUGIN_PORT to be present, got %q", observed["NUMEN_PLUGIN_PORT"])
	}
	if observed["NUMEN_PLUGIN_TOKEN"] != "plugin-token-xyz" {
		t.Errorf("expected NUMEN_PLUGIN_TOKEN to be present, got %q", observed["NUMEN_PLUGIN_TOKEN"])
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
	pluginDir := createFolderOutsideTemp(t, "plugin")
	vaultRoot := createFolderOutsideTemp(t, "vault")
	for _, one := range []string{filepath.Join(pluginDir, "sub", "dir"), filepath.Join(vaultRoot, "notes")} {
		if err := os.MkdirAll(one, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a folder that does not exist is refused", func(t *testing.T) {
		missing := filepath.Join(pluginDir, "missing")
		if _, err := ValidateWorkingDirectory(missing, pluginDir, vaultRoot); !errors.Is(err, ErrWorkingDirectoryEscape) {
			t.Fatalf("expected ErrWorkingDirectoryEscape, got %v", err)
		}
	})

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
		if !errors.Is(err, security.ErrProhibitedTempDir) {
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
	defaults := DefaultLimits()
	if defaults.CPULimitSeconds != 0 {
		t.Errorf("the default CPU limit is %d seconds, want none", defaults.CPULimitSeconds)
	}

	cmd := exec.CommandContext(t.Context(), "sleep", "5")
	ConfigureSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start sleep: %v", err)
	}
	defer func() {
		wiretest.Kill(t, cmd)
		wiretest.ReapProcess(t, cmd.Process)
	}()

	t.Run("apply valid limits", func(t *testing.T) {
		limits := Limits{CPULimitSeconds: 10}
		if err := ApplyLimits(cmd.Process.Pid, limits); err != nil {
			t.Fatalf("applyProcessLimits failed: %v", err)
		}
	})

	t.Run("apply zero limits no-op", func(t *testing.T) {
		if err := ApplyLimits(cmd.Process.Pid, Limits{}); err != nil {
			t.Fatalf("applyProcessLimits zero failed: %v", err)
		}
	})

	if runtime.GOOS == "linux" {
		t.Run("apply limits to invalid pid", func(t *testing.T) {
			err := ApplyLimits(-1, Limits{CPULimitSeconds: 5})
			if err == nil {
				t.Fatal("expected error applying limits to invalid pid, got nil")
			}
		})
	}
}

func getHostEnvironment() []string {
	return []string{
		"PATH=/usr/bin",
		"LOADER_PATH=/opt/accel/lib",
		"ACCEL_VISIBLE_DEVICES=0",
		"ACCEL_PATH_V12_4=/opt/accel-12.4",
		"ACCEL_PATH_V12_TOKEN=1",
		"ACCEL_API_KEY=1",
		"SAMPLE_RUNTIME_LIB=/opt/sample/libsample.so",
		"OTHER_SERVICE_KEY=1",
	}
}

func readEnvironment(env []string) map[string]string {
	observed := make(map[string]string, len(env))
	for _, entry := range env {
		if k, v, ok := stringsCut(entry, "="); ok {
			observed[k] = v
		}
	}
	return observed
}

func TestSanitizeEnvironment_PassesTheVariablesADependencyDeclares(t *testing.T) {
	declared := []string{"LOADER_PATH", "ACCEL_*", "accel_visible_devices"}
	observed := readEnvironment(SanitizeEnvironment(getHostEnvironment(), nil, declared))

	want := map[string]string{"LOADER_PATH": "/opt/accel/lib", "ACCEL_VISIBLE_DEVICES": "0", "ACCEL_PATH_V12_4": "/opt/accel-12.4", "PATH": "/usr/bin"}
	for name, value := range want {
		if observed[name] != value {
			t.Errorf("%s = %q, want %q", name, observed[name], value)
		}
	}
	for _, name := range []string{"ACCEL_PATH_V12_TOKEN", "ACCEL_API_KEY", "OTHER_SERVICE_KEY", "SAMPLE_RUNTIME_LIB"} {
		if _, ok := observed[name]; ok {
			t.Errorf("%s reached the plugin", name)
		}
	}
}

func TestSanitizeEnvironment_DropsTheVariablesNoDependencyDeclares(t *testing.T) {
	for _, declared := range [][]string{nil, {}, {"OTHER_*"}} {
		observed := readEnvironment(SanitizeEnvironment(getHostEnvironment(), nil, declared))
		for _, name := range []string{"LOADER_PATH", "ACCEL_VISIBLE_DEVICES", "ACCEL_PATH_V12_4", "SAMPLE_RUNTIME_LIB", "OTHER_SERVICE_KEY"} {
			if _, ok := observed[name]; ok {
				t.Errorf("%s reached a plugin that declares %v", name, declared)
			}
		}
		if observed["PATH"] != "/usr/bin" {
			t.Errorf("PATH = %q, want /usr/bin", observed["PATH"])
		}
	}
}

func TestSanitizeEnvironment_ThePluginVariablesFollowTheHostOnes(t *testing.T) {
	env := SanitizeEnvironment(getHostEnvironment(), []string{"SAMPLE_RUNTIME_LIB=/models/libsample.so"}, []string{"SAMPLE_RUNTIME_LIB"})
	if got := env[len(env)-1]; got != "SAMPLE_RUNTIME_LIB=/models/libsample.so" {
		t.Errorf("the last variable is %q, want the one the host provisioned", got)
	}
}

func createFolderOutsideTemp(t *testing.T, name string) string {
	t.Helper()
	dir := name + "-" + strconv.Itoa(os.Getpid())
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(dir); err != nil {
			t.Error(err)
		}
	})
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	return abs
}
