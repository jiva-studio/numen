package models

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestGetHomeNamesTheEnvironmentFolder(t *testing.T) {
	t.Setenv(EnvDir, "/srv/models")
	got, err := GetHome()
	if err != nil || got != "/srv/models" {
		t.Fatalf("GetHome() = %q, %v", got, err)
	}
}

func TestGetHomeIsUnderTheApplicationDataDirectory(t *testing.T) {
	cache := t.TempDir()
	t.Setenv(EnvDir, "")
	t.Setenv("XDG_CONFIG_HOME", cache)
	t.Setenv("AppData", cache)
	t.Setenv("HOME", cache)
	got, err := GetHome()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(filepath.ToSlash(got), "numen/models") || !strings.HasPrefix(got, cache) {
		t.Fatalf("GetHome() = %q", got)
	}
}

func TestIsOffline(t *testing.T) {
	for value, want := range map[string]bool{
		"": false, "0": false, "no": false, "1": true, "true": true, " TRUE ": true, "yes": true, "on": true,
	} {
		t.Setenv(EnvOffline, value)
		if got := IsOffline(); got != want {
			t.Errorf("%s=%q: IsOffline() = %v, want %v", EnvOffline, value, got, want)
		}
	}
}

func TestGetQuota(t *testing.T) {
	for value, want := range map[string]int64{
		"": DefaultQuota, "x": DefaultQuota, "0": DefaultQuota, "-5": DefaultQuota, "1024": 1024, " 2048 ": 2048,
	} {
		t.Setenv(EnvQuota, value)
		if got := GetQuota(); got != want {
			t.Errorf("%s=%q: GetQuota() = %d, want %d", EnvQuota, value, got, want)
		}
	}
}

func TestGetUsageSumsNestedFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, size := range map[string]int{"one": 3, "a/two": 5} {
		if err := os.WriteFile(filepath.Join(dir, name), make([]byte, size), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := GetUsage(dir); err != nil || got != 8 {
		t.Fatalf("GetUsage() = %d, %v, want 8", got, err)
	}
	if got, err := GetUsage(filepath.Join(dir, "missing")); err != nil || got != 0 {
		t.Fatalf("GetUsage(missing) = %d, %v, want 0", got, err)
	}
}

func TestGetUsageCountsWhatALiveReservationHasNotWritten(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "m.part-1")
	if err := os.WriteFile(part, make([]byte, 30), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part+reserveSuffix, []byte("100"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := GetUsage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(30 + len("100") + 70); got != want {
		t.Fatalf("GetUsage() = %d, want %d", got, want)
	}
}

func TestGetUsageIgnoresWhatAStaleReservationPromised(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "m.part-1"+reserveSuffix)
	if err := os.WriteFile(file, []byte("100"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * staleReservation)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}
	if got, err := GetUsage(dir); err != nil || got != 3 {
		t.Fatalf("GetUsage() = %d, %v, want 3", got, err)
	}
}

func TestGetUsageIgnoresAReservationThatIsNotANumber(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.part-1"+reserveSuffix), []byte("many"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := GetUsage(dir); err != nil || got != 4 {
		t.Fatalf("GetUsage() = %d, %v, want 4", got, err)
	}
}

func TestGetUsageReportsAFolderItCannotRead(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("folder permissions do not stop this user")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if _, err := GetUsage(dir); err == nil {
		t.Fatal("an unreadable folder was counted as empty")
	}
	if err := CheckQuota(dir, 1); err == nil {
		t.Fatal("CheckQuota passed over an unreadable folder")
	}
}

func TestCheckQuota(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), make([]byte, 6), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvQuota, "10")
	if err := CheckQuota(dir, 4); err != nil {
		t.Fatalf("10 of 10 bytes: %v", err)
	}
	if err := CheckQuota(dir, 5); err == nil {
		t.Fatal("11 of 10 bytes passed")
	}
}
