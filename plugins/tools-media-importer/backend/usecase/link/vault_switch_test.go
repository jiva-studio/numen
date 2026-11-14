package link_test

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/random"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// currentVault is the vault the host shows, which a person can switch to another while a run is on.
// With names a vault by identity and does not follow the switch.
type currentVault struct {
	*hosttest.Vault
	current *atomic.Pointer[hosttest.Vault]
}

func (v currentVault) GetVault(ctx context.Context) (domain.Vault, error) {
	return v.current.Load().GetVault(ctx)
}

func (v currentVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	return v.current.Load().Read(ctx, path)
}

func (v currentVault) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	return v.current.Load().Write(ctx, path, content, baseline)
}

func (v currentVault) Create(ctx context.Context, path string, content []byte) error {
	return v.current.Load().Create(ctx, path, content)
}

func (v currentVault) Scan(ctx context.Context, paths []string) error {
	return v.current.Load().Scan(ctx, paths)
}

func (v currentVault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	return v.current.Load().GetPathsOfType(ctx, noteType)
}

// currentStore is the plugin's store of the vault the host shows, switched with it.
type currentStore struct {
	*hosttest.Store
	current *atomic.Pointer[hosttest.Vault]
}

func (s currentStore) Read(ctx context.Context, name string) ([]byte, error) {
	return s.current.Load().Store().Read(ctx, name)
}

func (s currentStore) List(ctx context.Context) ([]host.StoreFile, error) {
	return s.current.Load().Store().List(ctx)
}

func (s currentStore) Append(ctx context.Context, name string, content []byte) error {
	return s.current.Load().Store().Append(ctx, name, content)
}

func (s currentStore) Write(ctx context.Context, name string, content []byte) error {
	return s.current.Load().Store().Write(ctx, name, content)
}

func (s currentStore) Delete(ctx context.Context, name string) error {
	return s.current.Load().Store().Delete(ctx, name)
}

func (s currentStore) Rename(ctx context.Context, from, to string) error {
	return s.current.Load().Store().Rename(ctx, from, to)
}

func TestADownloadWritesOnlyTheVaultItStartedIn(t *testing.T) {
	mills, harbor := hosttest.New("Mills"), hosttest.New("Harbor")
	mills.Beside(harbor)
	current := &atomic.Pointer[hosttest.Vault]{}
	current.Store(mills)
	texts := hosttest.NewTexts(mills.Vault().ID, pluginID)
	fetched := &sites{clip: bytes.Repeat([]byte("frame "), 100), started: make(chan struct{}), gate: make(chan struct{})}
	links := link.New(link.Host{
		PluginID: pluginID, Vaults: func(id domain.VaultID) link.Vault {
			if id == "" {
				current := currentVault{Vault: mills, current: current}
				return link.Vault{Files: current, Writer: current}
			}
			other := mills.With(id)
			return link.Vault{Files: other, Writer: other}
		}, Store: func(id domain.VaultID) link.Folder {
			if id == "" {
				active := currentStore{Store: mills.Store(), current: current}
				return link.Folder{Files: active, Writer: active}
			}
			other := mills.Store().With(id)
			return link.Folder{Files: other, Writer: other}
		},
		Texts: textsOf(texts), Queue: hosttest.NewQueue(), Clock: newFakeClock(time.Unix(1_700_000_000, 0)), Entropy: random.Entropy{},
	}, func() ports.Fetcher { return fetched }, func() fetch.Settings { return fetch.Settings{} })
	t.Cleanup(links.Close)
	made, err := links.Create(t.Context(), "", "", videoAddress)
	if err != nil {
		t.Fatal(err)
	}
	found, err := links.Read(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, isStarted, err := links.StartMediaCopy(t.Context(), found, fetch.MediaVideo); err != nil || !isStarted {
		t.Fatalf("StartMediaCopy = %v, %v", isStarted, err)
	}
	<-fetched.started

	current.Store(harbor)
	harbor.File(made.Path, "---\ntype: link\nurl: "+videoAddress+"\n---\n")
	close(fetched.gate)
	links.Wait()

	if files, err := harbor.Store().List(t.Context()); err != nil || len(files) != 0 {
		t.Errorf("the vault current after the switch holds %+v in the store (%v)", files, err)
	}
	if content, _ := harbor.Content(made.Path); bytes.Contains([]byte(content), []byte("media:")) {
		t.Errorf("the note of the vault current after the switch reads %q", content)
	}
	if _, err := mills.Store().Read(t.Context(), found.Key+".video.mp4"); err != nil {
		t.Errorf("the vault the download started in holds no copy: %v", err)
	}
	content, _ := mills.Content(made.Path)
	if !bytes.Contains([]byte(content), []byte("media: plugin://"+pluginID+"/"+found.Key+".video.mp4")) {
		t.Errorf("the note of the vault the download started in reads %q", content)
	}
}
