package textversion

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// liveKey names one file of one vault.
type liveKey struct {
	vault domain.VaultID
	path  string
}

// Changes reports to a listener the files whose versions changed, or whose live version changed with
// the file's content or the note's properties.
type Changes struct {
	texts liveTexts

	mu       sync.Mutex
	listener textdomain.VersionListener
	lives    map[liveKey]domain.TextVersionID
}

// liveTexts finds the live version of a file.
type liveTexts interface {
	FindLive(ctx context.Context, v domain.Vault, path string) ([]domain.TextVersion, domain.TextVersion, bool, error)
}

// NewChanges returns the reporter of the files whose versions changed, over the texts that tell
// which version is live. No listener is notified until SetListener is called.
func NewChanges(texts *Texts) *Changes {
	return &Changes{texts: texts, listener: noListener{}, lives: map[liveKey]domain.TextVersionID{}}
}

type noListener struct{}

func (noListener) ReportVersionsChanged(domain.Vault, string) {}

// SetListener sets the listener of version changes, the index of the displayed
// vault. Nil disables notifications.
func (u *Changes) SetListener(listener textdomain.VersionListener) {
	if listener == nil {
		listener = noListener{}
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	u.listener = listener
}

func (u *Changes) report(ctx context.Context, v domain.Vault, path string) {
	if _, err := u.updateLive(ctx, v, path); err != nil {
		u.mu.Lock()
		delete(u.lives, liveKey{vault: v.ID, path: path})
		u.mu.Unlock()
	}
	u.notify(v, path)
}

func (u *Changes) notify(v domain.Vault, path string) {
	u.mu.Lock()
	listener := u.listener
	u.mu.Unlock()
	listener.ReportVersionsChanged(v, path)
}

// ReportFilesChanged reports to the listener each changed file whose live version is not the one last
// found for it: a version became live or stopped being live because the file's content or the
// note's properties changed. A file with versions whose live version was not found before is
// reported; a file with no versions is not.
func (u *Changes) ReportFilesChanged(ctx context.Context, v domain.Vault, paths []string) error {
	var failures []error
	for _, path := range paths {
		isChanged, err := u.updateLive(ctx, v, path)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if isChanged {
			u.notify(v, path)
		}
	}
	return errors.Join(failures...)
}

func (u *Changes) updateLive(ctx context.Context, v domain.Vault, path string) (bool, error) {
	versions, live, hasLive, err := u.texts.FindLive(ctx, v, path)
	if err != nil && !errors.Is(err, textdomain.ErrFileUnreadable) {
		return false, err
	}
	key := liveKey{vault: v.ID, path: path}
	if len(versions) == 0 {
		u.mu.Lock()
		delete(u.lives, key)
		u.mu.Unlock()
		return false, nil
	}
	var liveID domain.TextVersionID
	if hasLive {
		liveID = live.GetID()
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	previous, isKnown := u.lives[key]
	u.lives[key] = liveID
	return !isKnown || previous != liveID, nil
}
