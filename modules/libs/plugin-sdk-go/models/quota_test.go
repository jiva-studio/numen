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

	first, err := Reserve(context.Background(), root, 60, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Room != 60 {
		t.Fatalf("room = %d, want 60", first.Room)
	}
	if _, err := Reserve(context.Background(), root, 60, ""); err == nil || !strings.Contains(err.Error(), EnvQuota) {
		t.Fatalf("second reservation error = %v", err)
	}
	first.Release()
	if _, err := Reserve(context.Background(), root, 60, ""); err != nil {
		t.Fatalf("after release: %v", err)
	}
}

func TestReserveQuotaOfAnUnknownSizeClaimsWhatIsLeft(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")
	if err := os.WriteFile(filepath.Join(root, "held"), make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := Reserve(context.Background(), root, -1, "")
	if err != nil {
		t.Fatal(err)
	}
	if claim.Room != 60 {
		t.Fatalf("room = %d, want 60", claim.Room)
	}
}

func TestReserveQuotaReportsAFolderItCannotCreate(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Reserve(context.Background(), filepath.Join(blocker, "models"), 1, ""); err == nil {
		t.Fatal("a reservation was made under a file")
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

func TestReservationKeepsItselfAlive(t *testing.T) {
	previous := touchInterval
	touchInterval = 10 * time.Millisecond
	t.Cleanup(func() { touchInterval = previous })
	root := t.TempDir()
	claim, err := Reserve(context.Background(), root, 10, "")
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
			t.Fatal("the reservation was not touched")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestReservationReleaseIsIdempotentAndRemovesTheFile(t *testing.T) {
	root := t.TempDir()
	claim, err := Reserve(context.Background(), root, 10, "")
	if err != nil {
		t.Fatal(err)
	}
	claim.Release()
	claim.Release()
	if _, err := os.Stat(claim.file); !os.IsNotExist(err) {
		t.Fatalf("the reservation file remains: %v", err)
	}
}

func TestReservationDoesNotCountTheBytesOfItsPartTwice(t *testing.T) {
	root := t.TempDir()
	t.Setenv(EnvQuota, "100")
	part := filepath.Join(root, "m.part-1")
	if err := os.WriteFile(part, make([]byte, 40), 0o644); err != nil {
		t.Fatal(err)
	}
	claim, err := Reserve(context.Background(), root, 50, part)
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
