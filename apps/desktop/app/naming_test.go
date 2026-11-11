package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func waitForIndex(t *testing.T, queries *query.Scenarios, v domain.Vault, path string) {
	t.Helper()
	for range 200 {
		titles, err := queries.Browse.GetTitles.Execute(t.Context(), v, []string{path})
		if err != nil {
			t.Fatal(err)
		}
		if len(titles.Missing) == 0 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s was never read into the index", path)
}

// TestTheAgentRenamesTheWayTheSettingsSay. `note_rename` calls the same use
// case the window does, and the tools pass the one setting to it.
func TestTheAgentRenamesTheWayTheSettingsSay(t *testing.T) {
	for name, c := range map[string]struct {
		sync     naming.SyncTitleAndFilename
		filename string
	}{
		"one name":   {sync: true, filename: "Disorder.md"},
		"told apart": {sync: false, filename: "Entropy.md"},
	} {
		t.Run(name, func(t *testing.T) {
			opened, cfg, cases := windowWithCases(t)
			response := fmt.Sprintf(`{"app":{"naming":{"syncTitleAndFilename":%v}}}`, bool(c.sync))
			if err := os.WriteFile(
				filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json"), []byte(response), 0o644,
			); err != nil {
				t.Fatal(err)
			}
			v := opened.State.GetCurrentVault()
			raw := "---\ntitle: Entropy\n---\nA measure.\n"
			if err := os.WriteFile(
				filepath.Join(v.Path, "Entropy.md"), []byte(raw), 0o644,
			); err != nil {
				t.Fatal(err)
			}
			// The note is in the index before it is renamed. A rename racing the
			// walk that first reads the file files it at two paths at once.
			waitForIndex(t, cases.Queries, v, "Entropy.md")

			notes := cases.Notes
			renamed, err := notes.Place.Rename.Execute(t.Context(), v, "Entropy.md", "Disorder")
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != c.filename {
				t.Errorf("the note is filed at %q", renamed.Path)
			}
			body, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(c.filename)))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "title: Disorder") {
				t.Errorf("the title was not written:\n%s", body)
			}
		})
	}
}
