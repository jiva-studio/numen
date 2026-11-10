package service

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

// switchableVault is the vault the host shows, which a test sets to another vault of the host.
type switchableVault struct {
	*hosttest.Vault

	mu     sync.Mutex
	active domain.VaultID
}

func (v *switchableVault) setActive(id domain.VaultID) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.active = id
}

func (v *switchableVault) getActive() *hosttest.Vault {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.Vault.With(v.active)
}

func (v *switchableVault) GetVault(ctx context.Context) (domain.Vault, error) {
	return v.getActive().GetVault(ctx)
}

func (v *switchableVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	return v.getActive().Read(ctx, path)
}

func TestCreateTextKeepsTheVaultActiveOnEntry(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "pdf", "testdata", "tiny.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	atlas := hosttest.OnDisk(t, "Atlas", map[string]string{"books/tiny.pdf": string(raw)})
	botany := hosttest.New("Botany")
	atlas.Beside(botany)
	files := &switchableVault{Vault: atlas, active: "Atlas"}
	vaults := func(id domain.VaultID) vaultFiles {
		if id == "" {
			return files
		}
		return atlas.With(id)
	}
	texts := hosttest.NewTexts("Atlas", pluginID)
	texts.File("books/tiny.pdf", raw)
	queue := hosttest.NewQueue()
	texts.SetStore(pluginID, atlas.Store())
	server := newTextServerOver(t, vaults, atlas, texts, queue)
	t.Cleanup(server.Close)
	held, err := queue.EnqueueWork(t.Context(), host.Work{IsUserStarted: true, OnWaiting: nil})
	if err != nil {
		t.Fatal(err)
	}

	if err := server.CreateText(t.Context(), TextRequest{Vault: "", Path: "books/tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	files.setActive("Botany")
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	server.Wait()

	if text := readLive(t, texts, "books/tiny.pdf"); text.Version().Kind != textKind {
		t.Errorf("Atlas has the live version %+v, want the text layer", text.Version())
	}
	inBotany, err := texts.With("Botany").ListVersions(t.Context(), "books/tiny.pdf")
	if err != nil || len(inBotany.Versions) != 0 {
		t.Errorf("Botany lists %+v (%v), want no version", inBotany.Versions, err)
	}
}
