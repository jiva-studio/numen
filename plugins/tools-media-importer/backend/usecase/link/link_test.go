package link_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/ytdlp"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

const pluginID = "md.numen.tools.media.importer"

const (
	articleAddress = "https://tidal-mills.example/history/rance"
	videoAddress   = "https://videos.example/watch?v=tidalmill01"
	silentAddress  = "https://quiet.example/"
)

// sites publishes an article at one address, a captioned video at another, and nothing at a third.
// With started and gate set, a download signals started once it begins and waits for gate to close.
type sites struct {
	texts       atomic.Int32
	failAt      int
	beforeGate  int
	clip        []byte
	media       []download.Media
	started     chan struct{}
	startedOnce sync.Once
	gate        chan struct{}
}

func (s *sites) CanCopy(address string) bool { return address == videoAddress }

func (s *sites) GetDownloadModel(string) download.Model {
	return download.Model{Tool: "sites", Version: "1"}
}

func (s *sites) GetMetadata(context.Context, string) (download.Metadata, error) {
	return download.Metadata{Bytes: int64(len(s.clip))}, nil
}

func (s *sites) GetText(_ context.Context, address string, _ download.PreferredCaptions) (download.Text, error) {
	s.texts.Add(1)
	switch address {
	case articleAddress:
		return download.Text{Producer: "article", Title: "Tidal Mills", Prose: "A tidal mill stored seawater behind a sluice gate."}, nil
	case videoAddress:
		return download.Text{Producer: "captions", Cues: []ytdlp.Cue{
			{Text: "The tide comes in.", From: 0, To: 1500},
			{Text: "The wheel turns.", From: 1500, To: 3000},
		}}, nil
	}
	return download.Text{}, download.ErrNothingDownloaded
}

func (s *sites) Download(_ context.Context, _ string, media download.Media, into io.Writer) (download.Copy, error) {
	s.media = append(s.media, media)
	rest := s.clip
	if s.beforeGate > 0 {
		if _, err := into.Write(s.clip[:s.beforeGate]); err != nil {
			return download.Copy{}, err
		}
		rest = s.clip[s.beforeGate:]
	}
	if s.gate != nil {
		s.startedOnce.Do(func() { close(s.started) })
		<-s.gate
	}
	if s.failAt > 0 {
		if _, err := into.Write(s.clip[:s.failAt]); err != nil {
			return download.Copy{}, err
		}
		return download.Copy{}, errors.New("the site hung up")
	}
	_, err := into.Write(rest)
	if media == download.MediaAudio {
		return download.Copy{MediaType: "audio/mp4", Extension: ".m4a"}, err
	}
	return download.Copy{MediaType: "video/mp4", Extension: ".mp4"}, err
}

type world struct {
	vault *hosttest.Vault
	store *hosttest.Store
	texts *hosttest.Texts
	queue *hosttest.Queue
	sites *sites
	links *link.Links
	limit int64
}

func newWorld(t *testing.T) *world {
	t.Helper()
	vault := hosttest.New("Mills")
	w := &world{
		vault: vault, store: vault.Store(), texts: hosttest.NewTexts(vault.Vault().ID, pluginID),
		queue: hosttest.NewQueue(), sites: &sites{clip: bytes.Repeat([]byte("frame "), 1000)},
	}
	w.links = link.New(link.Host{PluginID: pluginID, Vaults: vaultsOf(vault), Store: foldersOf(vault), Texts: textsOf(w.texts), Queue: w.queue, Clock: newFakeClock(time.Unix(1_700_000_000, 0))},
		func() link.Fetcher { return w.sites },
		func() link.Settings { return link.Settings{CopyMaxSize: w.limit} })
	w.texts.SetStore(pluginID, w.store)
	t.Cleanup(w.links.Close)
	return w
}

func (w *world) create(t *testing.T, address string) link.Link {
	t.Helper()
	return w.createIn(t, "links", address)
}

func (w *world) createIn(t *testing.T, folder, address string) link.Link {
	t.Helper()
	made, err := w.links.Create(t.Context(), "", folder, address)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.vault.Scan(t.Context(), []string{made.Path}); err != nil {
		t.Fatal(err)
	}
	content, _ := w.vault.Content(made.Path)
	w.texts.File(made.Path, []byte(content))
	return made
}

