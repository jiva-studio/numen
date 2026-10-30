package hosttest_test

import (
	"errors"
	"io/fs"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

func TestAStoreKeepsWhatIsAppendedAndListsIt(t *testing.T) {
	ctx := t.Context()
	s := hosttest.New("Physics").Store()

	if err := s.Append(ctx, "log", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := s.Append(ctx, "log", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := s.Append(ctx, "first", []byte("x")); err != nil {
		t.Fatal(err)
	}
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
	ctx := t.Context()
	v := hosttest.New("Physics")
	s := v.Store()
	if err := s.Append(ctx, "locked", []byte("x")); err != nil {
		t.Fatal(err)
	}
	s.Refuse("locked", fs.ErrPermission)

	_, readErr := s.Read(ctx, "locked")
	files, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
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
	ctx := t.Context()
	physics, chemistry := hosttest.New("Physics"), hosttest.New("Chemistry")
	physics.Beside(chemistry)
	if err := physics.Store().Append(ctx, "a", []byte("p")); err != nil {
		t.Fatal(err)
	}

	_, missing := chemistry.Store().Read(ctx, "a")
	got, err := chemistry.Store().With("Physics").Read(ctx, "a")

	if !errors.Is(missing, host.ErrNotExist) || err != nil || string(got) != "p" {
		t.Errorf("got %v, %q (%v)", missing, got, err)
	}
}

func TestStoreReadsAndListsExternalFile(t *testing.T) {
	ctx := t.Context()
	s := hosttest.New("Physics").Store()
	if err := s.Append(ctx, "log", []byte("abc")); err != nil {
		t.Fatal(err)
	}

	s.File("log", []byte("   "))
	s.File("copy", []byte("x"))
	got, err := s.Read(ctx, "log")
	if err != nil {
		t.Fatal(err)
	}
	files, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != "   " || len(files) != 2 {
		t.Errorf("read %q, listed %v", got, files)
	}
}

func TestStoreWriteAndDelete(t *testing.T) {
	ctx := t.Context()
	s := hosttest.New("Physics").Store()

	if err := s.Append(ctx, "a", []byte("first and longer")); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(ctx, "a", []byte("second")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(ctx, "a")
	if err != nil || string(got) != "second" {
		t.Errorf("read %q (%v), want the write alone", got, err)
	}
	if err := s.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "a"); err != nil {
		t.Errorf("a file that is not there was not removed: %v", err)
	}
	if _, err := s.Read(ctx, "a"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a deleted file was read with %v", err)
	}
	if err := s.Write(ctx, "../b", nil); !errors.Is(err, host.ErrOutside) {
		t.Errorf("a write outside the folder answered %v", err)
	}
}
