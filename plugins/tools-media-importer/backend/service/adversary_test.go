package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/clock"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/download"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/link"
)

// captionlessSites is the invented sites where the captions of the video cannot be fetched, while
// the video itself can be copied.
type captionlessSites struct{ inventedSites }

func (s captionlessSites) GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error) {
	if s.isVideo(address) {
		return download.Text{}, errors.New("the captions endpoint returned 429 Too Many Requests")
	}
	return s.inventedSites.GetText(ctx, address, want)
}

// silencedSites publishes the article once, and nothing at the same address after that.
type silencedSites struct {
	inventedSites
	calls *atomic.Int32
}

func (s silencedSites) GetText(ctx context.Context, address string, want download.PreferredCaptions) (download.Text, error) {
	if s.calls.Add(1) > 1 {
		return download.Text{}, download.ErrNothingDownloaded
	}
	return s.inventedSites.GetText(ctx, address, want)
}

func newSceneOver(t *testing.T, fetcher link.Fetcher) *importScene {
	t.Helper()
	vault := hosttest.New("Mills")
	store := vault.Store()
	texts := hosttest.NewTexts(vault.Vault().ID, PluginID)
	texts.SetStore(PluginID, store)
	hosted := link.Host{
		Vault: notedVault{Vault: vault, texts: texts, all: map[domain.VaultID]*hosttest.Texts{"Mills": texts}},
		Store: store, Texts: texts, Queue: hosttest.NewQueue(),
		Clock: clock.NewFake(time.Unix(1_700_000_000, 0)),
	}
	links := link.New(hosted,
		func() link.Fetcher { return fetcher },
		func() link.Settings { return link.Settings{IsAutomatic: true} })
	t.Cleanup(links.Close)
	registry, err := NewTools(links).Registry()
	if err != nil {
		t.Fatal(err)
	}
	return &importScene{vault: vault, store: store, texts: texts, links: links, registry: registry}
}

// The spec: "With copy true: where Links.IsCopyable the copy starts in the background". A failed
// fetch of the text is reported in error; it is no reason to drop the copy the call asked for.
func TestURLImportCopiesAVideoWhoseCaptionsFailed(t *testing.T) {
	release := make(chan struct{})
	close(release)
	s := newSceneOver(t, captionlessSites{inventedSites{video: download.NewMock(), release: release}})

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
func TestURLImportOfAnAddressThatNowPublishesNothingReportsNoTitle(t *testing.T) {
	release := make(chan struct{})
	close(release)
	s := newSceneOver(t, silencedSites{inventedSites{video: download.NewMock(), release: release}, &atomic.Int32{}})

	first := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "first"})
	if first.Title != articleTitle || first.Bytes == 0 {
		t.Fatalf("the first import returned %+v, want the article", first)
	}
	second := s.importURL(t, "Mills", importArguments{URL: articleAddress, Folder: "second"})

	if !second.HasText && (second.Title != "" || second.Producer != "") {
		t.Errorf("the second import returned %+v: nothing published, with the title and producer of an earlier fetch", second)
	}
}
