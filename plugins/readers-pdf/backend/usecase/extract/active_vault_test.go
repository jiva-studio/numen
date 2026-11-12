package extract_test

import (
	"context"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/internal/testsupport"
	"github.com/jiva-studio/numen/plugins/readers-pdf/backend/usecase/extract"
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

func (v *switchableVault) getActiveVault() *hosttest.Vault {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.Vault.With(v.active)
}

func (v *switchableVault) GetVault(ctx context.Context) (domain.Vault, error) {
	return v.getActiveVault().GetVault(ctx)
}

func (v *switchableVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	return v.getActiveVault().Read(ctx, path)
}

func TestCreateTextKeepsTheVaultActiveOnEntry(t *testing.T) {
	raw := testsupport.ReadFixture(t, "tiny.pdf")
	atlas := hosttest.OnDisk(t, "Atlas", map[string]string{"books/tiny.pdf": string(raw)})
	botany := hosttest.New("Botany")
	atlas.Beside(botany)
	files := &switchableVault{Vault: atlas, active: "Atlas"}
	vaults := func(id domain.VaultID) extract.Files {
		if id == "" {
			return files
		}
		return atlas.With(id)
	}
	texts := hosttest.NewTexts("Atlas", pluginID)
	texts.File("books/tiny.pdf", raw)
	queue := hosttest.NewQueue()
	texts.SetStore(pluginID, atlas.Store())
	writer := testsupport.NewWriter(t, vaults, atlas, texts, queue)
	held, err := queue.EnqueueWork(t.Context(), host.Work{IsUserStarted: true, OnWaiting: nil})
	if err != nil {
		t.Fatal(err)
	}

	if err := writer.CreateText(t.Context(), extract.Request{Vault: "", Path: "books/tiny.pdf"}); err != nil {
		t.Fatal(err)
	}
	files.setActive("Botany")
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	writer.Wait()

	if text := testsupport.ReadLive(t, texts, "books/tiny.pdf"); text.Version().Kind != extract.TextKind {
		t.Errorf("Atlas has the live version %+v, want the text layer", text.Version())
	}
	inBotany, err := texts.With("Botany").ListVersions(t.Context(), "books/tiny.pdf")
	if err != nil || len(inBotany.Versions) != 0 {
		t.Errorf("Botany lists %+v (%v), want no version", inBotany.Versions, err)
	}
}
