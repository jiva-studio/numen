package container

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
)

const (
	transcriber = "org.example.transcription"
	talkPath    = "talks/Morning talk.mp3"
)

// putTranscript records a transcript of the talk by a plugin this installation
// does not have, with its file in that plugin's folder.
func putTranscript(t *testing.T, db *Index, v domain.Vault, words string) domain.TextVersion {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, talkPath))
	if err != nil {
		t.Fatal(err)
	}
	files, err := textfiles.Stores{}.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := files.WriteTextFile(t.Context(), transcriber, "talk.jsonl", []domain.TextSegment{{Text: words, Selector: `{"t":[0,4]}`}})
	if err != nil {
		t.Fatal(err)
	}
	version := domain.TextVersion{
		Path: talkPath, PluginID: transcriber, Key: "talk", Kind: "transcript", File: "talk.jsonl",
		SourceHash: domain.HashContent(raw), Hash: hash, CreateTime: time.Date(2026, 5, 1, 8, 0, 0, 0, time.UTC),
	}
	repo, err := indexVersions{repo: db.db.TextVersions()}.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveVersion(t.Context(), version); err != nil {
		t.Fatal(err)
	}
	return version
}

// readIntoIndex reads the vault into the index, with the text versions as the
// texts of its sources.
func readIntoIndex(t *testing.T, cfg Config, db *Index, v domain.Vault) {
	t.Helper()
	if _, err := cfg.Scan(db).Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	extract, err := cfg.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := extract.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
}

func newTalkVault(t *testing.T) domain.Vault {
	t.Helper()
	v := testsupport.NewVault(t, map[string]string{"Note.md": "# Note\n\nA short note.\n"})
	if err := os.MkdirAll(filepath.Join(v.Path, "talks"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(v.Path, talkPath), []byte("ID3 invented samples"), 0o600); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestATranscriptOfAPluginNotInstalledIsLiveAndFound(t *testing.T) {
	v := newTalkVault(t)
	cfg := Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	versions, err := cfg.NewTextVersions(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg.TextVersions = versions
	version := putTranscript(t, db, v, "The heron waits by the reeds.\n")

	readIntoIndex(t, cfg, db, v)
	found, err := cfg.NewSearch(db, nil, nil).Execute(t.Context(), v, "heron", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != talkPath || found[0].Producer != string(version.GetID()) || found[0].Selector != `{"t":[0,4]}` {
		t.Fatalf("found %+v, want the talk through its transcript", found)
	}
}

func TestRebuildingTheIndexKeepsTheTextVersions(t *testing.T) {
	v := newTalkVault(t)
	path := filepath.Join(t.TempDir(), "index.db")
	cfg := Config{IndexPath: path}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	version := putTranscript(t, db, v, "The heron waits by the reeds.\n")
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	cfg.ShouldRebuildIndex = true
	db, err = cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	versions, err := cfg.NewTextVersions(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg.TextVersions = versions
	readIntoIndex(t, cfg, db, v)
	if _, err := db.DeleteVectorsOfOtherRecipes(t.Context(), "recipe"); err != nil {
		t.Fatal(err)
	}
	if err := db.Compact(t.Context()); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(v.Path, talkPath))
	if err != nil {
		t.Fatal(err)
	}
	listing, err := versions.ListVersions(t.Context(), v, talkPath, domain.HashContent(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 || listing.Live != version.GetID() {
		t.Fatalf("after the rebuild the talk has %+v, want its transcript live", listing)
	}
}

func TestRenamingAFileKeepsItsTextVersions(t *testing.T) {
	v := newTalkVault(t)
	cfg := Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	versions, err := cfg.NewTextVersions(db)
	if err != nil {
		t.Fatal(err)
	}
	cfg.TextVersions = versions
	version := putTranscript(t, db, v, "The heron waits by the reeds.\n")
	readIntoIndex(t, cfg, db, v)

	const renamed = "talks/Heron talk.mp3"
	move := vault.NewMove(cfg.VaultWriters(), db.Links(), db.SourcesKnown(), db.Sources(), note.Move{})
	if _, err := move.Execute(t.Context(), v, talkPath, renamed); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(filepath.Join(v.Path, renamed))
	if err != nil {
		t.Fatal(err)
	}
	listing, err := versions.ListVersions(t.Context(), v, renamed, domain.HashContent(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 || listing.Live != version.GetID() || listing.Versions[0].Path != renamed {
		t.Fatalf("after the rename the talk has %+v, want its transcript live at the new path", listing)
	}
	if old, err := versions.ListVersions(t.Context(), v, talkPath, domain.HashContent(raw)); err != nil || len(old.Versions) != 0 {
		t.Fatalf("the old path still has %+v (%v)", old, err)
	}
}
