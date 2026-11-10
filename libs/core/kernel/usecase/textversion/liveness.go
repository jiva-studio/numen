package textversion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

// liveKey names one file of one vault.
type liveKey struct {
	vault domain.VaultID
	path  string
}

// ReportFilesChanged reports to the listener each changed file whose live version is not the one last
// found for it: a version became live or stopped being live because the file's content or the
// note's properties changed. A file with versions whose live version was not found before is
// reported; a file with no versions is not.
func (u *Versions) ReportFilesChanged(ctx context.Context, v domain.Vault, paths []string) error {
	var failures []error
	for _, path := range paths {
		isChanged, err := u.updateLive(ctx, v, path)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		if isChanged {
			u.notifyListener(v, path)
		}
	}
	return errors.Join(failures...)
}

func (u *Versions) updateLive(ctx context.Context, v domain.Vault, path string) (bool, error) {
	versions, err := u.getVersions(ctx, v, path)
	if err != nil {
		return false, err
	}
	key := liveKey{vault: v.ID, path: path}
	if len(versions) == 0 {
		u.mu.Lock()
		delete(u.lives, key)
		u.mu.Unlock()
		return false, nil
	}
	live, err := u.findCurrentLive(ctx, v, path, versions)
	if err != nil {
		return false, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	previous, isKnown := u.lives[key]
	u.lives[key] = live
	return !isKnown || previous != live, nil
}

func (u *Versions) findCurrentLive(
	ctx context.Context, v domain.Vault, path string, versions []domain.TextVersion,
) (domain.TextVersionID, error) {
	contentHash, err := u.GetContentHash(ctx, v, path)
	if errors.Is(err, textdomain.ErrFileUnreadable) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	live, ok, err := u.findLive(ctx, v, path, versions, contentHash)
	if err != nil || !ok {
		return "", err
	}
	return live.GetID(), nil
}
