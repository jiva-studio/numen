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
