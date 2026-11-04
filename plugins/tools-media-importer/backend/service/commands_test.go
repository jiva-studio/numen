package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// videoAddress is an address the mock downloader takes for a video site.
const videoAddress = "https://www.youtube.com/watch?v=tidalmill01"

// newMockLinks is the links of an empty vault, reaching video sites through the mock downloader.
func newMockLinks(t *testing.T) (*link.Links, *hosttest.Store, link.Host) {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	hosted := link.Host{Vault: vault, Store: store, Texts: texts, Queue: hosttest.NewQueue(), Clock: clock.NewFake(time.Unix(1_700_000_000, 0))}
	links := link.New(hosted,
		func() link.Fetcher { return download.NewMock() },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	return links, store, hosted
}

// newMockCommands is the plugin's commands over an empty vault, reaching video sites through the
// mock downloader.
func newMockCommands(t *testing.T) (*command.Registry, *hosttest.Store) {
	t.Helper()
	links, store, hosted := newMockLinks(t)
	commands, err := newCommands(links, hosted)
	if err != nil {
		t.Fatal(err)
	}
	return commands, store
}

func TestImportWithWaitKeepsTheCopyBeforeItAnswers(t *testing.T) {
	commands, store := newMockCommands(t)

	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": videoAddress, "wait": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantMsg := made.OpenedPath + " is made and its text fetched, and a copy of 24044 bytes kept"
	if made.Message != wantMsg || made.OpenedPath == "" {
		t.Fatalf("import answered %+v, want %q", made, wantMsg)
	}
	if !hasCopy(t, store) {
		t.Error("the copy is not in the plugin's folder once import answered")
	}
}

func TestImportCommands_DeleteAndDownloadCopyAndText(t *testing.T) {
	commands, store := newMockCommands(t)
	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": videoAddress, "wait": "true"},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range []struct {
		id      string
		hasCopy bool
	}{{"delete-copy", false}, {"download-copy", true}, {"download-text", true}} {
		answer, err := commands.RunCommand(t.Context(), command.Request{Command: step.id, Path: made.OpenedPath})
		if err != nil || answer.Message == "" {
			t.Fatalf("%s answered %+v (%v)", step.id, answer, err)
		}
		if hasCopy(t, store) != step.hasCopy {
			t.Errorf("after %s the copy is kept: %v, want %v", step.id, !step.hasCopy, step.hasCopy)
		}
	}
}

// hasCopy reports whether the plugin's folder keeps a copy of a video.
func hasCopy(t *testing.T, store *hosttest.Store) bool {
	t.Helper()
	files, err := store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range files {
		if strings.HasSuffix(one.Name, ".wav") {
			return true
		}
	}
	return false
}

func TestDeleteTextNeedsTheVersionTheHostNames(t *testing.T) {
	commands, _ := newMockCommands(t)

	_, err := commands.RunCommand(t.Context(), command.Request{Command: "delete-text", Path: "Mills.md"})

	if !errors.Is(err, command.ErrBadArguments) {
		t.Fatalf("a deletion with no version: got %v, want ErrBadArguments", err)
	}
}

type configurableFetcher struct {
	canCopy   bool
	textErr   error
	copyErr   error
	startedCh chan struct{}
	blockCh   chan struct{}
}

func (f *configurableFetcher) CanCopy(string) bool { return f.canCopy }

func (f *configurableFetcher) GetDownloadModel(string) download.Model {
	return download.Model{Tool: "cfg", Version: "1", Producer: "article"}
}

func (f *configurableFetcher) GetMetadata(context.Context, string) (download.Metadata, error) {
	return download.Metadata{Bytes: 10}, nil
}

func (f *configurableFetcher) GetText(context.Context, string, download.PreferredCaptions) (download.Text, error) {
	if f.textErr != nil {
		return download.Text{}, f.textErr
	}
	return download.Text{Producer: "article", Title: "T", Prose: "prose"}, nil
}

func (f *configurableFetcher) Download(ctx context.Context, _ string, into io.Writer) (download.Copy, error) {
	if f.startedCh != nil {
		close(f.startedCh)
	}
	if f.blockCh != nil {
		<-f.blockCh
	}
	if f.copyErr != nil {
		return download.Copy{}, f.copyErr
	}
	if _, err := into.Write([]byte("0123456789")); err != nil {
		return download.Copy{}, err
	}
	return download.Copy{MediaType: "audio/wav", Extension: ".wav"}, nil
}

type errListStore struct {
	host.Store
	listErr error
	readErr error
}

func (s *errListStore) With(domain.VaultID) host.Store { return s }

