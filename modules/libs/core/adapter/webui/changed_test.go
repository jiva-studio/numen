package webui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// editable is a vault with a read and a save on it and nothing behind them.
// What is asked here is what the schema carries.
func editable(t *testing.T, notes map[string]string) *API {
	t.Helper()
	api := &API{
		Notes: Notes{
			Read:  &note.Read{Readers: filesystem.VaultReaders{}},
			Write: &note.Write{Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}},
		},
	}
	api.show(testsupport.NewVault(t, notes))
	return api
}

func at(t *testing.T, api *API, path string) *v1.LastRead {
	t.Helper()
	out, err := api.ReadNote(t.Context(), connect.NewRequest(&v1.ReadNoteRequest{Path: path}))
	if err != nil {
		t.Fatal(err)
	}
	if out.Msg.GetAt() == nil {
		t.Fatalf("the read of %s carried no fingerprint", path)
	}
	return &v1.LastRead{Prose: out.Msg.GetBody(), At: out.Msg.GetAt()}
}

// A note holding prose the client has not read is its own answer. A refusal is
// something a tab can do nothing about, and this one is a question for the
// person.
func TestAWriteOverProseTheClientNeverReadIsAnsweredChanged(t *testing.T) {
	api := editable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	seen := at(t, api, "Entropy.md")

	theirs := "# Entropy\n\nTheirs.\n"
	on := filepath.Join(api.Showing().Path, "Entropy.md")
	if err := os.WriteFile(on, []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := api.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Entropy.md",
		Body: "# Entropy\n\nMine.\n",
		Seen: seen,
	}))
	if err != nil {
		t.Fatalf("a note that changed came back as an error: %v", err)
	}
	if refusal := out.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_STALE {
		t.Errorf("a note that changed was answered %v", refusal)
	}
	if out.Msg.GetAt() != nil {
		t.Error("a write that wrote nothing answered with a fingerprint")
	}

	raw, err := os.ReadFile(on)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != theirs {
		t.Errorf("a write that was stopped wrote anyway:\n%s", raw)
	}
}

// The fingerprint a write answers with is what the client presents at its next
// write, and it is what lets a sitting hold more than one save.
func TestAWriteAnswersWithTheFileItProduced(t *testing.T) {
	api := editable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	seen := at(t, api, "Entropy.md")

	first, err := api.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Entropy.md",
		Body: "# Entropy\n\nOne.\n",
		Seen: seen,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if first.Msg.GetRefusal() == v1.Refusal_REFUSAL_STALE {
		t.Fatal("the first write said the note changed")
	}
	if first.Msg.GetAt() == nil {
		t.Fatal("the write answered with no fingerprint")
	}

	second, err := api.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Entropy.md",
		Body: "# Entropy\n\nTwo.\n",
		Seen: &v1.LastRead{Prose: seen.GetProse(), At: first.Msg.GetAt()},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if second.Msg.GetRefusal() == v1.Refusal_REFUSAL_STALE {
		t.Error("the write after a write said the note changed")
	}

	raw, err := os.ReadFile(filepath.Join(api.Showing().Path, "Entropy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "# Entropy\n\nTwo.\n" {
		t.Errorf("the second write did not land:\n%s", raw)
	}
}

// beaten is a vault's readers with something writing the note the moment it has
// been looked at, which is what a writer outside this process looks like from
// in here.
type beaten struct {
	port.VaultReaders
	after func(path string)
}

func (b beaten) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := b.VaultReaders.Open(v)
	if err != nil {
		return nil, err
	}
	return beatenReader{VaultReader: reader, after: b.after}, nil
}

type beatenReader struct {
	port.VaultReader
	after func(path string)
}

func (b beatenReader) Stat(ctx context.Context, path string) (domain.Fingerprint, error) {
	ref, err := b.VaultReader.Stat(ctx, path)
	b.after(path)
	return ref, err
}

// A join reads a note, splices its frontmatter and puts it back, and the note
// can move between the two. What the person meets is an answer and not a
// transport error.
func TestAJoinOverANoteThatMovedIsAnsweredChanged(t *testing.T) {
	api := editable(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
	})
	on := filepath.Join(api.Showing().Path, "Heat.md")

	var once sync.Once
	api.Notes.Linking = &note.EditLinks{
		Readers: beaten{VaultReaders: filesystem.VaultReaders{}, after: func(path string) {
			if path != "Heat.md" {
				return
			}
			once.Do(func() {
				if err := os.WriteFile(on, []byte("# Heat\n\nSomebody else's, longer.\n"), 0o644); err != nil {
					t.Error(err)
				}
			})
		}},
		Writers: filesystem.VaultWriters{},
	}

	out, err := api.WriteLink(t.Context(), connect.NewRequest(&v1.WriteLinkRequest{
		Path: "Heat.md",
		Link: &v1.NewLink{To: "Entropy.md", Role: v1.Role_ROLE_PARENT},
	}))
	if err != nil {
		t.Fatalf("a note that changed came back as an error: %v", err)
	}
	if refusal := out.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_STALE {
		t.Errorf("a note that changed was answered %v", refusal)
	}

	raw, err := os.ReadFile(on)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "to: Entropy") {
		t.Errorf("a join that was stopped wrote anyway:\n%s", raw)
	}
}
