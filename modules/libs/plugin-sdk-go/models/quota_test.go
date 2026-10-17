package models

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReserveQuotaRefusesWhatTheOtherReservationsLeaveNoRoomFor(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")

	first, err := reserveQuota(context.Background(), root, filepath.Join(root, "a.part-1"), 60)
	if err != nil {
		t.Fatal(err)
	}
	if first.room != 60 {
		t.Fatalf("room = %d, want 60", first.room)
	}
	if _, err := reserveQuota(context.Background(), root, filepath.Join(root, "b.part-1"), 60); err == nil || !strings.Contains(err.Error(), EnvQuota) {
		t.Fatalf("second reservation error = %v", err)
	}
	first.release()
	if _, err := reserveQuota(context.Background(), root, filepath.Join(root, "b.part-1"), 60); err != nil {
		t.Fatalf("after release: %v", err)
	}
}

func TestReserveQuotaOfAnUnknownSizeClaimsWhatIsLeft(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")
	if err := os.WriteFile(filepath.Join(root, "held"), make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := reserveQuota(context.Background(), root, filepath.Join(root, "a.part-1"), -1)
	if err != nil {
		t.Fatal(err)
	}
	if claim.room != 60 {
		t.Fatalf("room = %d, want 60", claim.room)
	}
}

func TestReserveQuotaReportsAFolderItCannotCreate(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := reserveQuota(context.Background(), filepath.Join(blocker, "models"), "x.part-1", 1); err == nil {
		t.Fatal("a reservation was made under a file")
	}
}

func TestReservationTouchKeepsItAlive(t *testing.T) {
	root := t.TempDir()
	claim, err := reserveQuota(context.Background(), root, filepath.Join(root, "a.part-1"), 10)
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(claim.file, old, old); err != nil {
		t.Fatal(err)
	}
	claim.touch()
	info, err := os.Stat(claim.file)
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(info.ModTime()) > time.Minute {
		t.Fatalf("modified %v ago", time.Since(info.ModTime()))
	}
}

func TestLockQuotaWaitsForTheHolder(t *testing.T) {
	root := t.TempDir()
	unlock, err := lockQuota(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, stop := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer stop()
	if _, err := lockQuota(ctx, root); err == nil {
		t.Fatal("the lock was taken twice")
	}
	unlock()
	again, err := lockQuota(context.Background(), root)
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
	unlock, err := lockQuota(context.Background(), root)
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
	if _, err := lockQuota(context.Background(), filepath.Join(blocker, "models")); err == nil {
		t.Fatal("a lock was taken under a file")
	}
}
