package models

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestReserveQuotaRejectsWhatTheOtherReservationsLeaveNoRoomFor(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")

	first, err := Reserve(t.Context(), root, 60, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Room != 60 {
		t.Fatalf("room = %d, want 60", first.Room)
	}
	if _, err := Reserve(t.Context(), root, 60, ""); err == nil || !strings.Contains(err.Error(), EnvQuota) {
		t.Fatalf("second reservation error = %v", err)
	}
	first.Release()
	again, err := Reserve(t.Context(), root, 60, "")
	if err != nil {
		t.Fatalf("after release: %v", err)
	}
	again.Release()
}

func TestReserveQuotaOfAnUnknownSizeClaimsWhatIsRemaining(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")
	if err := os.WriteFile(filepath.Join(root, "held"), make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := Reserve(t.Context(), root, -1, "")
	if err != nil {
		t.Fatal(err)
	}
	defer claim.Release()
	if claim.Room != 60 {
		t.Fatalf("room = %d, want 60", claim.Room)
	}
}

func TestReserveQuotaReportsAFolderItCannotCreate(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Reserve(t.Context(), filepath.Join(blocker, "models"), 1, ""); err == nil {
		t.Fatal("reservation succeeded under a regular file")
	}
}

func TestLockQuotaWaitsForTheHolder(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockQuota(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithTimeout(t.Context(), 150*time.Millisecond)
	defer stop()
	if _, err := lockQuota(ctx, root); err == nil {
		t.Fatal("lock was taken twice")
	}
	unlock()
	again, err := lockQuota(t.Context(), root)
	if err != nil {
		t.Fatalf("after unlock: %v", err)
	}
	again()
}

func TestLockQuotaBreaksAStaleLock(t *testing.T) {
	root := t.TempDir()
	lock := filepath.Join(root, lockName)
	if err := os.Mkdir(lock, 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * staleLock)
	if err := os.Chtimes(lock, old, old); err != nil {
		t.Fatal(err)
	}
	unlock, err := lockQuota(t.Context(), root)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}

func TestLockQuotaReportsAFolderItCannotCreate(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := lockQuota(t.Context(), filepath.Join(blocker, "models")); err == nil {
		t.Fatal("lock was taken under a file")
	}
}

func TestReservationPreservesItselfAlive(t *testing.T) {
	root := t.TempDir()
	claim, err := reserve(t.Context(), root, 10, "", 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer claim.Release()
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(claim.file, old, old); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		info, err := os.Stat(claim.file)
		if err != nil {
			t.Fatal(err)
		}
		if time.Since(info.ModTime()) < time.Minute {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("reservation was not kept alive")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestReservationReleaseIsIdempotentAndRemovesTheFile(t *testing.T) {
	root := t.TempDir()
	claim, err := Reserve(t.Context(), root, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	claim.Release()
	claim.Release()
	if _, err := os.Stat(claim.file); !os.IsNotExist(err) {
		t.Fatalf("reservation file remains: %v", err)
	}
}

func TestReservationDoesNotCountTheBytesOfItsPartTwice(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")
	part := filepath.Join(root, "m.part-1")
	if err := os.WriteFile(part, make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := Reserve(t.Context(), root, 50, part)
	if err != nil {
		t.Fatal(err)
	}
	defer claim.Release()
	used, err := GetUsage(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(40 + len("50\n"+part) + 10); used != want {
		t.Fatalf("GetUsage() = %d, want %d", used, want)
	}
}

func TestGetHomeNamesTheEnvironmentFolder(t *testing.T) {
	t.Setenv(EnvDir, "/srv/models")
	got, err := GetHome()
	if err != nil || got != "/srv/models" {
		t.Fatalf("GetHome() = %q, %v", got, err)
	}
}

func TestGetHomeIsBelowTheApplicationDataDirectory(t *testing.T) {
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
	tests := []struct {
		value string
		want  bool
	}{
		{"", false}, {"0", false}, {"no", false}, {"1", true}, {"true", true}, {" TRUE ", true}, {"yes", true}, {"on", true},
	}
	for _, tc := range tests {
		t.Setenv(EnvOffline, tc.value)
		if got := IsOffline(); got != tc.want {
			t.Errorf("%s=%q: IsOffline() = %v, want %v", EnvOffline, tc.value, got, tc.want)
		}
	}
}

func TestGetQuota(t *testing.T) {
	tests := []struct {
		value string
		want  int64
	}{
		{"", DefaultQuota}, {"x", DefaultQuota}, {"0", DefaultQuota}, {"-5", DefaultQuota}, {"1024", 1024}, {" 2048 ", 2048},
	}
	for _, tc := range tests {
		t.Setenv(EnvQuota, tc.value)
		if got := GetQuota(); got != tc.want {
			t.Errorf("%s=%q: GetQuota() = %d, want %d", EnvQuota, tc.value, got, tc.want)
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

func TestGetUsageCountsWhatALiveReservationHasNotSaved(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "m.part-1")
	if err := os.WriteFile(part, make([]byte, 30), 0o644); err != nil {
		t.Fatal(err)
	}
	reservation := "100\n" + part
	if err := os.WriteFile(filepath.Join(dir, "reservation-1"+reserveSuffix), []byte(reservation), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := GetUsage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if want := int64(30 + len(reservation) + 70); got != want {
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

func TestResolvePathFollowsALinkInTheExistingPart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("links need privileges on windows")
	}
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(base, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if got := resolvePath(filepath.Join(link, "new", "deeper")); got != filepath.Join(target, "new", "deeper") {
		t.Fatalf("resolvePath() = %q", got)
	}
	if got := resolvePath(link); got != target {
		t.Fatalf("resolvePath(link) = %q", got)
	}
}
