package note_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func (c changeFixture) saveVersion(t *testing.T, path, key string) {
	t.Helper()
	err := c.db.TextVersions().SaveVersion(t.Context(), c.vault.ID, domain.TextVersion{
		Path: path, PluginID: "org.example.transcriber", Key: key, Kind: "transcript", File: key + ".jsonl",
		InputHash: "in", TextHash: "out", CreateTime: time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (c changeFixture) getVersionKeys(t *testing.T, path string) []string {
	t.Helper()
	found, err := c.db.TextVersions().GetVersions(t.Context(), c.vault.ID, path)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, v := range found {
		keys = append(keys, v.Key)
	}
	return keys
}

func TestTrashingAFileMovesItsVersionsIntoTheTrashAndRestoringBringsThemBack(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"talks/Heat.md": "# Heat\n"})
	c.saveVersion(t, "talks/Heat.md", "first")

	removed, err := c.remove().Execute(t.Context(), c.vault, "talks/Heat.md")
	if err != nil {
		t.Fatal(err)
	}

	if got := c.getVersionKeys(t, "talks/Heat.md"); len(got) != 0 {
		t.Errorf("the path of the trashed file still has versions: %v", got)
	}
	if got := c.getVersionKeys(t, removed.Trashed); !slices.Equal(got, []string{"first"}) {
		t.Errorf("the trashed file has versions %v", got)
	}

	if _, err := c.move().Execute(t.Context(), c.vault, removed.Trashed, "talks/Heat.md"); err != nil {
		t.Fatal(err)
	}
	if got := c.getVersionKeys(t, "talks/Heat.md"); !slices.Equal(got, []string{"first"}) {
		t.Errorf("the restored file has versions %v", got)
	}
}

func TestTrashingAFolderMovesTheVersionsOfItsFiles(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"talks/Heat.md": "# Heat\n", "talks-old/Cold.md": "# Cold\n"})
	c.saveVersion(t, "talks/Heat.md", "first")
	c.saveVersion(t, "talks-old/Cold.md", "second")

	removed, err := c.remove().Execute(t.Context(), c.vault, "talks")
	if err != nil {
		t.Fatal(err)
	}

	if got := c.getVersionKeys(t, removed.Trashed+"/Heat.md"); !slices.Equal(got, []string{"first"}) {
		t.Errorf("the file in the trashed folder has versions %v", got)
	}
	if got := c.getVersionKeys(t, "talks-old/Cold.md"); !slices.Equal(got, []string{"second"}) {
		t.Errorf("a folder beside the trashed one has versions %v", got)
	}
}