func (w *world) readProperty(t *testing.T, path, key string) string {
	t.Helper()
	content, _ := w.vault.Content(path)
	note := markdown.Parse(domain.FileStat{Path: path, Kind: "", Size: 0, ModTime: time.Time{}}, []byte(content))
	value, isText := note.Frontmatter[key].(string)
	if !isText {
		return ""
	}
	return value
}

func (w *world) readLiveText(t *testing.T, path string) string {
	t.Helper()
	listing, err := w.texts.ListVersions(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		whole, err := host.ReadWholeText(t.Context(), w.texts, version)
		if err != nil {
			t.Fatal(err)
		}
		return whole.Text
	}
	return ""
}

func TestCreateMakesANoteOfTypeLinkWithTheAddress(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	if !strings.HasPrefix(made.Path, "links/") || !strings.HasSuffix(made.Path, ".md") {
		t.Errorf("the link is filed at %s", made.Path)
	}
	content, _ := w.vault.Content(made.Path)
	if !strings.Contains(content, "type: link\n") || !strings.Contains(content, "url: "+articleAddress+"\n") {
		t.Errorf("the note holds %q", content)
	}
	read, err := w.links.Read(t.Context(), made.Path)
	if err != nil || read != made {
		t.Errorf("the link reads back as %+v (%v), want %+v", read, err, made)
	}
	if _, err := w.links.Create(t.Context(), "", "links", articleAddress); !errors.Is(err, host.ErrAlreadyExists) {
		t.Errorf("a second link of one address was answered %v", err)
	}
	if _, err := w.links.Create(t.Context(), "", "", "file:///etc/passwd"); !errors.Is(err, address.ErrNotAURL) {
		t.Errorf("a file address was answered %v", err)
	}
	w.vault.File("notes/plain.md", "---\ntype: deck\nurl: https://a.example/\n---\n")
	if _, err := w.links.Read(t.Context(), "notes/plain.md"); !errors.Is(err, link.ErrNotALink) {
		t.Errorf("a note of another type was read as a link: %v", err)
	}
}

func TestAnArticleIsThePlainTextVersionOfItsLink(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	listing, err := w.texts.ListVersions(t.Context(), made.Path)
	if err != nil || len(listing.Versions) != 1 || listing.Live == "" {
		t.Fatalf("the versions are %+v (%v)", listing, err)
	}
	if kind := listing.Versions[0].Kind; kind != "article" {
		t.Errorf("an article is %q text", kind)
	}
	if text := w.readLiveText(t, made.Path); text != "A tidal mill stored seawater behind a sluice gate." {
		t.Errorf("the article reads %q", text)
	}
	if record, err := w.store.Read(t.Context(), made.Key+".json"); err != nil || !strings.Contains(string(record), `"Tidal Mills"`) {
		t.Errorf("the record reads %s (%v), want the title Tidal Mills", record, err)
	}
	if tasks := w.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks are left shown: %+v", tasks)
	}
}

func TestCaptionsAreTheTimedTextVersionOfTheirLink(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	listing, err := w.texts.ListVersions(t.Context(), made.Path)
	if err != nil || len(listing.Versions) != 1 || listing.Versions[0].Kind != "captions" {
		t.Fatalf("the versions are %+v (%v)", listing, err)
	}
	page, err := w.texts.ReadText(t.Context(), made.Path, listing.Live, "")
	if err != nil || len(page.Segments) != 2 {
		t.Fatalf("the captions read as %+v (%v)", page, err)
	}
	if second := page.Segments[1]; second.Text != "The wheel turns.\n" || second.Selector != `{"t":[1.5,3]}` {
		t.Errorf("the second caption is %+v", second)
	}
}

func TestFetchText_AsksAnEmptyAddressOnEachRun(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, silentAddress)

	for range 2 {
		if err := w.links.FetchText(t.Context(), made); err != nil {
			t.Fatal(err)
		}
	}
	if count := w.sites.texts.Load(); count != 2 {
		t.Errorf("the address was asked %d times, want twice", count)
	}
	listing, err := w.texts.ListVersions(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(listing.Versions) != 0 {
		t.Errorf("an address that publishes nothing has versions %+v", listing.Versions)
	}
}

