//go:build !nomcp

package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/settings"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

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
			said := bool(c.sync)
			cfg.Naming = settings.Naming{SyncTitleAndFilename: &said}
			v := opened.Showing()
			raw := "---\ntitle: Entropy\n---\nA measure.\n"
			if err := os.WriteFile(
				filepath.Join(v.Path, "Entropy.md"), []byte(raw), 0o644,
			); err != nil {
				t.Fatal(err)
			}

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
