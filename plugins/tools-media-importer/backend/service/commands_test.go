package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// videoAddress is an address the mock downloader takes for a video site.
const videoAddress = "https://www.youtube.com/watch?v=tidalmill01"

func newMockLinks(t *testing.T) (*link.Links, *hosttest.Store, link.Host) {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	hosted := link.Host{
		PluginID: PluginID,
		Vaults:   vaultsOf(vault),
		Store:    foldersOf(vault),
		Texts:    textsOf(texts),
		Queue:    hosttest.NewQueue(),
		Clock:    clock.NewFake(time.Unix(1_700_000_000, 0)),
	}
	links := link.New(hosted,
		func() link.Fetcher { return download.NewMock(webpage.New()) },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	return links, store, hosted
}

func newMockCommands(t *testing.T) (*command.Registry, *hosttest.Store) {
	t.Helper()
	commands, store, _ := newMockCommandsWithLinks(t)
	return commands, store
}

func newMockCommandsWithLinks(t *testing.T) (*command.Registry, *hosttest.Store, *link.Links) {
	t.Helper()
	links, store, _ := newMockLinks(t)
	commands, err := newCommands(links)
	if err != nil {
		t.Fatal(err)
	}
	return commands, store, links
}

func TestImportMakesTheLinkNoteAndNothingElse(t *testing.T) {
	commands, store := newMockCommands(t)

	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: commandImport, Arguments: command.Arguments{argumentAddress: videoAddress},
	})
	if err != nil {
		t.Fatal(err)
	}
	if made.OpenedPath == "" || made.Message != "Created "+made.OpenedPath {
		t.Fatalf("import answered %+v", made)
	}
	if hasCopy(t, store) {
		t.Error("the import downloaded the media")
	}
}

func TestTheHandlersDownloadAndDeleteTheMediaAndTheText(t *testing.T) {
	commands, store, links := newMockCommandsWithLinks(t)
	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: commandImport, Arguments: command.Arguments{argumentAddress: videoAddress},
	})
	if err != nil {
		t.Fatal(err)
	}

	for _, step := range []struct {
		id      string
		hasCopy bool
	}{{commandDownloadMedia, true}, {commandDeleteMedia, false}, {commandCreateText, false}} {
		answer, err := commands.RunCommand(t.Context(), command.Request{Command: step.id, Path: made.OpenedPath})
		links.Wait()
		if err != nil || answer.Message == "" {
			t.Fatalf("%s answered %+v (%v)", step.id, answer, err)
		}
		if hasCopy(t, store) != step.hasCopy {
			t.Errorf("after %s the copy is kept: %v, want %v", step.id, !step.hasCopy, step.hasCopy)
		}
	}
	for _, removed := range []string{"link.import", "link.downloadText", "link.deleteText", "link.downloadVideo", "link.deleteVideo"} {
		if _, err := commands.RunCommand(t.Context(), command.Request{Command: removed, Path: made.OpenedPath}); !errors.Is(err, command.ErrUnknownCommand) {
			t.Errorf("%s answered %v, want ErrUnknownCommand", removed, err)
		}
	}
}

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

// Delete text is the core's command: the core deletes the version, and the plugin handles none.
func TestTheImporterHandlesNoDeleteText(t *testing.T) {
	commands, _ := newMockCommands(t)

	_, err := commands.RunCommand(t.Context(), command.Request{Command: "app:version.delete", Path: "Mills.md"})

	if !errors.Is(err, command.ErrUnknownCommand) {
		t.Fatalf("app:version.delete answered %v, want ErrUnknownCommand", err)
	}
}

type configurableFetcher struct {
	canCopy       bool
	textErr       error
	copyErr       error
	startedCh     chan struct{}
	blockCh       chan struct{}
	textStartedCh chan struct{}
	textBlockCh   chan struct{}
}

func (f *configurableFetcher) CanCopy(string) bool { return f.canCopy }

func (f *configurableFetcher) GetDownloadModel(string) download.Model {
	return download.Model{Tool: "cfg", Version: "1", Producer: "article"}
}

func (f *configurableFetcher) GetMetadata(context.Context, string) (download.Metadata, error) {
	return download.Metadata{Bytes: 10}, nil
}

func (f *configurableFetcher) GetText(context.Context, string, download.PreferredCaptions) (download.Text, error) {
	if f.textStartedCh != nil {
		close(f.textStartedCh)
	}
	if f.textBlockCh != nil {
		<-f.textBlockCh
	}
	if f.textErr != nil {
		return download.Text{}, f.textErr
	}
	return download.Text{Producer: "article", Title: "T", Prose: "prose"}, nil
}

