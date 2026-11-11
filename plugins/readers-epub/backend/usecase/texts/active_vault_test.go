package texts

import (
	"context"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-epub/backend/testsupport"
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
	raw := testsupport.CreateImageEpub(t)
	atlas := hosttest.New("Atlas")
	atlas.File("books/atlas.epub", string(raw))
	botany := hosttest.New("Botany")
	atlas.Beside(botany)
	files := &switchableVault{Vault: atlas, active: "Atlas"}
	vaults := func(id domain.VaultID) Vault {
		if id == "" {
			return files
		}
		return atlas.With(id)
	}
	texts := hosttest.NewTexts("Atlas", pluginID)
	texts.File("books/atlas.epub", raw)
	texts.SetStore(pluginID, atlas.Store())
	queue := hosttest.NewQueue()
	server := NewWriter(vaults, textsOf(texts), storeOf(atlas), queue)
	t.Cleanup(server.Close)
	held, err := queue.EnqueueWork(t.Context(), host.Work{IsUserStarted: true, OnWaiting: nil})
	if err != nil {
		t.Fatal(err)
	}

	if err := server.CreateText(t.Context(), Request{Vault: "", Path: "books/atlas.epub"}); err != nil {
		t.Fatal(err)
	}
	files.setActive("Botany")
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	server.Wait()

	if text := readLive(t, texts, "books/atlas.epub"); text.Version().Kind == "" {
		t.Errorf("Atlas has the live version %+v, want the text of the book", text.Version())
	}
	inBotany, err := texts.With("Botany").ListVersions(t.Context(), "books/atlas.epub")
	if err != nil || len(inBotany.Versions) != 0 {
		t.Errorf("Botany lists %+v (%v), want no version", inBotany.Versions, err)
	}
}
