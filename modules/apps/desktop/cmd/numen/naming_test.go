//go:build !nomcp

package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// indexed waits until the vault holds the note at this path, which is what says
// the walk that reads a file it has just been given is over.
func indexed(t *testing.T, opened *webui.Opened, v domain.Vault, path string) {
	t.Helper()
	for range 200 {
		shown, err := opened.Index.Queries().Notes(t.Context(), v.ID, []string{path})
		if err != nil {
			t.Fatal(err)
		}
		if _, held := shown[path]; held {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%s was never read into the index", path)
}

// TestTheAgentRenamesTheWayTheSettingsSay. `note_rename` reaches the same use
// case the window does, and the tools carry the one setting to it.
func TestTheAgentRenamesTheWayTheSettingsSay(t *testing.T) {
	for name, c := range map[string]struct {
		sync note.Sync
		at   string
	}{
		"one name":   {sync: true, at: "Disorder.md"},
		"told apart": {sync: false, at: "Entropy.md"},
	} {
		t.Run(name, func(t *testing.T) {
			opened, cfg := windowOn(t)
			said := fmt.Sprintf(`{"naming":{"sync_title_and_filename":%v}}`, bool(c.sync))
			if err := os.WriteFile(
				filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json"), []byte(said), 0o644,
			); err != nil {
				t.Fatal(err)
			}
			v := opened.Showing()
			raw := "---\ntitle: Entropy\n---\nA measure.\n"
			if err := os.WriteFile(
				filepath.Join(v.Path, "Entropy.md"), []byte(raw), 0o644,
			); err != nil {
				t.Fatal(err)
			}
			// The note is in the index before it is renamed. A rename racing the
			// walk that first reads the file files it at two paths at once.
			indexed(t, opened, v, "Entropy.md")

			core := agentCore(cfg, opened, v.Path, io.Discard)
			renamed, err := core.Rename.Execute(t.Context(), v, "Entropy.md", "Disorder")
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Path != c.at {
				t.Errorf("the note is filed at %q", renamed.Path)
			}
			body, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(c.at)))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "title: Disorder") {
				t.Errorf("the title was not written:\n%s", body)
			}
		})
	}
}
