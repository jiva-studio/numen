package version_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/index"
	"github.com/jiva-studio/numen/libs/core/adapter/index/version"
	"github.com/jiva-studio/numen/libs/core/internal/testsupport/indexfile"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const vault domain.VaultID = "01VAULT0000000000000000000"

func openRepository(t *testing.T) *version.Repository {
	t.Helper()
	db, err := index.Open(t.Context(), indexfile.Path(t), indexfile.SkipFlush())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db.TextVersions()
}

func newVersion(path, pluginID, key string) domain.TextVersion {
	return domain.TextVersion{
		Path: path, PluginID: pluginID, Key: key, Kind: "transcript", File: key + ".jsonl",
		SourceHash: "s", Hash: "h", CreateTime: time.Date(2026, 4, 2, 10, 0, 0, 123, time.UTC),
	}
}

func save(t *testing.T, repo *version.Repository, versions ...domain.TextVersion) {
	t.Helper()
	for _, v := range versions {
		if err := repo.SaveVersion(t.Context(), vault, v); err != nil {
			t.Fatal(err)
		}
	}
}

func getIDs(t *testing.T, repo *version.Repository, path string) []domain.TextVersionID {
	t.Helper()
	versions, err := repo.GetVersions(t.Context(), vault, path)
	if err != nil {
		t.Fatal(err)
	}
	var ids []domain.TextVersionID
	for _, v := range versions {
		ids = append(ids, v.GetID())
	}
	return ids
}

func TestSaveVersionKeepsTheOrderAndReplacesInPlace(t *testing.T) {
	repo := openRepository(t)
	first, second := newVersion("a.mp3", "p.one", "k"), newVersion("a.mp3", "p.two", "k")
	second.Source = first.GetID()
	save(t, repo, first, second)

	first.Hash, first.File = "h2", "k2.jsonl"
	save(t, repo, first)
	got, err := repo.GetVersions(t.Context(), vault, "a.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != first || got[1] != second {
		t.Fatalf("got %+v, want the replaced first version still first", got)
	}
}

func TestMoveVersionsFollowsAFileAndAFolder(t *testing.T) {
	repo := openRepository(t)
	save(t, repo,
		newVersion("talks/a.mp3", "p.one", "k"),
		newVersion("talks/old/b.mp3", "p.one", "k"),
		newVersion("talks-old/c.mp3", "p.one", "k"),
		newVersion("archive/a.mp3", "p.one", "stale"),
	)

	if err := repo.MoveVersions(t.Context(), vault, "talks/a.mp3", "archive/a.mp3"); err != nil {
		t.Fatal(err)
	}
	if got := getIDs(t, repo, "archive/a.mp3"); !slices.Equal(got, []domain.TextVersionID{"p.one/k"}) {
		t.Fatalf("the moved file has %v, want its own version alone", got)
	}
	if err := repo.MoveVersions(t.Context(), vault, "talks", "lectures"); err != nil {
		t.Fatal(err)
	}
	if got := getIDs(t, repo, "lectures/old/b.mp3"); len(got) != 1 {
		t.Fatalf("a file under the moved folder has %v", got)
	}
	if got := getIDs(t, repo, "talks-old/c.mp3"); len(got) != 1 {
		t.Fatalf("a folder beside the moved one lost its versions: %v", got)
	}
}

func TestRemovePluginVersionsReturnsWhatItRemoved(t *testing.T) {
	repo := openRepository(t)
	save(t, repo, newVersion("a.mp3", "p.one", "k"), newVersion("b.pdf", "p.one", "k"), newVersion("a.mp3", "p.two", "k"))

	removed, err := repo.RemovePluginVersions(t.Context(), vault, "p.one")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 2 || removed[0].Path != "a.mp3" || removed[1].Path != "b.pdf" {
		t.Fatalf("removed %+v", removed)
	}
	all, err := repo.ListVersions(t.Context(), vault)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].PluginID != "p.two" {
		t.Fatalf("left %+v, want the other plugin's version", all)
	}
	if err := repo.RemoveVersion(t.Context(), vault, "a.mp3", "p.two", "k"); err != nil {
		t.Fatal(err)
	}
	if got := getIDs(t, repo, "a.mp3"); len(got) != 0 {
		t.Fatalf("left %v", got)
	}
}