func (s *errListStore) List(ctx context.Context) ([]host.StoreFile, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.Store.List(ctx)
}

func (s *errListStore) Read(ctx context.Context, name string) ([]byte, error) {
	if s.readErr != nil {
		return nil, s.readErr
	}
	return s.Store.Read(ctx, name)
}

func TestDescribeSizeBoundariesAndCommandErrorPaths(t *testing.T) {
	if got := describeSize((1 << 20) - 1); got != "1048575 bytes" {
		t.Errorf("describeSize(1MB-1) = %q, want %q", got, "1048575 bytes")
	}
	if got := describeSize(1 << 20); got != "1 MB" {
		t.Errorf("describeSize(1MB) = %q, want %q", got, "1 MB")
	}

	vault := hosttest.New("Mills")
	es := &errListStore{Store: vault.Store()}
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, vault.Store())
	hosted := link.Host{Vault: vault, Store: es, Texts: texts, Queue: hosttest.NewQueue(), Clock: clock.NewFake(time.Unix(1_700_000_000, 0))}
	cf := &configurableFetcher{}
	links := link.New(hosted,
		func() link.Fetcher { return cf },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	commands, err := newCommands(links, hosted)
	if err != nil {
		t.Fatal(err)
	}

	// import with invalid address.
	if _, err := commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": "not-a-url"},
	}); err == nil {
		t.Error("import(not-a-url) succeeded")
	}

	// import when FetchText fails.
	cf.textErr = errors.New("fetch failed")
	res, err := commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": "https://a1.example/fail-text"},
	})
	if err == nil || res.OpenedPath == "" {
		t.Errorf("import(FetchText err) = (%+v, %v)", res, err)
	}
	cf.textErr = nil

	// import when not copyable.
	cf.canCopy = false
	res, err = commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": "https://a2.example/article-1"},
	})
	if err != nil || res.Message != res.OpenedPath+" is made and its text fetched" {
		t.Errorf("import(article) = (%+v, %v)", res, err)
	}

	// import when copyable without wait.
	cf.canCopy = true
	res, err = commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": "https://a3.example/video-bg"},
	})
	if err != nil || res.Message != res.OpenedPath+" is made and its text fetched; the video is being copied" {
		t.Errorf("import(bg copy) = (%+v, %v)", res, err)
	}
	links.Wait()

	// import when copyable with wait="true" and Copy fails.
	cf.copyErr = errors.New("copy failed")
	res, err = commands.RunCommand(t.Context(), command.Request{
		Command: "import", Arguments: command.Arguments{"address": "https://a4.example/video-fail", "wait": "true"},
	})
	if err == nil || res.OpenedPath == "" {
		t.Errorf("import(copy fail) = (%+v, %v)", res, err)
	}
	cf.copyErr = nil

	// download-text, download-copy, delete-copy on missing note.
	for _, cmd := range []string{"download-text", "download-copy", "delete-copy"} {
		if _, err := commands.RunCommand(t.Context(), command.Request{Command: cmd, Path: "missing.md"}); err == nil {
			t.Errorf("%s(missing.md) succeeded", cmd)
		}
	}

	// download-text when FetchText fails.
	cf.textErr = errors.New("fetch text failed")
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "download-text", Path: res.OpenedPath}); err == nil {
		t.Error("download-text(err) succeeded")
	}
	cf.textErr = nil

	// download-copy when Copy fails with non-ErrBeingFetched error.
	cf.canCopy = false
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "download-copy", Path: res.OpenedPath}); err == nil {
		t.Error("download-copy(not copyable) succeeded")
	}
	cf.canCopy = true

	// download-copy when Copy is already in progress (ErrBeingFetched).
	cf.startedCh = make(chan struct{})
	cf.blockCh = make(chan struct{})
	foundLink, err := links.Read(t.Context(), res.OpenedPath)
	if err != nil {
		t.Fatal(err)
	}
	links.StartCopy(t.Context(), foundLink)
	<-cf.startedCh
	ans, callErr := commands.RunCommand(t.Context(), command.Request{Command: "download-copy", Path: res.OpenedPath})
	close(cf.blockCh)
	links.Wait()
	cf.startedCh = nil
	cf.blockCh = nil
	if callErr != nil || !strings.Contains(ans.Message, "already being made") {
		t.Fatalf("download-copy(in progress) = (%+v, %v)", ans, callErr)
	}

	// delete-copy when DeleteCopy fails.
	es.listErr = errors.New("list error")
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: "delete-copy", Path: res.OpenedPath}); err == nil {
		t.Error("delete-copy(listErr) succeeded")
	}
	es.listErr = nil
}
