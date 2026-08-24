package webui_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/webui"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

// vault gives a folder an identity and registers it, and answers with the
// configuration that opens it.
func vault(t *testing.T, embedding embed.Config) container.Config {
	t.Helper()
	root := t.TempDir()
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}
	cfg := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
		Embedding:    embedding,
	}
	registry, err := cfg.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (usecase.Add{
		Registry: registry,
		Identity: cfg.VaultIdentity(),
		Now:      time.Now,
	}).Execute(root, "asked"); err != nil {
		t.Fatal(err)
	}
	return cfg
}

// The embedder an installation configured reaches the search as well as the
// filling of the index, and this asks for it.
func TestTheEmbedderConfiguredIsTheOneOnHand(t *testing.T) {
	embedding := embed.Defaults()
	embedding.Model.Name = "asked-for"
	embedding.Indexing.Use = embed.UseService
	embedding.Indexing.Service.Name = "asked-for"
	// Nowhere: this test wants the embedder built, not called.
	embedding.Indexing.Service.BaseURL = "http://127.0.0.1:1/v1"
	t.Setenv(embed.KeyEnvVar, "sk-test")

	opened, err := webui.Open(t.Context(), vault(t, embedding), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	if opened.Embedder == nil {
		t.Fatal("the vault opened with no embedder on hand")
	}
	if got := opened.Embedder.Model().Name; got != "asked-for" {
		t.Errorf("got %q", got)
	}
}

// An installation with no model opens, and says so by having none.
func TestAVaultWithNoModelOpensAnyway(t *testing.T) {
	opened, err := webui.Open(t.Context(), vault(t, embed.Config{}), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	if opened.Embedder != nil {
		t.Errorf("got an embedder: %v", opened.Embedder.Model())
	}
}

// Reading a vault is work, and work is one entry in the list of what is being
// done. A pass that reports itself anywhere else draws a second card for the
// same work, with a count of its own that nobody can place.
func TestAVaultSaysWhatItIsDoingWhileItReadsItself(t *testing.T) {
	cfg := vault(t, embed.Config{})

	opened, err := webui.Open(t.Context(), cfg, "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	// Said before the reading goroutine starts, so a window that opens on a
	// fresh vault is shown the walk from its first moment.
	list := opened.API.Tasking.List()
	if len(list) != 1 {
		t.Errorf("a vault just opened is doing %+v, want the walk of its notes", list)
	}
	// Nobody asked for it, so it is drawn once it has lasted.
	for _, at := range list {
		if at.Asked {
			t.Errorf("%q says a person asked for it", at.Doing)
		}
	}

	// Each pass takes itself out when it ends, so a vault that has been read
	// says it is doing nothing.
	waitFor(t, func() bool { return len(opened.API.Tasking.List()) == 0 })
}

// waitFor gives a background reading its time and says what it was waiting for.
func waitFor(t *testing.T, done func() bool) {
	t.Helper()
	for range 200 {
		if done() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the vault never said it was finished, so a client asks for ever")
}
