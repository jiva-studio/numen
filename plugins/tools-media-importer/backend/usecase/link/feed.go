package link

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/changes"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// CursorFile is the file of the plugin's store folder of a vault that keeps the cursor of the file
// change feed of the vault.
const CursorFile = "links.cursor.json"

// cursorStore keeps the cursor in the plugin's folder of the store of a vault.
type cursorStore struct {
	folder Folder
}

func (s cursorStore) Read(ctx context.Context, name string) ([]byte, error) {
	return s.folder.Files.Read(ctx, name)
}

func (s cursorStore) Write(ctx context.Context, name string, content []byte) error {
	return s.folder.Writer.Write(ctx, name, content)
}

// feedRun is the background run of FetchMissingTexts. isRunning is true while one runs, isAgain asks
// it to run once more when it ends, and isReview to read every link note in that run. isConfigured is
// true once the settings were applied.
type feedRun struct {
	isRunning    bool
	isAgain      bool
	isReview     bool
	isConfigured bool
}

// Configure applies the settings through update and fetches the missing texts in the background: over
// every link note when the automatic download was turned on after the first settings, and from the
// cursor of the feed otherwise.
func (l *Links) Configure(ctx context.Context, update func()) {
	wasAutomatic := l.getSettings().IsTextDownloadAutomatic
	update()
	l.mu.Lock()
	wasConfigured := l.feed.isConfigured
	l.feed.isConfigured = true
	l.mu.Unlock()
	l.StartMissingTextFetch(ctx, wasConfigured && !wasAutomatic)
}

// StartMissingTextFetch runs FetchMissingTexts in the background; a call while it runs makes it run
// once more after. With isReview set, the run fetches the text of every link note that has none, as
// it does when the feed resets. A failure is reported.
func (l *Links) StartMissingTextFetch(ctx context.Context, isReview bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.feed.isReview = l.feed.isReview || isReview
	if l.feed.isRunning {
		l.feed.isAgain = true
		return
	}
	l.feed.isRunning = true
	l.wg.Go(func() {
		runCtx, cancel := l.withClosing(context.WithoutCancel(ctx))
		defer cancel()
		for isReview := l.takeReview(); ; isReview = l.takeReview() {
			if err := l.fetchRun(runCtx, isReview); err != nil && runCtx.Err() == nil {
				l.reportError("download the missing texts of the links", err)
			}
			if !l.takeAgain() {
				return
			}
		}
	})
}

func (l *Links) fetchRun(ctx context.Context, isReview bool) error {
	if isReview {
		return l.fetchMissingTexts(ctx, true)
	}
	return l.FetchMissingTexts(ctx)
}

func (l *Links) takeReview() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	isReview := l.feed.isReview
	l.feed.isReview = false
	return isReview
}

func (l *Links) takeAgain() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.feed.isAgain {
		l.feed.isRunning = false
		return false
	}
	l.feed.isAgain = false
	return true
}

// FetchMissingTexts fetches, while the setting to download texts automatically is on, the text of each
// link note of the vault the host shows that was added or got another address since the cursor of its
// file change feed and has no text of its address yet, one note at a time. When the feed resets, every
// link note of the vault is read. A fetch that fails is reported and passed over.
func (l *Links) FetchMissingTexts(ctx context.Context) error {
	return l.fetchMissingTexts(ctx, false)
}

func (l *Links) fetchMissingTexts(ctx context.Context, isReview bool) error {
	if !l.getSettings().IsTextDownloadAutomatic {
		return nil
	}
	shown, err := l.host.GetVault("").Files.GetVault(ctx)
	if errors.Is(err, host.ErrNoVault) {
		return nil
	}
	if err != nil {
		return err
	}
	vault := l.getVaultByID(shown.ID)
	from := changes.Source{Feed: vault.Feed, Store: cursorStore{folder: l.getStoreByID(shown.ID)}, CursorFile: CursorFile}
	if isReview {
		if err := changes.Forget(ctx, from); err != nil {
			return err
		}
	}
	events := []string{host.EventFileAdded, host.EventNoteFrontmatterChanged}
	return changes.Follow(ctx, from, events,
		func(ctx context.Context, batch []host.FileChange) error {
			paths := make([]string, 0, len(batch))
			for _, change := range batch {
				paths = append(paths, change.Path)
			}
			return l.fetchMissingTextsAt(ctx, vault.Files, paths)
		},
		func(ctx context.Context) error {
			paths, err := vault.Files.ListPathsOfType(ctx, NoteType)
			if err != nil {
				return err
			}
			return l.fetchMissingTextsAt(ctx, vault.Files, paths)
		})
}

func (l *Links) fetchMissingTextsAt(ctx context.Context, files Files, paths []string) error {
	isSeen := make(map[string]bool, len(paths))
	for _, path := range paths {
		if path == "" || isSeen[path] {
			continue
		}
		isSeen[path] = true
		if err := l.fetchMissingText(ctx, files, path); err != nil {
			return err
		}
	}
	return nil
}

func (l *Links) fetchMissingText(ctx context.Context, files Files, path string) error {
	found, err := readLink(ctx, files, path)
	if errors.Is(err, ErrNotALink) || errors.Is(err, host.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	hasText, err := l.hasText(ctx, found)
	if err != nil || hasText {
		return err
	}
	if err := l.FetchText(ctx, found); err != nil && ctx.Err() == nil {
		l.reportBackgroundError(ctx, "fetch the text of", found, err)
	}
	return ctx.Err()
}

func (l *Links) hasText(ctx context.Context, found Link) (bool, error) {
	listing, err := l.getTexts(found).ListVersions(ctx, found.Path)
	if err != nil {
		return false, fmt.Errorf("list the text versions of %s: %w", found.Path, err)
	}
	want := domain.TextSourceProperty{Key: urlKey, Value: found.URL}
	for _, version := range listing.Versions {
		if version.PluginID == l.host.PluginID && version.SourceProperty == want {
			return true, nil
		}
	}
	return false, nil
}
