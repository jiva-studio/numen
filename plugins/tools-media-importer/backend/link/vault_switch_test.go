package link_test

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// shownVault is the vault the host shows, which a person can switch to another while a run is on.
// With names a vault by identity and does not follow the switch.
type shownVault struct {
	*hosttest.Vault
	shown *atomic.Pointer[hosttest.Vault]
}

func (v shownVault) GetVault(ctx context.Context) (domain.Vault, error) {
	return v.shown.Load().GetVault(ctx)
}

func (v shownVault) Read(ctx context.Context, path string) ([]byte, domain.FileStat, error) {
	return v.shown.Load().Read(ctx, path)
}

func (v shownVault) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	return v.shown.Load().Write(ctx, path, content, baseline)
}

func (v shownVault) Create(ctx context.Context, path string, content []byte) error {
	return v.shown.Load().Create(ctx, path, content)
}

func (v shownVault) Scan(ctx context.Context, paths []string) error {
	return v.shown.Load().Scan(ctx, paths)
}

func (v shownVault) GetPathsOfType(ctx context.Context, noteType domain.NoteType) ([]string, error) {
	return v.shown.Load().GetPathsOfType(ctx, noteType)
}

// shownStore is the plugin's store of the vault the host shows, switched with it.
type shownStore struct {
	*hosttest.Store
	shown *atomic.Pointer[hosttest.Vault]
}

func (s shownStore) Read(ctx context.Context, name string) ([]byte, error) {
	return s.shown.Load().Store().Read(ctx, name)
}

func (s shownStore) List(ctx context.Context) ([]host.StoreFile, error) {
	return s.shown.Load().Store().List(ctx)
}

func (s shownStore) Append(ctx context.Context, name string, content []byte) error {
	return s.shown.Load().Store().Append(ctx, name, content)
}

func (s shownStore) Write(ctx context.Context, name string, content []byte) error {
	return s.shown.Load().Store().Write(ctx, name, content)
}

func (s shownStore) Delete(ctx context.Context, name string) error {
	return s.shown.Load().Store().Delete(ctx, name)
}

func (s shownStore) Rename(ctx context.Context, from, to string) error {
	return s.shown.Load().Store().Rename(ctx, from, to)
}

func TestADownloadWritesOnlyTheVaultItStartedIn(t *testing.T) {
	mills, harbour := hosttest.New("Mills"), hosttest.New("Harbour")
	mills.Beside(harbour)
	shown := &atomic.Pointer[hosttest.Vault]{}
	shown.Store(mills)
	texts := hosttest.NewTexts(mills.Vault().ID, pluginID)
	fetched := &sites{clip: bytes.Repeat([]byte("frame "), 100), started: make(chan struct{}), gate: make(chan struct{})}
	links := link.New(link.Host{
		PluginID: pluginID, Vault: shownVault{Vault: mills, shown: shown}, Store: shownStore{Store: mills.Store(), shown: shown},
		Texts: texts, Queue: hosttest.NewQueue(), Clock: clock.NewFake(time.Unix(1_700_000_000, 0)),
	}, func() link.Fetcher { return fetched }, func() link.Settings { return link.Settings{} })
	t.Cleanup(links.Close)
	made, err := links.Create(t.Context(), "", "", videoAddress)
	if err != nil {
		t.Fatal(err)
	}
	found, err := links.Read(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, isStarted, err := links.StartMediaCopy(t.Context(), found, download.MediaVideo); err != nil || !isStarted {
		t.Fatalf("StartMediaCopy = %v, %v", isStarted, err)
	}
	<-fetched.started

	shown.Store(harbour)
	harbour.File(made.Path, "---\ntype: link\nurl: "+videoAddress+"\n---\n")
	close(fetched.gate)
	links.Wait()

	if files, _ := harbour.Store().List(t.Context()); len(files) != 0 {
		t.Errorf("the vault shown after the switch holds %+v in the store", files)
	}
	if content, _ := harbour.Content(made.Path); bytes.Contains([]byte(content), []byte("media:")) {
		t.Errorf("the note of the vault shown after the switch reads %q", content)
	}
	if _, err := mills.Store().Read(t.Context(), found.Key+".video.mp4"); err != nil {
		t.Errorf("the vault the download started in holds no copy: %v", err)
	}
	content, _ := mills.Content(made.Path)
	if !bytes.Contains([]byte(content), []byte("media: plugin://"+pluginID+"/"+found.Key+".video.mp4")) {
		t.Errorf("the note of the vault the download started in reads %q", content)
	}
}
