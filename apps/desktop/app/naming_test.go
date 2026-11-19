package app

import (
	"connectrpc.com/connect"
	"fmt"
	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1/numenv1connect"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
)

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
			cfg := newWindowConfig(t)
			root := t.TempDir()
			response := fmt.Sprintf(`{"app":{"naming":{"syncTitleAndFilename":%v}}}`, bool(c.sync))
			if err := os.WriteFile(
				filepath.Join(filepath.Dir(cfg.RegistryPath), "numen.json"), []byte(response), 0o644,
			); err != nil {
				t.Fatal(err)
			}
			const path = "Entropy.md"
			if err := os.WriteFile(filepath.Join(root, path), []byte("---\ntitle: Entropy\n---\nA measure.\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			opened := openWindowAt(t, cfg, root)
			server := httptest.NewServer(opened.NewHandler(http.NotFoundHandler(), numenv1connect.VaultServiceName, numenv1connect.NoteServiceName))
			t.Cleanup(server.Close)
			vaults := numenv1connect.NewVaultServiceClient(server.Client(), server.URL)
			deadline := time.After(10 * time.Second)
			for {
				state, err := vaults.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
				if err != nil {
					t.Fatal(err)
				}
				if state.Msg.GetScan().GetIsReady() {
					break
				}
				if reason := state.Msg.GetScan().GetError(); reason != "" {
					t.Fatalf("initial scan failed: %s", reason)
				}
				select {
				case <-deadline:
					t.Fatal("initial scan did not finish")
				case <-time.After(10 * time.Millisecond):
				}
			}

			client := numenv1connect.NewNoteServiceClient(server.Client(), server.URL)
			renamed, err := client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{Path: path, Title: "Disorder"}))
			if err != nil {
				t.Fatal(err)
			}
			if renamed.Msg.GetPath() != c.filename {
				t.Errorf("the note is filed at %q", renamed.Msg.GetPath())
			}
			body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(c.filename)))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(body), "title: Disorder") {
				t.Errorf("the title was not written:\n%s", body)
			}
		})
	}
}