func TestCopyWritesTheVideoToTheKeyFile(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	size, err := w.links.Copy(t.Context(), made, download.MediaVideo)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(w.sites.clip)) {
		t.Errorf("the copy came to %d bytes, want %d", size, len(w.sites.clip))
	}
	copied, err := w.store.Read(t.Context(), made.Key+".video.mp4")
	if err != nil || !bytes.Equal(copied, w.sites.clip) {
		t.Errorf("the folder keeps %d bytes (%v)", len(copied), err)
	}
	if _, err := w.store.Read(t.Context(), made.Key+".part"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the copy arriving is left: %v", err)
	}
	if again, err := w.links.Copy(t.Context(), made, download.MediaVideo); err != nil || again != size || len(w.sites.media) != 1 {
		t.Errorf("a second copy came to %d bytes (%v) and asked the site %v", again, err, w.sites.media)
	}
	if err := w.links.DeleteCopy(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	if _, err := w.store.Read(t.Context(), made.Key+".video.mp4"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a deleted copy is still kept: %v", err)
	}
	if got := w.readProperty(t, made.Path, "media"); got != "" {
		t.Errorf("the note of a deleted copy holds media %q", got)
	}
}

func TestCopyDeletesThePartialFileOnFailure(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	w.sites.failAt = 100
	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err == nil {
		t.Fatal("a copy the site hung up on was kept")
	}
	w.sites.failAt = 0
	w.limit = 100
	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); !errors.Is(err, link.ErrTooLarge) {
		t.Errorf("a copy past the limit was answered %v", err)
	}
	files, err := w.store.List(t.Context())
	if err != nil || len(files) != 0 {
		t.Errorf("the folder keeps %+v (%v)", files, err)
	}
	tasks := w.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].FailureReason == "" || tasks[0].Detail != made.Path {
		t.Errorf("the failure is shown as %+v", tasks)
	}
	if _, err := w.links.Copy(t.Context(), w.create(t, articleAddress), download.MediaVideo); !errors.Is(err, link.ErrNotCopyable) {
		t.Errorf("an article was copied: %v", err)
	}
}

func TestTheTextIsStoredInTheCommonFormatAlone(t *testing.T) {
	w := newWorld(t)
	for _, made := range []link.Link{w.create(t, articleAddress), w.create(t, videoAddress)} {
		if err := w.links.FetchText(t.Context(), made); err != nil {
			t.Fatal(err)
		}
	}
	files, err := w.store.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range files {
		if !strings.HasSuffix(one.Name, ".jsonl") && !strings.HasSuffix(one.Name, ".json") {
			t.Errorf("the folder keeps %s", one.Name)
		}
	}
}

func (w *world) deleteVersion(t *testing.T, made link.Link) {
	t.Helper()
	if err := w.store.Delete(t.Context(), made.Key+".jsonl"); err != nil {
		t.Fatal(err)
	}
	if err := w.texts.RemoveVersion(t.Context(), made.Path, made.Key); err != nil {
		t.Fatal(err)
	}
}

func TestFetchTextFetchesADeletedTextAgain(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	w.deleteVersion(t, made)
	if text := w.readLiveText(t, made.Path); text != "" {
		t.Errorf("the deleted article reads %q", text)
	}

	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	if count := w.sites.texts.Load(); count != 2 {
		t.Errorf("the address was asked %d times, want twice", count)
	}
	if text := w.readLiveText(t, made.Path); text == "" {
		t.Error("a second run leaves no article")
	}
}

func TestFetchTextStoresTheArticleOfANewLink(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	if text := w.readLiveText(t, made.Path); text == "" {
		t.Error("the new link has no article")
	}
}

