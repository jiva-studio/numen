package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/internal/adapter/webpage"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

// captionlessSites is the fake sites where the captions of the video cannot be fetched, while
// the video itself can be copied.
type captionlessSites struct{ fakeSites }

func (s captionlessSites) GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error) {
	if s.isVideo(address) {
		return download.Text{}, errors.New("the captions endpoint returned 429 Too Many Requests")
	}
	return s.fakeSites.GetText(ctx, address, want)
}

// emptiedSites returns the article on the first fetch and ErrNothingDownloaded on every later fetch.
type emptiedSites struct {
	fakeSites
	calls *atomic.Int32
}

func (s emptiedSites) GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error) {
	if s.calls.Add(1) > 1 {
		return download.Text{}, download.ErrNothingDownloaded
	}
	return s.fakeSites.GetText(ctx, address, want)
}

func newFixtureWithSites(t *testing.T, fetcher link.Fetcher) *importFixture {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	hosted := link.Host{
		Vaults: textVault{Vault: vault, texts: texts, all: map[domain.VaultID]*hosttest.Texts{"Mills": texts}}.vaults(),
		Store:  foldersOf(vault), Texts: textsOf(texts), Queue: hosttest.NewQueue(),
		Clock: newFakeClock(time.Unix(1_700_000_000, 0)),
	}
	links := link.New(hosted,
		func() link.Fetcher { return fetcher },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	registry, err := NewTools(links).Registry()
	if err != nil {
		t.Fatal(err)
	}
	return &importFixture{vault: vault, store: store, texts: texts, links: links, registry: registry}
}

// The spec: "With copy true: where Links.IsCopyable the copy starts in the background". A failed
// fetch of the text is reported in error; the requested download still starts.
func TestURLImportCopiesAVideoWhoseCaptionsFailed(t *testing.T) {
	release := make(chan struct{})
	close(release)
	s := newFixtureWithSites(t, captionlessSites{fakeSites{video: download.NewMock(webpage.New()), release: release}})

	out := s.importURL(t, "Mills", importArguments{URL: videoAddress, DownloadVideo: true})
	s.links.Wait()

	if out.Error == "" {
		t.Fatalf("url_import returned %+v, want the failure of the captions", out)
	}
	if out.DownloadTaskID == "" && out.DownloadError == "" {
		t.Errorf("url_import with copy set returned %+v: no copy task and no reason why no copy was started", out)
	}
	if !hasCopy(t, s.store) {
		t.Error("the copyable video was not copied")
	}
}

// An address fetched once with a title, then found to publish nothing, is reported as nothing with
// no title and no producer: the text the result describes does not exist.
func TestURLImportOfAddressWithNoContentReportsNoTitle(t *testing.T) {
	release := make(chan struct{})
	close(release)
	s := newFixtureWithSites(t, emptiedSites{fakeSites{video: download.NewMock(webpage.New()), release: release}, &atomic.Int32{}})

	first := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "first"})
	if first.Title != articleTitle || first.Bytes == 0 {
		t.Fatalf("the first import returned %+v, want the article", first)
	}
	second := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "second"})

	if !second.HasText && (second.Title != "" || second.Producer != "") {
		t.Errorf("the second import returned %+v: nothing published, with the title and producer of an earlier fetch", second)
	}
}
