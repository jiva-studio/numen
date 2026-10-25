package file_test

import (
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// typed is the index, answering what each note is from a fixed table.
type typed struct {
	noteport.NoteQueries
	types     map[string]domain.NoteType
	err       error
	requested [][]string
}

func (q *typed) GetTypes(_ context.Context, _ domain.VaultID, paths []string) (map[string]domain.NoteType, error) {
	q.requested = append(q.requested, paths)
	return q.types, q.err
}

func TestAFolderListsItsEntriesAndAsksTheIndexAboutTheNotes(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{
		"decks/spanish.md": "# spanish\n",
		"decks/ideas.md":   "# ideas\n",
		"decks/talk.mp3":   "sound",
		"decks/inner/a.md": "# a\n",
	})
	index := &typed{types: map[string]domain.NoteType{"decks/spanish.md": "deck"}}

	listing, err := file.NewGetFolderEntries(vaultreader.Readers{}, index).Execute(t.Context(), v, "decks")
	if err != nil {
		t.Fatal(err)
	}

	if len(listing.Entries) != 4 {
		t.Fatalf("the folder came back as %d entries", len(listing.Entries))
	}
	if listing.Types["decks/spanish.md"] != "deck" {
		t.Errorf("the deck is %v", listing.Types["decks/spanish.md"])
	}
	if len(index.requested) != 1 || len(index.requested[0]) != 2 {
		t.Errorf("the index was asked %v, want one question about the two notes", index.requested)
	}
}

func TestAFolderListsWithoutAnIndex(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"a.md": "# a\n"})

	listing, err := file.NewGetFolderEntries(vaultreader.Readers{}, nil).Execute(t.Context(), v, "")
	if err != nil || len(listing.Entries) == 0 || len(listing.Types) != 0 {
		t.Fatalf("listed %v with types %v, %v", listing.Entries, listing.Types, err)
	}
}

func TestAFolderThatIsNotThereIsNotListed(t *testing.T) {
	v := testsupport.NewVault(t, nil)

	if _, err := file.NewGetFolderEntries(vaultreader.Readers{}, nil).Execute(t.Context(), v, "nowhere"); err == nil {
		t.Fatal("a folder that is not there was listed")
	}
}

func TestAnIndexThatCannotAnswerFailsTheListing(t *testing.T) {
	v := testsupport.NewVault(t, map[string]string{"a.md": "# a\n"})
	failure := errors.New("index closed")

	_, err := file.NewGetFolderEntries(vaultreader.Readers{}, &typed{err: failure}).Execute(t.Context(), v, "")
	if err == nil || err.Error() != failure.Error() {
		t.Fatalf("error is %v, want the index's own", err)
	}
}

// unopenable is readers that cannot open the vault, for the reason a missing folder gives.
type unopenable struct{ vaultport.VaultReaders }

func (unopenable) Open(domain.Vault) (vaultport.VaultReader, error) {
	return nil, fs.ErrNotExist
}

func TestAVaultThatCannotBeOpenedIsNotAFolderThatIsMissing(t *testing.T) {
	v := testsupport.NewVault(t, nil)

	_, err := file.NewGetFolderEntries(unopenable{}, nil).Execute(t.Context(), v, "")
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("error is %v, want a failure that is not a missing folder", err)
	}
}
