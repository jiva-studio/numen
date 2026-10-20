package hosttest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/modules/libs/plugin-sdk-go/host/hosttest"
)

func TestAWriteFromAnOldReadIsStale(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	v.File("a.md", "one")
	_, old, _ := v.Read(ctx, "a.md")
	if _, err := v.Write(ctx, "a.md", []byte("two"), old); err != nil {
		t.Fatal(err)
	}

	_, err := v.Write(ctx, "a.md", []byte("three"), old)

	if !errors.Is(err, host.ErrStale) {
		t.Errorf("got %v", err)
	}
}

func TestACreateOverAFileIsOccupied(t *testing.T) {
	v := hosttest.New("Physics")
	v.File("a.md", "one")

	if err := v.Create(context.Background(), "a.md", nil); !errors.Is(err, host.ErrOccupied) {
		t.Errorf("got %v", err)
	}
	if err := v.Create(context.Background(), "../a.md", nil); !errors.Is(err, host.ErrOutside) {
		t.Errorf("got %v", err)
	}
}

func TestAHiddenVaultAnswersNoVault(t *testing.T) {
	v := hosttest.New("Physics")
	v.Hide()

	if _, err := v.GetVault(context.Background()); !errors.Is(err, host.ErrNoVault) {
		t.Errorf("got %v", err)
	}
}

func TestADataFileGrowsByAppending(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	_ = v.AppendData(ctx, "log", []byte("a"))
	_ = v.AppendData(ctx, "log", []byte("b"))

	got, err := v.ReadData(ctx, "log")

	if err != nil || string(got) != "ab" {
		t.Errorf("got %q (%v)", got, err)
	}
}

func TestAnotherVaultIsWorkedByItsIdentity(t *testing.T) {
	ctx := context.Background()
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	chemistry.File("a.md", "x")
	physics.Beside(chemistry)

	known, _ := physics.GetVaults(ctx)
	_, _, readErr := physics.With("Chemistry").Read(ctx, "a.md")
	_, _, unknown := physics.With("Biology").Read(ctx, "a.md")

	if len(known) != 2 || readErr != nil || !errors.Is(unknown, host.ErrUnknownVault) {
		t.Errorf("the vaults were %v, the read gave %v, an unknown vault gave %v", known, readErr, unknown)
	}
}

func TestAMoveIsToldToTheWatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	v := hosttest.New("Physics")
	moves, err := v.WatchMoves(ctx)
	if err != nil {
		t.Fatal(err)
	}

	v.Move()

	if _, open := <-moves; !open {
		t.Error("the watch closed")
	}
}
