package container

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/textfiles"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/usecase/filetext"
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
	extract, err := cfg.IndexTexts(db.Sources(), db.SourcesKnown(), v)
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

// newTalkIndex is the talk vault read into an index, with a transcript of the
// talk recorded.
func newTalkIndex(t *testing.T) (Config, *Index, domain.Vault, domain.TextVersion) {
	t.Helper()
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
	return cfg, db, v, version
}

// refreshPaths brings the named paths up to date, as the watch does with the
// paths it reports.
func refreshPaths(t *testing.T, cfg Config, db *Index, v domain.Vault, paths ...string) {
	t.Helper()
	if _, err := makeRefresh(cfg, db, db.Notes()).Execute(t.Context(), v, paths); err != nil {
		t.Fatal(err)
	}
}

func TestRenamingAFileOnDiskMovesItsTextVersions(t *testing.T) {
	cfg, db, v, version := newTalkIndex(t)
	const renamed = "talks/Heron talk.mp3"
	if err := os.Rename(filepath.Join(v.Path, talkPath), filepath.Join(v.Path, renamed)); err != nil {
		t.Fatal(err)
	}

	refreshPaths(t, cfg, db, v, talkPath, renamed)
	readIntoIndex(t, cfg, db, v)

	raw, err := os.ReadFile(filepath.Join(v.Path, renamed))
	if err != nil {
		t.Fatal(err)
	}
	listing, err := cfg.TextVersions.ListVersions(t.Context(), v, renamed, domain.HashContent(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 || listing.Live != version.GetID() || listing.Versions[0].Path != renamed {
		t.Fatalf("after the rename on disk the talk has %+v, want its transcript live at the new path", listing)
	}
	found, err := cfg.NewSearch(db, nil, nil).Execute(t.Context(), v, "heron", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != renamed {
		t.Fatalf("found %+v, want the talk at its new path", found)
	}
}

func TestDeletingAFileOnDiskKeepsItsTextVersions(t *testing.T) {
	cfg, db, v, version := newTalkIndex(t)
	raw, err := os.ReadFile(filepath.Join(v.Path, talkPath))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(v.Path, talkPath)); err != nil {
		t.Fatal(err)
	}

	refreshPaths(t, cfg, db, v, talkPath)

	listing, err := cfg.TextVersions.ListVersions(t.Context(), v, talkPath, domain.HashContent(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 || listing.Versions[0].GetID() != version.GetID() {
		t.Fatalf("after the delete the talk's path has %+v, want its transcript to stay", listing)
	}
	found, err := cfg.NewSearch(db, nil, nil).Execute(t.Context(), v, "heron", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("found %+v, want nothing for a deleted talk", found)
	}
}

// cutOnChange cuts a file again each time its versions change, as the index of
// the shown vault does.
type cutOnChange struct {
	t       *testing.T
	extract interface {
		IndexOne(ctx context.Context, v domain.Vault, path string) (filetext.IndexResult, error)
	}
}

func (c cutOnChange) ReportVersionsChanged(v domain.Vault, path string) {
	if _, err := c.extract.IndexOne(c.t.Context(), v, path); err != nil {
		c.t.Error(err)
	}
}

func TestDeletingPluginDataRemovesItsFolderAndVersions(t *testing.T) {
	cfg, db, v, _ := newTalkIndex(t)
	const other = "org.example.subtitles"
	files, err := textfiles.Stores{}.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := files.WriteTextFile(t.Context(), other, "notes.jsonl", []domain.TextSegment{{Text: "A kingfisher dives.\n"}})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(v.Path, talkPath))
	if err != nil {
		t.Fatal(err)
	}
	subtitles := domain.TextVersion{
		Path: talkPath, PluginID: other, Key: "notes", Kind: "subtitles", File: "notes.jsonl",
		SourceHash: domain.HashContent(raw), Hash: hash, CreateTime: time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC),
	}
	repo, err := indexVersions{repo: db.db.TextVersions()}.Open(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveVersion(t.Context(), subtitles); err != nil {
		t.Fatal(err)
	}
	extract, err := cfg.IndexTexts(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		t.Fatal(err)
	}
	cfg.TextVersions.SetListener(cutOnChange{t: t, extract: extract})
	data := EditorAssembly{cfg: cfg}.OpenPluginData()

	if err := data.DeletePluginData(t.Context(), v, transcriber); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(v.Path, ".numen", transcriber)); !os.IsNotExist(err) {
		t.Fatalf("the transcriber's folder is still there (%v)", err)
	}
	if _, err := os.Stat(filepath.Join(v.Path, ".numen", other, "notes.jsonl")); err != nil {
		t.Fatalf("the other plugin's file is gone: %v", err)
	}
	listing, err := cfg.TextVersions.ListVersions(t.Context(), v, talkPath, domain.HashContent(raw))
	if err != nil || len(listing.Versions) != 1 || listing.Live != subtitles.GetID() {
		t.Fatalf("after the delete the talk has %+v (%v), want the other plugin's version alone and live", listing, err)
	}
	found, err := cfg.NewSearch(db, nil, nil).Execute(t.Context(), v, "heron", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Fatalf("found %+v, want the deleted transcript out of the index", found)
	}
	found, err = cfg.NewSearch(db, nil, nil).Execute(t.Context(), v, "kingfisher", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Producer != string(subtitles.GetID()) {
		t.Fatalf("found %+v, want the talk through the other plugin's version", found)
	}
}

func TestRenamingAFileWhileTheCoreIsStoppedMovesItsTextVersions(t *testing.T) {
	v := newTalkVault(t)
	path := filepath.Join(t.TempDir(), "index.db")
	cfg := Config{IndexPath: path}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TextVersions, err = cfg.NewTextVersions(db); err != nil {
		t.Fatal(err)
	}
	version := putTranscript(t, db, v, "The heron waits by the reeds.\n")
	readIntoIndex(t, cfg, db, v)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	const renamed = "talks/Heron talk.mp3"
	if err := os.Rename(filepath.Join(v.Path, talkPath), filepath.Join(v.Path, renamed)); err != nil {
		t.Fatal(err)
	}

	db, err = cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if cfg.TextVersions, err = cfg.NewTextVersions(db); err != nil {
		t.Fatal(err)
	}
	readIntoIndex(t, cfg, db, v)

	raw, err := os.ReadFile(filepath.Join(v.Path, renamed))
	if err != nil {
		t.Fatal(err)
	}
	listing, err := cfg.TextVersions.ListVersions(t.Context(), v, renamed, domain.HashContent(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 1 || listing.Live != version.GetID() {
		t.Fatalf("after the rescan the talk has %+v, want its transcript live at the new path", listing)
	}
	found, err := cfg.NewSearch(db, nil, nil).Execute(t.Context(), v, "heron", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != renamed {
		t.Fatalf("found %+v, want the talk at its new path", found)
	}
}
