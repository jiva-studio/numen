package editor

import (
	"io"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// openedWindow is a window put together the way a person's window is: through
// Open, on a vault of this test's own.
func openedWindow(t *testing.T) *Installation {
	t.Helper()

	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	add := vaults.Add{Identity: cfg.VaultIdentity(), Registry: registry, Now: time.Now}
	if _, err := add.Execute(root, "one"); err != nil {
		t.Fatal(err)
	}

	opened, err := Open(t.Context(), cfg, "one", io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })
	return opened
}

// A note saved through the window is in the index when the save is answered.
// Search, links and headings are all asked of the index, so a save that levels
// nothing is a note the vault cannot find until the watch happens to come past
// — and on a vault nothing can watch, until the window is opened again.
//
// The writer the window saves through is made with what levels it, and this
// fails where it is not.
func TestANoteSavedThroughTheWindowIsFindableAtOnce(t *testing.T) {
	opened := openedWindow(t)

	const body = "tetragrammaton is a word nothing else in this vault holds"
	written, err := opened.API.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Kept.md",
		Body: body,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refused := written.Msg.GetError(); refused != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the save was refused: %v", refused)
	}

	found, err := opened.Index.Passages().Lexical(
		t.Context(), opened.Showing().ID, "tetragrammaton",
		[]domain.SourceKind{domain.KindNote}, 10, false,
	)
	if err != nil {
		t.Fatal(err)
	}
	// The word is indexed over chunks, so the one note answers from each of its
	// own that holds it.
	if len(found) == 0 {
		t.Error("the note saved through the window is not found")
	}
	for _, p := range found {
		if p.Source != "Kept.md" {
			t.Errorf("the search answered with %s, and only Kept.md holds the word", p.Source)
		}
	}
}
