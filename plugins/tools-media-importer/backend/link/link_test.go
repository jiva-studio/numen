package link_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
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

// create makes the link note of an address, and gives the texts its file as the host would.
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
	if _, err := w.links.Create(t.Context(), "links", articleAddress); !errors.Is(err, host.ErrAlreadyExists) {
		t.Errorf("a second link of one address was answered %v", err)
	}
	if _, err := w.links.Create(t.Context(), "", "file:///etc/passwd"); !errors.Is(err, address.ErrNotAURL) {
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
	text, err := w.links.ReadArticle(t.Context(), made)
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
	if text, err := w.links.ReadArticle(t.Context(), made); err != nil || text == "" {
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
	copied, err := w.store.Read(t.Context(), made.Hash+".mp4")
	if err != nil || !bytes.Equal(copied, w.sites.clip) {
		t.Errorf("the folder keeps %d bytes (%v)", len(copied), err)
	}
	if _, err := w.store.Read(t.Context(), made.Hash+".part"); !errors.Is(err, host.ErrNotExist) {
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

func TestTheTextIsStoredInTheCommonFormatAlone(t *testing.T) {
	w := newWorld(t)
	for _, made := range []link.Link{w.create(t, articleAddress), w.create(t, videoAddress)} {
		if err := w.links.FetchText(t.Context(), made, true); err != nil {
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

// deleteVersion deletes the live version of a link and its file, as the host does for a person.
func (w *world) deleteVersion(t *testing.T, made link.Link) {
	t.Helper()
	if err := w.store.Delete(t.Context(), made.Key+".jsonl"); err != nil {
		t.Fatal(err)
	}
	if err := w.texts.RemoveVersion(t.Context(), made.Path, made.Key); err != nil {
		t.Fatal(err)
	}
}

func TestATextThePersonDeletedIsNotFetchedAgainUnlessAPersonAsks(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	if err := w.links.FetchText(t.Context(), made, true); err != nil {
		t.Fatal(err)
	}
	w.deleteVersion(t, made)

	if err := w.links.FetchText(t.Context(), made, false); err != nil {
		t.Fatal(err)
	}
	if count := w.sites.texts.Load(); count != 1 {
		t.Errorf("a deleted text was fetched again: the address was asked %d times", count)
	}
	if listing, _ := w.texts.ListVersions(t.Context(), made.Path); len(listing.Versions) != 0 {
		t.Errorf("a deleted text has versions %+v", listing.Versions)
	}
	if isDeleted, err := w.links.IsTextDeleted(t.Context(), made); err != nil || !isDeleted {
		t.Errorf("IsTextDeleted = %v (%v); want true", isDeleted, err)
	}

	if err := w.links.FetchText(t.Context(), made, true); err != nil {
		t.Fatal(err)
	}
	if text, err := w.links.ReadArticle(t.Context(), made); err != nil || text == "" {
		t.Errorf("a person asking again leaves the article %q (%v)", text, err)
	}
	if isDeleted, err := w.links.IsTextDeleted(t.Context(), made); err != nil || isDeleted {
		t.Errorf("IsTextDeleted after a fetch = %v (%v); want false", isDeleted, err)
	}
}

func TestAFetchNobodyAskedForFetchesALinkNeverFetched(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)

	if err := w.links.FetchText(t.Context(), made, false); err != nil {
		t.Fatal(err)
	}
	if text, err := w.links.ReadArticle(t.Context(), made); err != nil || text == "" {
		t.Errorf("the article reads %q (%v)", text, err)
	}
}

// edit writes the note of a link again with words of the person below the frontmatter.
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

func TestAnEditedNoteGetsItsStoredTextWithoutAFetch(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, articleAddress)
	if err := w.links.FetchText(t.Context(), made, true); err != nil {
		t.Fatal(err)
	}
	edited := w.edit(t, made)

	if err := w.links.FetchText(t.Context(), edited, false); err != nil {
		t.Fatal(err)
	}
	if count := w.sites.texts.Load(); count != 1 {
		t.Errorf("the address was asked %d times, want once", count)
	}
	if text, err := w.links.ReadArticle(t.Context(), edited); err != nil || text == "" {
		t.Errorf("the edited note reads %q (%v)", text, err)
	}
}

func TestTheCopyFollowsAnEditOfTheNote(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	size, err := w.links.Copy(t.Context(), made)
	if err != nil {
		t.Fatal(err)
	}
	edited := w.edit(t, made)

	if found, err := w.links.FindCopy(t.Context(), edited); err != nil || found != size {
		t.Errorf("the copy of the edited note is found as %d bytes (%v), want %d", found, err, size)
	}
	if _, err := w.store.Read(t.Context(), edited.Hash+".mp4"); err != nil {
		t.Errorf("the copy is not kept under the hash of the edited note: %v", err)
	}
}

func TestFetchTextAfterACopyFetchesTheText(t *testing.T) {
	w := newWorld(t)
	made := w.create(t, videoAddress)
	if _, err := w.links.Copy(t.Context(), made); err != nil {
		t.Fatal(err)
	}

	if err := w.links.FetchText(t.Context(), made, false); err != nil {
		t.Fatal(err)
	}
	if count := w.sites.texts.Load(); count != 1 {
		t.Errorf("the address was asked %d times for text, want once", count)
	}
}

func TestTheTextOfALinkIsRecordedInTheVaultTheRequestNames(t *testing.T) {
	w := newWorld(t)
	chemistry := hosttest.New("Chemistry")
	w.vault.Beside(chemistry)
	note := "---\ntype: link\nurl: " + articleAddress + "\n---\n"
	chemistry.File("links/rance.md", note)
	other := w.texts.With("Chemistry").(*hosttest.Texts)
	other.File("links/rance.md", []byte(note))
	other.SetStore(pluginID, chemistry.Store())

	if err := link.NewProvider(w.links).CreateText(t.Context(), host.TextCreation{Vault: "Chemistry", Path: "links/rance.md"}); err != nil {
		t.Fatal(err)
	}
	w.links.Wait()

	inChemistry, err := other.ListVersions(t.Context(), "links/rance.md")
	if err != nil || inChemistry.Live == "" {
		t.Fatalf("Chemistry lists %+v (%v), want the article live", inChemistry, err)
	}
	if inShown, _ := w.texts.ListVersions(t.Context(), "links/rance.md"); len(inShown.Versions) != 0 {
		t.Errorf("the shown vault holds %+v, want no version", inShown.Versions)
	}
	if names, _ := w.store.List(t.Context()); len(names) != 0 {
		t.Errorf("the shown vault's folder holds %v, want nothing", names)
	}
	read, err := w.links.ReadArticle(t.Context(), link.Link{Vault: "Chemistry", Path: "links/rance.md"})
	if err != nil || read == "" {
		t.Errorf("the article read in Chemistry is %q (%v), want its text", read, err)
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
