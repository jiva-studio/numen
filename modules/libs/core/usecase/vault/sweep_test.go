package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

const video = "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

// A person who deletes a link note deletes the video's words with it: what was
// fetched is the application's own, kept in its own folder, and a folder that
// grows with what nothing points at is one nobody can clean by hand.
func TestDeletingALinkNoteTakesWhatWasFetchedForIt(t *testing.T) {
	ctx := t.Context()
	v := testsupport.NewVault(t, map[string]string{
		"talk.md": "---\ntype: link\nurl: " + video + "\n---\n\nWhat I made of it.\n",
	})
	db := openIndex(t)
	stores := filesystem.DerivedStores{Area: text.Transcript, Areas: []string{text.Article, text.Copies}}
	held, err := stores.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	hash := text.Fingerprint([]byte(video))
	words := text.Artifact(text.Captions, hash)
	if err := held.Write(ctx, words, []byte("WEBVTT\n\n00:00.000 --> 00:02.000\nwhat was said\n")); err != nil {
		t.Fatal(err)
	}

	scan := scanner(filesystem.VaultReaders{}, db)
	scan.Derived = stores
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}
	if _, err := held.Read(ctx, words); err != nil {
		t.Fatalf("the words went before the note did: %v", err)
	}

	if err := os.Remove(filepath.Join(v.Path, "talk.md")); err != nil {
		t.Fatal(err)
	}
	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed != 1 {
		t.Fatalf("removed %d notes, want the one deleted", res.Removed)
	}
	if _, err := held.Read(ctx, words); err == nil {
		t.Error("the words are still here, and no note points at the address any more")
	}
}

// Two notes on one video hold what was fetched between them: it is named by the
// address, and one of them going leaves the other with its words.
func TestOneOfTwoNotesOnOneVideoGoing(t *testing.T) {
	ctx := t.Context()
	front := "---\ntype: link\nurl: " + video + "\n---\n\n"
	v := testsupport.NewVault(t, map[string]string{
		"mine.md":   front + "What I made of it.\n",
		"theirs.md": front + "What they made of it.\n",
	})
	db := openIndex(t)
	stores := filesystem.DerivedStores{Area: text.Transcript, Areas: []string{text.Article, text.Copies}}
	held, err := stores.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	words := text.Artifact(text.Captions, text.Fingerprint([]byte(video)))
	if err := held.Write(ctx, words, []byte("WEBVTT\n\n00:00.000 --> 00:02.000\nwhat was said\n")); err != nil {
		t.Fatal(err)
	}

	scan := scanner(filesystem.VaultReaders{}, db)
	scan.Derived = stores
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(v.Path, "mine.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}

	if _, err := held.Read(ctx, words); err != nil {
		t.Errorf("the other note lost the words it points at: %v", err)
	}
}

// A link note the vault holds nothing fetched for is reached as the walk finds
// it, where the settings ask for that. Off, which is the default, a walk
// reaches off the machine nowhere.
func TestALinkNoteIsFetchedUnaskedWhereTheSettingsSaySo(t *testing.T) {
	ctx := t.Context()
	v := testsupport.NewVault(t, map[string]string{
		"talk.md":     "---\ntype: link\nurl: " + video + "\n---\n\nWhat I made of it.\n",
		"ordinary.md": "# Entropy\n\nIt grows.\n",
	})
	db := openIndex(t)
	stores := filesystem.DerivedStores{Area: text.Transcript, Areas: []string{text.Article, text.Copies}}

	var asked []string
	scan := scanner(filesystem.VaultReaders{}, db)
	scan.Derived = stores
	scan.Types = db.Queries()
	scan.Fetches = func(_ context.Context, _ domain.Vault, path string) error {
		asked = append(asked, path)
		return nil
	}

	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != "talk.md" {
		t.Errorf("the walk reached %v, want the one note pointing at an address", asked)
	}
	if res.Fetched != 1 {
		t.Errorf("the walk reached %d addresses, want 1", res.Fetched)
	}
}

// A walk given nothing to fetch with reaches off the machine nowhere, which is
// what every walk does until the settings say otherwise.
func TestAWalkReachesNoAddressByItself(t *testing.T) {
	ctx := t.Context()
	v := testsupport.NewVault(t, map[string]string{
		"talk.md": "---\ntype: link\nurl: " + video + "\n---\n\nWhat I made of it.\n",
	})
	db := openIndex(t)
	scan := scanner(filesystem.VaultReaders{}, db)
	scan.Derived = filesystem.DerivedStores{Area: text.Transcript, Areas: []string{text.Article, text.Copies}}

	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	if res.Fetched != 0 {
		t.Errorf("a walk nobody asked reached %d addresses", res.Fetched)
	}
}
