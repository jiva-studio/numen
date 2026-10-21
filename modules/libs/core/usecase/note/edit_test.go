package note_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	"github.com/jiva-studio/numen/modules/libs/notes/markdown"
)

var errDiskGone = errors.New("the disk did not answer")

// brokenReaders is a vault on disk whose reader answers with the errors it is
// given.
type brokenReaders struct {
	open error
	read error
	stat error
}

func (b brokenReaders) Open(v domain.Vault) (port.VaultReader, error) {
	if b.open != nil {
		return nil, b.open
	}
	inner, err := filesystem.VaultReaders{}.Open(v)
	if err != nil {
		return nil, err
	}
	return brokenReader{VaultReader: inner, on: b}, nil
}

type brokenReader struct {
	port.VaultReader
	on brokenReaders
}

func (r brokenReader) Read(ctx context.Context, path string) ([]byte, error) {
	if r.on.read != nil {
		return nil, r.on.read
	}
	return r.VaultReader.Read(ctx, path)
}

func (r brokenReader) GetFingerprint(ctx context.Context, path string) (domain.Fingerprint, error) {
	if r.on.stat != nil {
		return domain.Fingerprint{}, r.on.stat
	}
	return r.VaultReader.GetFingerprint(ctx, path)
}

// brokenWriters is a vault on disk whose writers answer with the errors they
// are given.
type brokenWriters struct {
	port.VaultWriters
	hold error
	open error
}

func (b brokenWriters) Open(v domain.Vault) (port.VaultWriter, error) {
	if b.open != nil {
		return nil, b.open
	}
	return b.VaultWriters.Open(v)
}

func (b brokenWriters) Lock(ctx context.Context, v domain.Vault) (func(), error) {
	if b.hold != nil {
		return nil, b.hold
	}
	return b.VaultWriters.Lock(ctx, v)
}

func titleHeat(d *markdown.Document) error { return d.SetTitle("Heat") }

func (c changing) exists(path string) bool {
	_, err := os.Stat(filepath.Join(c.vault.Path, filepath.FromSlash(path)))
	return err == nil
}

func (c changing) overwrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(c.vault.Path, filepath.FromSlash(path)), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAnEditNeedsNoIndexToLeaveTheNoteSaved(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	edit := note.NewEdit(filesystem.VaultReaders{}, filesystem.VaultWriters{}, nil, time.Now)

	content, err := edit.Apply(t.Context(), c.vault, "Heat.md", titleHeat)
	if err != nil {
		t.Fatalf("an edit with no index to level failed: %v", err)
	}

	if content.IsZero() {
		t.Error("the write landed and no fingerprint came back")
	}
	if body := c.read(t, "Heat.md"); body == "# Heat\n" {
		t.Errorf("the change was not written:\n%s", body)
	}
}

func TestAnEditThatCannotReachTheVaultChangesNothing(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		readers port.VaultReaders
		writers port.VaultWriters
	}{
		{"the write lock fails", filesystem.VaultReaders{}, brokenWriters{VaultWriters: filesystem.VaultWriters{}, hold: errDiskGone}},
		{"the vault will not open for reading", brokenReaders{open: errDiskGone}, filesystem.VaultWriters{}},
		{"the vault will not open for writing", filesystem.VaultReaders{}, brokenWriters{VaultWriters: filesystem.VaultWriters{}, open: errDiskGone}},
		{"the note will not read", brokenReaders{read: errDiskGone}, filesystem.VaultWriters{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})

			content, err := note.NewEdit(tc.readers, tc.writers, c.index, time.Now).
				Apply(t.Context(), c.vault, "Heat.md", titleHeat)

			if !errors.Is(err, errDiskGone) {
				t.Errorf("want the error the vault gave, got %v", err)
			}
			if !content.IsZero() {
				t.Errorf("a fingerprint came back for a write that never landed: %+v", content)
			}
			if body := c.read(t, "Heat.md"); body != "# Heat\n" {
				t.Errorf("the note was written anyway:\n%s", body)
			}
		})
	}
}

// The file is asked about only when something is held against what it is: a
// tab that presents what it last saw, or a caller that said nothing and is held
// to what is read just now.
func TestAnEditAsksWhatTheFileIsOnlyWhereSomethingIsHeldAgainstIt(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		known     bool
		overwrite bool
		baseline  bool
		wantStat  bool
	}{
		{"a caller that said nothing", false, false, false, true},
		{"a caller that names the file it read", true, false, false, false},
		{"a caller writing over what is there", false, true, false, false},
		{"a caller that names the file and writes over it", true, true, false, false},
		{"a tab that presents what it saw", true, false, true, true},
		{"a tab that saw it and writes over it", false, true, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
			current := c.openNote(t, "Heat.md")
			request := 0
			edit := note.NewEdit(
				watchedReaders{inner: filesystem.VaultReaders{}, stat: func(string) { request++ }},
				filesystem.VaultWriters{}, c.index, time.Now)
			if tc.known {
				edit.Fingerprint = current.Fingerprint
			}
			edit.ShouldOverwrite = tc.overwrite
			if tc.baseline {
				edit.Baseline = current
			}

			if _, err := edit.Apply(t.Context(), c.vault, "Heat.md", titleHeat); err != nil {
				t.Fatal(err)
			}

			if got := request > 0; got != tc.wantStat {
				t.Errorf("asked what the file is = %v, want %v", got, tc.wantStat)
			}
		})
	}
}

