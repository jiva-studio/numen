package hosttest_test

import (
	"context"
	"errors"
	"io/fs"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

func TestAStoreKeepsWhatIsAppendedAndListsIt(t *testing.T) {
	ctx := context.Background()
	s := hosttest.New("Physics").Store()

	_ = s.Append(ctx, "log", []byte("a"))
	_ = s.Append(ctx, "log", []byte("b"))
	_ = s.Append(ctx, "first", []byte("x"))
	got, err := s.Read(ctx, "log")
	files, listErr := s.List(ctx)
	_, missing := s.Read(ctx, "none")

	if err != nil || string(got) != "ab" {
		t.Errorf("read %q (%v)", got, err)
	}
	if listErr != nil || !slices.Equal(files, []host.StoreFile{{Name: "first", Size: 1}, {Name: "log", Size: 2}}) {
		t.Errorf("listed %v (%v)", files, listErr)
	}
	if !errors.Is(missing, host.ErrNotExist) {
		t.Errorf("a missing file was read with %v", missing)
	}
}

func TestAStoreRefusesWhatTheHostRefuses(t *testing.T) {
	ctx := context.Background()
	v := hosttest.New("Physics")
	s := v.Store()
	_ = s.Append(ctx, "locked", []byte("x"))
	s.Refuse("locked", fs.ErrPermission)

	_, readErr := s.Read(ctx, "locked")
	files, _ := s.List(ctx)
	for _, name := range []string{"", "../a", "/a", ".hidden", "a/b", `a\b`} {
		if err := s.Append(ctx, name, nil); !errors.Is(err, host.ErrOutside) {
			t.Errorf("%q was appended to with %v", name, err)
		}
	}
	_, unknownErr := s.With("Biology").Read(ctx, "locked")
	v.Hide()
	_, noVaultErr := s.Read(ctx, "locked")

	if !errors.Is(readErr, fs.ErrPermission) || len(files) != 1 {
		t.Errorf("got %v and %v", readErr, files)
	}
	if !errors.Is(unknownErr, host.ErrUnknownVault) || !errors.Is(noVaultErr, host.ErrNoVault) {
		t.Errorf("got %v and %v", unknownErr, noVaultErr)
	}
}

func TestEachVaultHasAStoreOfItsOwn(t *testing.T) {
	ctx := context.Background()
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	physics.Beside(chemistry)
	_ = physics.Store().Append(ctx, "a", []byte("p"))

	_, missing := chemistry.Store().Read(ctx, "a")
	got, err := chemistry.Store().With("Physics").Read(ctx, "a")

	if !errors.Is(missing, host.ErrNotExist) || err != nil || string(got) != "p" {
		t.Errorf("got %v, %q (%v)", missing, got, err)
	}
}

func TestStoreReadsAndListsExternalFile(t *testing.T) {
	ctx := context.Background()
	s := hosttest.New("Physics").Store()
	_ = s.Append(ctx, "log", []byte("abc"))

	s.File("log", []byte("   "))
	s.File("copy", []byte("x"))
	got, _ := s.Read(ctx, "log")
	files, _ := s.List(ctx)

	if string(got) != "   " || len(files) != 2 {
		t.Errorf("read %q, listed %v", got, files)
	}
}