func (w *world) edit(t *testing.T, made link.Link) link.Link {
	t.Helper()
	content, _ := w.vault.Content(made.Path)
	content += "Seen on a visit to the estuary.\n"
	w.vault.File(made.Path, content)
	w.texts.File(made.Path, []byte(content))
	edited, err := w.links.Read(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	return edited
}

func TestTheCopyFollowsAnEditOfTheNote(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	size, err := w.links.Copy(t.Context(), made, download.MediaVideo)
	if err != nil {
		t.Fatal(err)
	}
	edited := w.edit(t, made)

	if again, err := w.links.Copy(t.Context(), edited, download.MediaVideo); err != nil || again != size || len(w.sites.media) != 1 {
		t.Errorf("the copy of the edited note came to %d bytes (%v) and asked the site %v, want %d", again, err, w.sites.media, size)
	}
}

func TestCopyWritesTheAddressOfTheMediaIntoTheNote(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err != nil {
		t.Fatal(err)
	}

	if want, got := "plugin://"+pluginID+"/"+made.Key+".video.mp4", w.readProperty(t, made.Path, "media"); got != want {
		t.Errorf("the note holds media %q, want %q", got, want)
	}
	read, err := w.links.Read(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.links.DeleteCopy(t.Context(), read); err != nil {
		t.Fatal(err)
	}
	if got := w.readProperty(t, made.Path, "media"); got != "" {
		t.Errorf("after the deletion the note holds media %q", got)
	}
}

func TestTheTextIsLiveAfterTheMediaAddressIsSet(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}

	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err != nil {
		t.Fatal(err)
	}

	content, _ := w.vault.Content(made.Path)
	w.texts.File(made.Path, []byte(content))
	if text := w.readLiveText(t, made.Path); text == "" {
		t.Error("the note with its media has no live text")
	}
}

func TestTheTextIsMadeFromTheURLAsTheNoteWritesIt(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}

	listing, err := w.texts.ListVersions(t.Context(), made.Path)
	if err != nil || len(listing.Versions) != 1 {
		t.Fatalf("the versions are %+v (%v)", listing, err)
	}
	version := listing.Versions[0]
	if want := (domain.TextSourceProperty{Key: "url", Value: articleAddress}); version.SourceProperty != want || version.InputHash != "" {
		t.Errorf("the version is made from %+v with input hash %q, want %+v and none", version.SourceProperty, version.InputHash, want)
	}
}

func TestTheTextStaysLiveAfterAnEditOfTheNote(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}

	edited := w.edit(t, made)

	listing, err := w.texts.ListVersions(t.Context(), edited.Path)
	if err != nil || listing.Live != domain.TextVersionID(pluginID+"/"+made.Key) {
		t.Errorf("after an edit the live version is %q (%v)", listing.Live, err)
	}
}

func TestTheTextIsNotLiveOnceTheURLChanges(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}

	content, _ := w.vault.Content(made.Path)
	content = strings.Replace(content, "url: "+articleAddress, "url: https://example.org/another-mill", 1)
	w.vault.File(made.Path, content)
	w.texts.File(made.Path, []byte(content))

	if listing, err := w.texts.ListVersions(t.Context(), made.Path); err != nil || listing.Live != "" {
		t.Errorf("after the url changed the live version is %q (%v), want none", listing.Live, err)
	}
}

func TestCopyRecordsNoVersion(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	w.texts.TakeChanges()

	read, err := w.links.Read(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.links.Copy(t.Context(), read, download.MediaVideo); err != nil {
		t.Fatal(err)
	}
	if changes := w.texts.TakeChanges(); len(changes) != 0 {
		t.Errorf("writing the media address put versions of %v", changes)
	}
}

func TestCreateWritesThePageThatFramesThePlayer(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, "https://www.youtube.com/watch?v=tidalmill01")

	if want, got := "plugin://"+pluginID+"/"+made.Key+".html", w.readProperty(t, made.Path, "page"); got != want {
		t.Fatalf("the note holds page %q, want %q", got, want)
	}
	page, err := w.store.Read(t.Context(), made.Key+".html")
	if err != nil || !strings.Contains(string(page), "https://www.youtube-nocookie.com/embed/tidalmill01") {
		t.Errorf("the page reads %q (%v)", page, err)
	}
	if article := w.create(t, articleAddress); w.readProperty(t, article.Path, "page") != "" {
		t.Errorf("an article holds page %q", w.readProperty(t, article.Path, "page"))
	}
}

