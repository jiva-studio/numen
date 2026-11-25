package pull

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
)

const (
	// KeyUnsaved is the prefix of the health item of files the host could not write.
	KeyUnsaved = "unsaved-files"

	firstRetry = time.Minute
	lastRetry  = 30 * time.Minute
)

// GetUnsavedText is what the warning of files this computer cannot write says.
func GetUnsavedText(count int) string {
	if count == 1 {
		return "1 file from the sync server could not be saved on this computer"
	}
	return fmt.Sprintf("%d files from the sync server could not be saved on this computer", count)
}

// fileError is a refusal of the host to write one file. It stops that file only.
type fileError struct {
	path  string
	cause error
}

func (e *fileError) Error() string { return fmt.Sprintf("write %s: %v", e.path, e.cause) }

func (e *fileError) Unwrap() error { return e.cause }

func getRefusal(ctx context.Context, err error) (*fileError, bool) {
	var refusal *fileError
	return refusal, errors.As(err, &refusal) && ctx.Err() == nil
}

type retryKey struct {
	vault domain.VaultID
	file  string
}

type retry struct {
	rev      string
	failures int
	until    time.Time
}

// Retries is when each file the host refused is tried again: a minute after the first refusal, then twice as long
// after each, up to half an hour. A new revision of the file is tried at once. They are kept while the plugin runs.
type Retries struct {
	mu   sync.Mutex
	held map[retryKey]retry
}

// NewRetries returns the waits of no file.
func NewRetries() *Retries { return &Retries{mu: sync.Mutex{}, held: map[retryKey]retry{}} }

func (r *Retries) isWaiting(key retryKey, rev string, now time.Time) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	one, isHeld := r.held[key]
	return isHeld && one.rev == rev && now.Before(one.until)
}

func (r *Retries) fail(key retryKey, rev string, now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	one := r.held[key]
	if one.rev != rev {
		one = retry{rev: rev, failures: 0, until: time.Time{}}
	}
	one.failures++
	one.until = now.Add(min(firstRetry<<(one.failures-1), lastRetry))
	r.held[key] = one
}

func (r *Retries) forget(key retryKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.held, key)
}

func (r *run) getRetryKey(stored *syncv1.StoredRevision) retryKey {
	return retryKey{vault: r.in.VaultID, file: string(stored.GetRevision().GetFileId())}
}

// skipFile leaves a file the host refused as it is: the position stays before its revision, the file is named in the
// warning and it is tried again after the wait.
func (r *run) skipFile(stored *syncv1.StoredRevision, refusal *fileError) {
	r.retries.fail(r.getRetryKey(stored), string(stored.GetRevId()), r.deps.Clock.Now())
	r.keepUnsaved(stored, refusal.path)
}

func (r *run) keepUnsaved(stored *syncv1.StoredRevision, path string) {
	r.hold(stored.GetSeq())
	if !slices.Contains(r.unsaved, path) {
		r.unsaved = append(r.unsaved, path)
	}
}

func (r *run) isWaiting(stored *syncv1.StoredRevision) bool {
	return r.retries.isWaiting(r.getRetryKey(stored), string(stored.GetRevId()), r.deps.Clock.Now())
}

func (r *run) reportUnsaved(ctx context.Context) error {
	key := healthkey.Create(KeyUnsaved, string(r.in.VaultID))
	if len(r.unsaved) == 0 {
		return r.deps.Health.Clear(ctx, key)
	}
	detail := strings.Join(r.unsaved, ", ")
	if len(detail) > health.MaxDetailLength {
		detail = strings.ToValidUTF8(detail[:health.MaxDetailLength], "")
	}
	return r.deps.Health.Report(ctx, health.Item{Key: key, Severity: health.Warning, Text: GetUnsavedText(len(r.unsaved)), Detail: detail, Action: nil})
}
