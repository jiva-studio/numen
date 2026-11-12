package link

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/download"
)

// text version of the link. An address that publishes no text stores nothing.
func (l *Links) FetchText(ctx context.Context, link Link) error {
	task, err := l.beginTextFetch(ctx, link)
	if err != nil {
		return err
	}
	return l.endTextFetch(ctx, link, task)
}

// StartTextFetch shows the task of the text of a link and fetches the text in the background. It
// returns ErrBeingFetched, and starts nothing, when another run fetches the text. A failure of the
// fetch is current as the error of the task.
func (l *Links) StartTextFetch(ctx context.Context, link Link) error {
	task, err := l.beginTextFetch(ctx, link)
	if err != nil {
		return err
	}
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		l.reportBackgroundError(runCtx, "fetch the text of", link, l.endTextFetch(runCtx, link, task))
	})
	return nil
}

func (l *Links) beginTextFetch(ctx context.Context, link Link) (host.Task, error) {
	if !l.claim("text", link, link.Key) {
		return host.Task{}, ErrBeingFetched
	}
	task := host.Task{
		ID: getTaskID("text", link.Vault, link.Path), Title: fetchingText, Detail: link.Path,
		CompletedCount: 0, TotalCount: 0, Unit: host.UnitThings, FailureReason: "", IsUserStarted: true,
	}
	l.reportTask(ctx, task)
	return task, nil
}

func (l *Links) endTextFetch(ctx context.Context, link Link, task host.Task) error {
	defer l.release("text", link, link.Key)
	err := l.fetchText(ctx, folder{store: l.getStore(link)}, link)
	l.endTask(ctx, task, err)
	return err
}

func (l *Links) fetchText(ctx context.Context, folder folder, link Link) error {
	fetcher, settings := l.fetcher(), l.settings()
	fetched, err := fetcher.GetText(ctx, string(link.Address), download.PreferredCaptions{
		Languages: settings.Captions, IsAutomatic: settings.IsAutomatic,
	})
	if errors.Is(err, download.ErrNothingDownloaded) {
		return nil
	}
	if err != nil {
		return err
	}
	model := fetcher.GetDownloadModel(string(link.Address))
	rec, err := folder.readRecord(ctx, link.Key)
	if err != nil {
		return err
	}
	rec.Address, rec.Title, rec.Producer = string(link.Address), fetched.Title, fetched.Producer
	rec.Downloader = model.Tool + " " + model.Version
	rec.Kind = getKind(fetched)
	segments, err := getSegments(fetched)
	// gomutants:disable-next-line BRANCH_IF reason="getSegments cannot fail for integer cue timestamps"
	if err != nil {
		return err
	}
	if err := folder.writeText(ctx, link.Key, segments, rec); err != nil {
		return err
	}
	return l.putVersion(ctx, link, rec.Kind)
}

func (l *Links) putVersion(ctx context.Context, link Link, kind domain.TextKind) error {
	_, err := l.getTexts(link).PutVersion(ctx, host.TextPut{
		Path: link.Path, Key: link.Key, SourceProperty: domain.TextSourceProperty{Key: urlKey, Value: link.URL},
		Kind: kind, File: link.Key + textExt,
	})
	return err
}

// copyChunk is how much of a copy is sent to the store in one call.

type TextInfo struct {
	Title    string
	Producer string
}

// GetTextInfo returns the title and the producer of the text fetched for the address of a link. A
// link with no text fetched has an empty title and producer.
func (l *Links) GetTextInfo(ctx context.Context, link Link) (TextInfo, error) {
	rec, err := folder{store: l.getStore(link)}.readRecord(ctx, link.Key)
	if err != nil {
		return TextInfo{}, err
	}
	return TextInfo{Title: rec.Title, Producer: rec.Producer}, nil
}

// ReadArticle is the text of the live version of a link, read in the link's vault, and nothing
// where it has none.
func (l *Links) ReadArticle(ctx context.Context, link Link) (string, error) {
	texts := l.getTexts(link)
	listing, err := texts.ListVersions(ctx, link.Path)
	if err != nil || listing.Live == "" {
		return "", err
	}
	for _, version := range listing.Versions {
		if version.GetID() != listing.Live {
			continue
		}
		whole, err := host.ReadWholeText(ctx, texts, version)
		if err != nil {
			return "", err
		}
		return whole.Text, nil
	}
	return "", nil
}
