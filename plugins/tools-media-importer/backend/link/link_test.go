package link_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/urlfile"
)

const pluginID = "md.numen.tools.media.importer"

const (
	articleAddress = "https://tidal-mills.example/history/rance"
	videoAddress   = "https://videos.example/watch?v=tidalmill01"
	silentAddress  = "https://quiet.example/"
)

// sites publishes an article at one address, a captioned video at another, and nothing at a third.
type sites struct {
	texts  atomic.Int32
	failAt int
	clip   []byte
}

func (s *sites) CanCopy(address string) bool { return address == videoAddress }

func (s *sites) GetDownloadModel(string) download.DownloadModel {
	return download.DownloadModel{Tool: "sites", Version: "1"}
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
		return download.Text{Producer: "captions", Cues: []download.Cue{
			{Text: "The tide comes in.", From: 0, To: 1500},
			{Text: "The wheel turns.", From: 1500, To: 3000},
		}}, nil
	}
	return download.Text{}, download.ErrNothingDownloaded
}

func (s *sites) Download(_ context.Context, _ string, into io.Writer) (download.Copy, error) {
	if s.failAt > 0 {
		if _, err := into.Write(s.clip[:s.failAt]); err != nil {
			return download.Copy{}, err
		}
		return download.Copy{}, errors.New("the site hung up")
	}
	_, err := into.Write(s.clip)
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
	w.links = link.New(link.Host{Vault: vault, Store: w.store, Texts: w.texts, Queue: w.queue},
		func() link.Fetcher { return w.sites },
		func() link.Settings { return link.Settings{CopyMaxSize: w.limit} })
	w.texts.SetStore(pluginID, w.store)
	t.Cleanup(w.links.Close)
	return w
}

// create makes the link of an address, and gives the texts its file as the host would.
func (w *world) create(t *testing.T, address string) link.Link {
	t.Helper()
	made, err := w.links.Create(t.Context(), "links", address)
	if err != nil {
		t.Fatal(err)
	}
	content, _ := w.vault.Content(made.Path)
	w.texts.File(made.Path, []byte(content))
	return made
}

func TestALinkIsTheAddressInAFileNamedByIt(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	if !strings.HasPrefix(made.Path, "links/") || !strings.HasSuffix(made.Path, ".url") {
		t.Errorf("the link is filed at %s", made.Path)
	}
	content, _ := w.vault.Content(made.Path)
	if address, err := urlfile.Read([]byte(content)); err != nil || address != articleAddress {
		t.Errorf("the file holds %q (%v)", content, err)
	}
	read, err := w.links.Read(t.Context(), made.Path)
	if err != nil || read != made {
		t.Errorf("the link reads back as %+v (%v), want %+v", read, err, made)
	}
	if _, err := w.links.Create(t.Context(), "links", articleAddress); !errors.Is(err, host.ErrOccupied) {
		t.Errorf("a second link of one address was answered %v", err)
	}
	if _, err := w.links.Create(t.Context(), "", "file:///etc/passwd"); !errors.Is(err, domain.ErrNotAURL) {
		t.Errorf("a file address was answered %v", err)
	}
	if _, err := w.links.Read(t.Context(), "notes/plain.md"); !errors.Is(err, link.ErrNotALink) {
		t.Errorf("a note was read as a link: %v", err)
	}
}

func TestAnArticleIsThePlainTextVersionOfItsLink(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	if err := w.links.FetchText(t.Context(), made, true); err != nil {
		t.Fatal(err)
	}
	listing, err := w.texts.ListVersions(t.Context(), made.Path)
	if err != nil || len(listing.Versions) != 1 || listing.Live == "" {
		t.Fatalf("the versions are %+v (%v)", listing, err)
	}
	if kind := listing.Versions[0].Kind; kind != "article" {
		t.Errorf("an article is %q text", kind)
	}
	text, err := w.links.ReadArticle(t.Context(), made.Path)
	if err != nil || text != "A tidal mill stored seawater behind a sluice gate." {
		t.Errorf("the article reads %q (%v)", text, err)
	}
	if title, err := w.links.GetTitle(t.Context(), made); err != nil || title != "Tidal Mills" {
		t.Errorf("the page is called %q (%v)", title, err)
	}
	if tasks := w.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("tasks are left shown: %+v", tasks)
	}
}

