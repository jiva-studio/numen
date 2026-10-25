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
	listed, listErr := s.List(ctx)
	_, missing := s.Read(ctx, "none")

	if err != nil || string(got) != "ab" {
		t.Errorf("read %q (%v)", got, err)
	}
	if listErr != nil || !slices.Equal(listed, []host.StoreFile{{Name: "first", Size: 1}, {Name: "log", Size: 2}}) {
		t.Errorf("listed %v (%v)", listed, listErr)
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

	_, refused := s.Read(ctx, "locked")
	listed, _ := s.List(ctx)
	for _, name := range []string{"", "../a", "/a", ".hidden", "a/b", `a\b`} {
		if err := s.Append(ctx, name, nil); !errors.Is(err, host.ErrOutside) {
			t.Errorf("%q was appended to with %v", name, err)
		}
	}
	_, unknown := s.With("Biology").Read(ctx, "locked")
	v.Hide()
	_, hidden := s.Read(ctx, "locked")

	if !errors.Is(refused, fs.ErrPermission) || len(listed) != 1 {
		t.Errorf("got %v and %v", refused, listed)
	}
	if !errors.Is(unknown, host.ErrUnknownVault) || !errors.Is(hidden, host.ErrNoVault) {
		t.Errorf("got %v and %v", unknown, hidden)
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

func TestAFilePutByAnotherProgramIsReadAndListed(t *testing.T) {
	ctx := context.Background()
	s := hosttest.New("Physics").Store()
	_ = s.Append(ctx, "log", []byte("abc"))

	s.File("log", []byte("   "))
	s.File("copy", []byte("x"))
	got, _ := s.Read(ctx, "log")
	listed, _ := s.List(ctx)

	if string(got) != "   " || len(listed) != 2 {
		t.Errorf("read %q and listed %v", got, listed)
	}
}
