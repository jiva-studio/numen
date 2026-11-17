package link

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/address"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/domain/fetch"
)

func (l *Links) claim(work string, link Link, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	name := getWorkName(work, link, key)
	if _, isRunning := l.running[name]; isRunning {
		return false
	}
	l.running[name] = make(chan struct{})
	return true
}

func (l *Links) waitClaim(ctx context.Context, work string, link Link, key string) error {
	name := getWorkName(work, link, key)
	for {
		l.mu.Lock()
		released, isRunning := l.running[name]
		if !isRunning {
			l.running[name] = make(chan struct{})
			l.mu.Unlock()
			return nil
		}
		l.mu.Unlock()
		select {
		case <-released:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (l *Links) release(work string, link Link, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	name := getWorkName(work, link, key)
	if released, isRunning := l.running[name]; isRunning {
		close(released)
		delete(l.running, name)
	}
}

func getWorkName(work string, link Link, key string) string {
	return work + "\x00" + string(link.Vault) + "\x00" + key
}

// IsCopyable reports whether something on this machine copies what the address publishes.
func (l *Links) IsCopyable(parsed address.URL) bool { return l.fetcher().CanCopy(string(parsed)) }

// StartFetch fetches the text of a link in the background, and a copy of the video where one is
// published. A failure is current as the task of the work that failed.
func (l *Links) StartFetch(ctx context.Context, link Link) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		l.reportBackgroundError(runCtx, "fetch the text of", link, l.FetchText(runCtx, link))
		if l.IsCopyable(link.Address) {
			_, err := l.Copy(runCtx, link, fetch.MediaVideo)
			l.reportBackgroundError(runCtx, "copy the video of", link, err)
		}
	})
}

// StartCopy copies the media of a link in the background. A failure is current as its task.
func (l *Links) StartCopy(ctx context.Context, link Link, media fetch.Media) {
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		_, err := l.Copy(runCtx, link, media)
		l.reportBackgroundError(runCtx, "copy the video of", link, err)
	})
}

func (l *Links) reportBackgroundError(ctx context.Context, work string, link Link, err error) {
	if err == nil || ctx.Err() != nil || errors.Is(err, ErrBeingFetched) {
		return
	}
	l.reportError(work+" "+link.Path, err)
}

func (l *Links) reportError(what string, err error) {
	if l.host.Report != nil && err != nil {
		l.host.Report(what, err)
	}
}

// taskTitles are the names of each type of work on a link in the task list.
const (
	fetchingText = "Fetching the text of a link"
	copyingVideo = "Copying a video"
)

func getTaskID(work string, vault domain.VaultID, path string) string {
	return work + ":" + string(vault) + ":" + path
}

// GetCopyTaskID returns the id of the copy of the video of a link note of a vault in the list of tasks.
func GetCopyTaskID(vault domain.VaultID, path string) string { return getTaskID("copy", vault, path) }

// FetchText fetches the text published at a link's address and stores it as a

func (l *Links) reportTask(ctx context.Context, task host.Task) {
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr diagnostic log on queue failure has no observable effect"
	if err := l.host.Queue.ReportTask(ctx, task); err != nil {
		l.reportError("show the task "+task.ID, err)
	}
}

func (l *Links) endTask(ctx context.Context, task host.Task, err error) {
	ctx = context.WithoutCancel(ctx)
	if err == nil || errors.Is(err, context.Canceled) {
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr diagnostic log on queue failure has no observable effect"
		if finishErr := l.host.Queue.FinishTask(ctx, task.ID); finishErr != nil {
			l.reportError("finish the task "+task.ID, finishErr)
		}
		return
	}
	task.FailureReason = err.Error()
	l.reportTask(ctx, task)
}

// storeWriter appends what is written to a file of the store, a chunk at a time, and stops past a