func TestCaptionsAreTheTimedTextVersionOfTheirLink(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	if err := w.links.FetchText(t.Context(), made, true); err != nil {
		t.Fatal(err)
	}
	listing, err := w.texts.ListVersions(t.Context(), made.Path)
	if err != nil || len(listing.Versions) != 1 || listing.Versions[0].Kind != "transcript" {
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

func TestAnAddressThatPublishesNothingIsNotAskedAgainUnlessAPersonAsks(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, silentAddress)

	for _, isUserStarted := range []bool{true, false} {
		if err := w.links.FetchText(t.Context(), made, isUserStarted); err != nil {
			t.Fatal(err)
		}
	}
	if count := w.sites.texts.Load(); count != 1 {
		t.Errorf("the address was asked %d times, want once", count)
	}
	if err := w.links.FetchText(t.Context(), made, true); err != nil || w.sites.texts.Load() != 2 {
		t.Errorf("a person asking again asked the address %d times in all (%v)", w.sites.texts.Load(), err)
	}
	if listing, _ := w.texts.ListVersions(t.Context(), made.Path); len(listing.Versions) != 0 {
		t.Errorf("an address that publishes nothing has versions %+v", listing.Versions)
	}
}

func TestTheHostAsksForTheTextOfALinkAndItIsFetched(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	provider := link.NewProvider(w.links)
	if err := provider.CreateText(t.Context(), host.TextCreation{Vault: w.vault.Vault().ID, Path: made.Path}); err != nil {
		t.Fatal(err)
	}
	w.links.Wait()
	if text, err := w.links.ReadArticle(t.Context(), made.Path); err != nil || text == "" {
		t.Errorf("the article reads %q (%v)", text, err)
	}
	if err := provider.CreateText(t.Context(), host.TextCreation{Vault: w.vault.Vault().ID, Path: made.Path, Source: "v1"}); !errors.Is(err, host.ErrInvalidVersion) {
		t.Errorf("a request made from a version was answered %v", err)
	}
}

func TestCopyWritesTheVideoToTheKeyFile(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	size, err := w.links.Copy(t.Context(), made)
	if err != nil {
		t.Fatal(err)
	}
	if size != int64(len(w.sites.clip)) {
		t.Errorf("the copy came to %d bytes, want %d", size, len(w.sites.clip))
	}
	copied, err := w.store.Read(t.Context(), made.Key+".mp4")
	if err != nil || !bytes.Equal(copied, w.sites.clip) {
		t.Errorf("the folder keeps %d bytes (%v)", len(copied), err)
	}
	if _, err := w.store.Read(t.Context(), made.Key+".part"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the copy arriving is left: %v", err)
	}
	if found, err := w.links.FindCopy(t.Context(), made); err != nil || found != size {
		t.Errorf("the copy is found as %d bytes (%v)", found, err)
	}
	if err := w.links.DeleteCopy(t.Context(), made); err != nil {
		t.Fatal(err)
	}
	if found, err := w.links.FindCopy(t.Context(), made); err != nil || found != 0 {
		t.Errorf("a deleted copy is found as %d bytes (%v)", found, err)
	}
}

func TestCopyDeletesThePartialFileOnFailure(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)

	w.sites.failAt = 100
	if _, err := w.links.Copy(t.Context(), made); err == nil {
		t.Fatal("a copy the site hung up on was kept")
	}
	w.sites.failAt = 0
	w.limit = 100
	if _, err := w.links.Copy(t.Context(), made); !errors.Is(err, link.ErrTooLarge) {
		t.Errorf("a copy past the limit was answered %v", err)
	}
	files, err := w.store.List(t.Context())
	if err != nil || len(files) != 0 {
		t.Errorf("the folder keeps %+v (%v)", files, err)
	}
	tasks := w.queue.ListTasks()
	if len(tasks) != 1 || tasks[0].Error == "" || tasks[0].About != made.Path {
		t.Errorf("the failure is shown as %+v", tasks)
	}
	if _, err := w.links.Copy(t.Context(), w.create(t, articleAddress)); !errors.Is(err, link.ErrNotCopyable) {
		t.Errorf("an article was copied: %v", err)
	}
}