func TestFetchTextAfterACopyFetchesTheText(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if _, err := w.links.Copy(t.Context(), made, download.MediaVideo); err != nil {
		t.Fatal(err)
	}

	if err := w.links.FetchText(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	if count := w.sites.texts.Load(); count != 1 {
		t.Errorf("the address was asked %d times for text, want once", count)
	}
}

func TestALinkOfTheShownVaultCarriesItsIdentity(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	read, err := w.links.Read(t.Context(), made.Path)
	if err != nil {
		t.Fatal(err)
	}
	if vaultID := w.vault.Vault().ID; made.Vault != vaultID || read.Vault != vaultID {
		t.Errorf("the link is made in %q and read in %q, want %q both times", made.Vault, read.Vault, vaultID)
	}
}

// The copy is kept by the address, not by the note: a note of the address made again after the first
// was deleted points at the copy kept from before.
func TestCopyOfAStoredVideoSetsTheMediaOfANewNote(t *testing.T) {
	w := newWorld(t)
	first := w.create(t, videoAddress)
	if _, err := w.links.Copy(t.Context(), first, download.MediaVideo); err != nil {
		t.Fatal(err)
	}
	w.vault.Remove(first.Path)
	again := w.create(t, videoAddress)

	if _, err := w.links.Copy(t.Context(), again, download.MediaVideo); err != nil {
		t.Fatal(err)
	}

	if want, got := "plugin://"+pluginID+"/"+again.Key+".video.mp4", w.readProperty(t, again.Path, "media"); got != want {
		t.Errorf("the note made again holds media %q, want %q", got, want)
	}
}

func TestCopyInTwoVaultsOnTheSamePathReportsTwoTasks(t *testing.T) {
	w := newWorld(t)
	chemistry := hosttest.New("Chemistry")
	w.vault.Beside(chemistry)
	chemTexts := w.texts.With("Chemistry")
	chemTexts.SetStore(pluginID, chemistry.Store())

	const chunk = 1 << 20
	w.sites.clip = bytes.Repeat([]byte("a"), 2*chunk)
	w.sites.beforeGate = chunk
	w.sites.started = make(chan struct{})
	w.sites.gate = make(chan struct{})

	inMills := w.create(t, videoAddress)
	inChemistry, err := w.links.Create(t.Context(), "Chemistry", "links", videoAddress)
	if err != nil {
		t.Fatal(err)
	}
	if inMills.Path != inChemistry.Path {
		t.Fatalf("paths %q and %q, want the same path", inMills.Path, inChemistry.Path)
	}

	if _, isStarted, err := w.links.StartMediaCopy(t.Context(), inMills, download.MediaVideo); err != nil || !isStarted {
		t.Fatalf("StartMediaCopy in Mills: started=%v, err=%v", isStarted, err)
	}
	<-w.sites.started

	if _, _, err := w.links.StartMediaCopy(t.Context(), inMills, download.MediaVideo); !errors.Is(err, link.ErrBeingFetched) {
		t.Fatalf("second StartMediaCopy in Mills returned %v, want %v", err, link.ErrBeingFetched)
	}
	if _, isStarted, err := w.links.StartMediaCopy(t.Context(), inChemistry, download.MediaVideo); err != nil || !isStarted {
		t.Fatalf("StartMediaCopy in Chemistry: started=%v, err=%v", isStarted, err)
	}

	millsID := link.GetCopyTaskID("Mills", inMills.Path)
	chemID := link.GetCopyTaskID("Chemistry", inChemistry.Path)
	byID := map[string]host.Task{}
	for _, task := range w.queue.ListTasks() {
		byID[task.ID] = task
	}
	mTask, okM := byID[millsID]
	cTask, okC := byID[chemID]
	if !okM || !okC {
		t.Fatalf("tasks %+v, want both %s and %s", w.queue.ListTasks(), millsID, chemID)
	}
	if mTask.CompletedCount != chunk || mTask.TotalCount != 2*chunk || cTask.CompletedCount != 0 || cTask.TotalCount != 0 {
		t.Errorf("tasks %+v and %+v, want %d/%d in Mills and 0/0 in Chemistry", mTask, cTask, chunk, 2*chunk)
	}

	close(w.sites.gate)
	w.links.Wait()
	if tasks := w.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks %+v after both copies ended", tasks)
	}
}

// newFakeClock is a clock that reports start until the test moves it.
func newFakeClock(start time.Time) *clock.Fake {
	source := new(clock.Fake)
	source.Set(start)
	return source
}
