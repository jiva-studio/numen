package recognition_test

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/wasm"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/internal/adapter/document"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/usecase/recognition"
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

func (v *switchableVault) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return v.getActive().StatFile(ctx, path)
}

func TestStartKeepsTheVaultActiveOnEntry(t *testing.T) {
	s := newScene(t)
	chemistry := hosttest.New("Chemistry")
	s.vault.Beside(chemistry)
	files := &switchableVault{Vault: s.vault, active: "Physics"}
	documents := document.NewReader()
	run, err := recognition.New(
		recognition.Host{Vaults: func(id domain.VaultID) recognition.Files {
			if id == "" {
				return files
			}
			return s.vault.With(id)
		}, Store: foldersOf(s.store), Texts: s.textsOf, Queue: s.queue, Documents: documents},
		func(context.Context) (recognition.Recogniser, error) { return s.recogniser, nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	compilation := wasm.NewCompilationCache(os.Getenv(cacheEnv))
	t.Cleanup(func() {
		if err := compilation.Close(context.WithoutCancel(t.Context())); err != nil {
			t.Error(err)
		}
	})
	documents.SetCompilationCache(compilation)
	held, err := s.queue.EnqueueWork(t.Context(), host.Work{IsUserStarted: true, OnWaiting: nil})
	if err != nil {
		t.Fatal(err)
	}

	runEnd, isStarted, err := run.Start(t.Context(), recognition.Request{Vault: "", Path: "books/scan.pdf"})
	if err != nil || !isStarted {
		t.Fatalf("Start = %v, %v", isStarted, err)
	}
	files.setActive("Chemistry")
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}
	<-runEnd

	if text := s.readLive(t, "books/scan.pdf"); text.Version().PluginID != pluginID {
		t.Errorf("Physics has the live version %+v, want the recognition", text.Version())
	}
	inChemistry, err := s.texts.With("Chemistry").ListVersions(t.Context(), "books/scan.pdf")
	if err != nil || len(inChemistry.Versions) != 0 {
		t.Errorf("Chemistry lists %+v (%v), want no version", inChemistry.Versions, err)
	}
}
