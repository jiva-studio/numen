package index

import (
	"context"
	"crypto/sha256"
	"errors"
	"io/fs"
	"reflect"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// checkFunc brings the entry of one note up to date and reports whether its task list changed.
type checkFunc func(ctx context.Context, path string) (bool, error)

// visit runs check for each path, with at most maxReads at once, and reports whether any call
// changed a task list. It stops at the first failure of the host or when ctx is done.
func (i *Index) visit(ctx context.Context, paths []string, check checkFunc) (bool, error) {
	var (
		group     sync.WaitGroup
		mu        sync.Mutex
		isChanged bool
		failure   error
	)
	slots := make(chan struct{}, maxReads)
loop:
	for _, path := range paths {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			break loop
		}
		group.Go(func() {
			defer func() { <-slots }()
			changed, err := check(ctx, path)
			mu.Lock()
			defer mu.Unlock()
			isChanged = isChanged || changed
			if failure == nil {
				failure = err
			}
		})
	}
	group.Wait()
	if failure == nil {
		failure = ctx.Err()
	}
	return isChanged, failure
}

// checkPaths brings the entries of the paths up to date and changes the revision when a task list
// changed, or when isChanged says the caller changed one.
func (i *Index) checkPaths(ctx context.Context, paths []string, isChanged bool) error {
	changed, err := i.visit(ctx, paths, i.checkNote)
	if changed || isChanged {
		i.mu.Lock()
		i.bump()
		i.mu.Unlock()
	}
	return err
}

// checkNote reads a note again when its file stat is not the one the index holds.
func (i *Index) checkNote(ctx context.Context, path string) (bool, error) {
	stat, err := i.files.StatFile(ctx, path)
	if err != nil {
		return i.recordFailure(ctx, path, err)
	}
	if stat.Kind != domain.KindNote {
		return i.drop(path), nil
	}
	i.mu.Lock()
	known, isKnown := i.entries[path]
	i.mu.Unlock()
	if isKnown && known.stat.IsUnchanged(stat) && known.stat.FileID == stat.FileID {
		return false, nil
	}
	return i.readNote(ctx, path)
}

// readNote reads a note and parses it, unless its bytes are the ones the index parsed before.
func (i *Index) readNote(ctx context.Context, path string) (bool, error) {
	raw, stat, err := i.files.Read(ctx, path)
	if err != nil {
		return i.recordFailure(ctx, path, err)
	}
	hash := sha256.Sum256(raw)
	i.mu.Lock()
	known, isKnown := i.entries[path]
	i.mu.Unlock()
	if isKnown && known.hash == hash {
		known.stat = stat
		i.set(path, known)
		return false, nil
	}
	next := entry{stat: stat, hash: hash}
	tasks, err := i.parse(raw)
	if err != nil {
		next.isUnreadable = true
	}
	for n := range tasks {
		tasks[n].Path = path
	}
	next.tasks = tasks
	i.set(path, next)
	return !isSame(known, next), nil
}

// recordFailure handles a note that cannot be statted or read. A failure of the host is returned. A
// note that is gone or is not a note loses its entry; any other note is unreadable.
func (i *Index) recordFailure(ctx context.Context, path string, err error) (bool, error) {
	if ctx.Err() != nil || errors.Is(err, host.ErrNoVault) || errors.Is(err, host.ErrUnknownVault) {
		return false, err
	}
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
		return i.drop(path), nil
	}
	i.mu.Lock()
	known := i.entries[path]
	i.entries[path] = entry{isUnreadable: true}
	i.mu.Unlock()
	return !isSame(known, entry{isUnreadable: true}), nil
}

// drop removes the entry of a note and reports whether it held tasks or was unreadable.
func (i *Index) drop(path string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	known, isKnown := i.entries[path]
	delete(i.entries, path)
	return isKnown && !isSame(known, entry{})
}

func (i *Index) set(path string, next entry) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.entries[path] = next
}

// isSame reports whether two entries show the same tasks and the same readability.
func isSame(a, b entry) bool {
	if a.isUnreadable != b.isUnreadable || len(a.tasks) != len(b.tasks) {
		return false
	}
	return len(a.tasks) == 0 || reflect.DeepEqual(a.tasks, b.tasks)
}