func (f *configurableFetcher) Download(ctx context.Context, _ string, _ download.Media, into io.Writer) (download.Copy, error) {
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
	*hosttest.Store
	listErr error
}

func (s *errListStore) List(ctx context.Context) ([]host.StoreFile, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.Store.List(ctx)
}

func TestCommandErrorPaths(t *testing.T) {
	vault := hosttest.New("Mills")
	es := &errListStore{Store: vault.Store()}
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, vault.Store())
	hosted := link.Host{
		PluginID: PluginID,
		Vaults:   vaultsOf(vault),
		Store:    func(domain.VaultID) link.Folder { return link.Folder{Files: es, Writer: es} },
		Texts:    textsOf(texts),
		Queue:    hosttest.NewQueue(),
		Clock:    clock.NewFake(time.Unix(1_700_000_000, 0)),
	}
	cf := &configurableFetcher{}
	links := link.New(hosted,
		func() link.Fetcher { return cf },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	commands, err := newCommands(links)
	if err != nil {
		t.Fatal(err)
	}

	// import with invalid address.
	if _, err := commands.RunCommand(t.Context(), command.Request{
		Command: commandImport, Arguments: command.Arguments{argumentAddress: "not-a-url"},
	}); err == nil {
		t.Error("import(not-a-url) succeeded")
	}
	res, err := commands.RunCommand(t.Context(), command.Request{
		Command: commandImport, Arguments: command.Arguments{argumentAddress: "https://a3.example/video-bg"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// The handlers over a missing note.
	for _, cmd := range []string{commandCreateText, commandDownloadMedia, commandDeleteMedia} {
		if _, err := commands.RunCommand(t.Context(), command.Request{Command: cmd, Path: "missing.md"}); err == nil {
			t.Errorf("%s(missing.md) succeeded", cmd)
		}
	}

	// app:link.downloadMedia when Copy fails with non-ErrBeingFetched error.
	cf.canCopy = false
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: commandDownloadMedia, Path: res.OpenedPath}); err == nil {
		t.Error("app:link.downloadMedia(not copyable) succeeded")
	}
	cf.canCopy = true

	// app:link.downloadMedia when Copy is already in progress (ErrBeingFetched).
	cf.startedCh = make(chan struct{})
	cf.blockCh = make(chan struct{})
	foundLink, err := links.Read(t.Context(), res.OpenedPath)
	if err != nil {
		t.Fatal(err)
	}
	links.StartCopy(t.Context(), foundLink, download.MediaVideo)
	<-cf.startedCh
	ans, callErr := commands.RunCommand(t.Context(), command.Request{Command: commandDownloadMedia, Path: res.OpenedPath})
	close(cf.blockCh)
	links.Wait()
	cf.startedCh = nil
	cf.blockCh = nil
	if callErr != nil || !strings.HasPrefix(ans.Message, "Already downloading the media of ") {
		t.Fatalf("app:link.downloadMedia(in progress) = (%+v, %v)", ans, callErr)
	}

	// app:link.deleteMedia when DeleteCopy fails.
	es.listErr = errors.New("list error")
	if _, err := commands.RunCommand(t.Context(), command.Request{Command: commandDeleteMedia, Path: res.OpenedPath}); err == nil {
		t.Error("app:link.deleteMedia(listErr) succeeded")
	}
	es.listErr = nil
}

// handlerDeadline is how long a handler whose work runs as a task may take to answer.
const handlerDeadline = 5 * time.Second

func newConfiguredCommands(t *testing.T, cf *configurableFetcher) (*command.Registry, *link.Links, *hosttest.Queue, string) {
	t.Helper()
	vault := hosttest.New("Mills")
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, vault.Store())
	queue := hosttest.NewQueue()
	hosted := link.Host{
		PluginID: PluginID,
		Vaults:   vaultsOf(vault),
		Store:    foldersOf(vault),
		Texts:    textsOf(texts),
		Queue:    queue,
		Clock:    clock.NewFake(time.Unix(1_700_000_000, 0)),
	}
	links := link.New(hosted,
		func() link.Fetcher { return cf },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	commands, err := newCommands(links)
	if err != nil {
		t.Fatal(err)
	}
	made, err := commands.RunCommand(t.Context(), command.Request{
		Command: commandImport, Arguments: command.Arguments{argumentAddress: "https://a3.example/video-bg"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return commands, links, queue, made.OpenedPath
}

func runWithin(t *testing.T, commands *command.Registry, req command.Request) (command.Result, error) {
	t.Helper()
	type answer struct {
		res command.Result
		err error
	}
	answers := make(chan answer, 1)
	go func() {
		res, err := commands.RunCommand(t.Context(), req)
		answers <- answer{res, err}
	}()
	select {
	case got := <-answers:
		return got.res, got.err
	case <-time.After(handlerDeadline):
		t.Fatalf("%s did not answer within %v while its work ran", req.Command, handlerDeadline)
		return command.Result{}, nil
	}
}

func findTask(queue *hosttest.Queue, id string) (host.Task, bool) {
	for _, one := range queue.ListTasks() {
		if one.ID == id {
			return one, true
		}
	}
	return host.Task{}, false
}

func TestDownloadCaptionsAnswersOnceItsTaskIsShown(t *testing.T) {
	cf := &configurableFetcher{textStartedCh: make(chan struct{}), textBlockCh: make(chan struct{})}
	commands, links, queue, path := newConfiguredCommands(t, cf)

	res, err := runWithin(t, commands, command.Request{Command: commandCreateText, Path: path})

	if err != nil || res.Message == "" {
		t.Fatalf("app:file.createText answered %+v (%v)", res, err)
	}
	if _, isShown := findTask(queue, "text:Mills:"+path); !isShown {
		t.Errorf("the fetch of the text is not shown as a task once the handler answered: %+v", queue.ListTasks())
	}
	<-cf.textStartedCh
	close(cf.textBlockCh)
	links.Wait()
	if tasks := queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are still shown after the fetch ended", tasks)
	}
}

func TestAFailedFetchOfTheCaptionsFailsTheTaskAndNotTheHandler(t *testing.T) {
	cf := &configurableFetcher{textErr: errors.New("tide-tables answered 404 Not Found")}
	commands, links, queue, path := newConfiguredCommands(t, cf)

	_, err := runWithin(t, commands, command.Request{Command: commandCreateText, Path: path})
	links.Wait()

	if err != nil {
		t.Errorf("app:file.createText answered %v, want the failure on its task alone", err)
	}
	task, isShown := findTask(queue, "text:Mills:"+path)
	if !isShown || task.FailureReason != "tide-tables answered 404 Not Found" {
		t.Errorf("the task of the fetch is %+v (shown: %v), want it failed with the reason", task, isShown)
	}
}

func TestDownloadMediaAnswersOnceItsTaskIsShown(t *testing.T) {
	cf := &configurableFetcher{canCopy: true, startedCh: make(chan struct{}), blockCh: make(chan struct{})}
	commands, links, queue, path := newConfiguredCommands(t, cf)

	res, err := runWithin(t, commands, command.Request{Command: commandDownloadMedia, Path: path})

	if err != nil || res.Message == "" {
		t.Fatalf("app:link.downloadMedia answered %+v (%v)", res, err)
	}
	if _, isShown := findTask(queue, link.GetCopyTaskID("Mills", path)); !isShown {
		t.Errorf("the copy is not shown as a task once the handler answered: %+v", queue.ListTasks())
	}
	<-cf.startedCh
	close(cf.blockCh)
	links.Wait()
	if tasks := queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are still shown after the copy ended", tasks)
	}
}

func TestAFailedCopyFailsTheTaskAndNotTheHandler(t *testing.T) {
	cf := &configurableFetcher{canCopy: true, copyErr: errors.New("the site answered 403 Forbidden")}
	commands, links, queue, path := newConfiguredCommands(t, cf)

	_, err := runWithin(t, commands, command.Request{Command: commandDownloadMedia, Path: path})
	links.Wait()

	if err != nil {
		t.Errorf("app:link.downloadMedia answered %v, want the failure on its task alone", err)
	}
	task, isShown := findTask(queue, link.GetCopyTaskID("Mills", path))
	if !isShown || !strings.Contains(task.FailureReason, "403 Forbidden") {
		t.Errorf("the task of the copy is %+v (shown: %v), want it failed with the reason", task, isShown)
	}
}

func TestDownloadMediaOfAnAddressNothingCopiesFailsBeforeAnyTask(t *testing.T) {
	commands, _, queue, path := newConfiguredCommands(t, &configurableFetcher{canCopy: false})

	_, err := runWithin(t, commands, command.Request{Command: commandDownloadMedia, Path: path})

	if !errors.Is(err, link.ErrNotCopyable) {
		t.Errorf("app:link.downloadMedia answered %v, want ErrNotCopyable", err)
	}
	if tasks := queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are shown for a copy that never started", tasks)
	}
}