// A caller that named no fingerprint is held to the file as it stood before it
// was read, so a save that lands in between is not written over.
func TestAnEditThatNamedNoFingerprintIsHeldToWhatItReadJustNow(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	theirs := "# Heat\n\nSaved in the meantime, and longer.\n"
	edit := note.NewEdit(
		watchedReaders{inner: filesystem.VaultReaders{}, read: func(string) { c.overwrite(t, "Heat.md", theirs) }},
		filesystem.VaultWriters{}, c.index, time.Now)

	_, err := edit.Apply(t.Context(), c.vault, "Heat.md", titleHeat)

	if !errors.Is(err, port.ErrStale) {
		t.Fatalf("want ErrStale, got %v", err)
	}
	if body := c.read(t, "Heat.md"); body != theirs {
		t.Errorf("the edit wrote over a save it never read:\n%s", body)
	}
}

func TestAnEditOfANoteThatCannotBeLookedAtIsRefused(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		stat error
		want error
	}{
		{"the vault will not answer", errDiskGone, errDiskGone},
		{"the vault holds no note there", fs.ErrNotExist, note.ErrNoNote},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
			edit := note.NewEdit(brokenReaders{stat: tc.stat}, filesystem.VaultWriters{}, c.index, time.Now)

			_, err := edit.Apply(t.Context(), c.vault, "Heat.md", titleHeat)

			if !errors.Is(err, tc.want) {
				t.Errorf("want %v, got %v", tc.want, err)
			}
			if body := c.read(t, "Heat.md"); body != "# Heat\n" {
				t.Errorf("the note was written anyway:\n%s", body)
			}
		})
	}
}

func TestAnEditOfANoteTheVaultDoesNotHoldMakesNothing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Entropy.md": "# Entropy\n"})
	elsewhere := c.openNote(t, "Entropy.md").Fingerprint

	for _, target := range []domain.Fingerprint{{}, elsewhere} {
		edit := c.editWith(filesystem.VaultReaders{}, filesystem.VaultWriters{})
		edit.Fingerprint = target

		_, err := edit.Apply(t.Context(), c.vault, "Heat.md", titleHeat)

		if !errors.Is(err, note.ErrNoNote) {
			t.Errorf("want ErrNoNote, got %v", err)
		}
		if c.exists("Heat.md") {
			t.Error("an edit made a note that was not there")
		}
	}
}

func (c changing) editWith(readers port.VaultReaders, writers port.VaultWriters) note.Edit {
	return note.NewEdit(readers, writers, c.index, time.Now)
}

// A caller writing over what is there makes the note when it is missing, and
// every other failure of the read is still a failure.
func TestAnOverwriteStopsAtAReadThatFailsForAnotherReason(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": "# Heat\n"})
	edit := c.editWith(brokenReaders{read: errDiskGone}, filesystem.VaultWriters{})
	edit.ShouldOverwrite = true

	_, err := edit.Apply(t.Context(), c.vault, "Heat.md", titleHeat)

	if !errors.Is(err, errDiskGone) {
		t.Errorf("want the error the vault gave, got %v", err)
	}
	if body := c.read(t, "Heat.md"); body != "# Heat\n" {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestAnEditOfFrontmatterThatCannotBeReadChangesNothing(t *testing.T) {
	t.Parallel()
	const broken = "---\ntitle: [unclosed\n---\n# Heat\n"
	c := openChanging(t, map[string]string{"Heat.md": broken})
	called := false

	_, err := c.editWith(filesystem.VaultReaders{}, filesystem.VaultWriters{}).
		Apply(t.Context(), c.vault, "Heat.md", func(*markdown.Document) error {
			called = true
			return nil
		})

	if err == nil || errors.Unwrap(err) == nil || !strings.Contains(err.Error(), "Heat.md") {
		t.Errorf("want the parse failure wrapped under the path, got %v", err)
	}
	if called {
		t.Error("the change ran over a note that did not open")
	}
	if body := c.read(t, "Heat.md"); body != broken {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

func TestAnEditThatCannotStampAnIdentifierWritesNothing(t *testing.T) {
	t.Parallel()
	const half = "---\ntitle: Heat\n"
	c := openChanging(t, map[string]string{"Heat.md": half})

	_, err := c.editWith(filesystem.VaultReaders{}, filesystem.VaultWriters{}).
		Apply(t.Context(), c.vault, "Heat.md", func(*markdown.Document) error { return nil })

	if !errors.Is(err, markdown.ErrUnterminated) {
		t.Errorf("want ErrUnterminated, got %v", err)
	}
	if body := c.read(t, "Heat.md"); body != half {
		t.Errorf("the note was written anyway:\n%s", body)
	}
}

// bytesWithNoNote answers every read with bytes and a missing file.
type bytesWithNoNote struct{ brokenReaders }

func (b bytesWithNoNote) Open(v domain.Vault) (port.VaultReader, error) {
	inner, err := b.brokenReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return staleBytesReader{VaultReader: inner}, nil
}

type staleBytesReader struct{ port.VaultReader }

func (staleBytesReader) Read(context.Context, string) ([]byte, error) {
	return []byte("Left over, not the person's."), fs.ErrNotExist
}

func TestAnOverwriteOfAMissingNoteContainsOnlyTheNewText(t *testing.T) {
	t.Parallel()
	c := openChanging(t, nil)
	edit := c.editWith(bytesWithNoNote{}, filesystem.VaultWriters{})
	edit.ShouldOverwrite = true

	if _, err := edit.Apply(t.Context(), c.vault, "Heat.md", func(d *markdown.Document) error {
		return d.SetBody("Mine.\n")
	}); err != nil {
		t.Fatal(err)
	}

	if body := c.read(t, "Heat.md"); body != "Mine.\n" {
		t.Errorf("the note holds %q", body)
	}
}
