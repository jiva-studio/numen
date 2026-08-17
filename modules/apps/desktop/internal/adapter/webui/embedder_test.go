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
	embedding.Use = embed.UseService
	embedding.Service.Name = "asked-for"
	// Nowhere: this test wants the embedder built, not called.
	embedding.Service.BaseURL = "http://127.0.0.1:1/v1"
	t.Setenv(embed.KeyEnvVar, "sk-test")

	opened, err := webui.Open(t.Context(), vault(t, embedding), os.Stderr)
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
	opened, err := webui.Open(t.Context(), vault(t, embed.Config{}), os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	if opened.Embedder != nil {
		t.Errorf("got an embedder: %v", opened.Embedder.Model())
	}
}

// The vault says whether it has work in hand, and a client asks again for as long
// as it does. The counts of that work move with no file changing, so nothing else
// can say there is more to come: never set, and the numbers stand still while a
// library is read; never cleared, and a client asks every two seconds for ever.
func TestAVaultSaysWhileItIsBeingRead(t *testing.T) {
	cfg := vault(t, embed.Config{})

	opened, err := webui.Open(t.Context(), cfg, os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = opened.Close() })

	// Set before the reading goroutine starts, so a client asking between opening
	// and the first read is told there is more to come.
	if !opened.API.Busy.Load() {
		t.Error("a vault just opened says it has nothing in hand")
	}

	waitFor(t, func() bool { return !opened.API.Busy.Load() })
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
