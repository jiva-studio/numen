package index

import (
	"context"
	"crypto/sha256"
	"errors"
	"reflect"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// checkFunc brings the entry of one note up to date and reports whether its task list changed.
type checkFunc func(ctx context.Context, path string) (bool, error)

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

func (i *Index) checkPaths(ctx context.Context, paths []string, isChanged bool) error {
	changed, err := i.visit(ctx, paths, i.checkNote)
	if changed || isChanged {
		i.mu.Lock()
		i.bump()
		i.mu.Unlock()
	}
	return err
}

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

func (i *Index) readNote(ctx context.Context, path string) (bool, error) {
	raw, stat, err := i.files.Read(ctx, path)
	if err != nil {
		return i.recordFailure(ctx, path, err)
	}
	return i.storeNote(path, raw, stat), nil
}

func (i *Index) storeNote(path string, raw []byte, stat domain.FileStat) bool {
	hash := sha256.Sum256(raw)
	i.mu.Lock()
	known, isKnown := i.entries[path]
	i.mu.Unlock()
	if isKnown && known.hash == hash {
		known.stat = stat
		i.set(path, known)
		return false
	}
	next := entry{stat: stat, hash: hash, tasks: nil, isUnreadable: false}
	tasks, err := i.parse(raw)
	if err != nil {
		next.isUnreadable = true
	}
	for n := range tasks {
		tasks[n].Path = path
	}
	next.tasks = tasks
	i.set(path, next)
	return !isSame(known, next)
}

func (i *Index) recordFailure(ctx context.Context, path string, err error) (bool, error) {
	if ctx.Err() != nil || errors.Is(err, host.ErrNoVault) || errors.Is(err, host.ErrUnknownVault) {
		return false, err
	}
	if errors.Is(err, host.ErrNotExist) || errors.Is(err, host.ErrNotANote) {
		return i.drop(path), nil
	}
	i.mu.Lock()
	unreadable := entry{stat: domain.FileStat{}, hash: [sha256.Size]byte{}, tasks: nil, isUnreadable: true}
	known := i.entries[path]
	i.entries[path] = unreadable
	i.mu.Unlock()
	return !isSame(known, unreadable), nil
}

func (i *Index) drop(path string) bool {
	i.mu.Lock()
	defer i.mu.Unlock()
	known, isKnown := i.entries[path]
	delete(i.entries, path)
	return isKnown && !isSame(known, emptyEntry())
}

func (i *Index) set(path string, next entry) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.entries[path] = next
}

func isSame(a, b entry) bool {
	if a.isUnreadable != b.isUnreadable || len(a.tasks) != len(b.tasks) {
		return false
	}
	return len(a.tasks) == 0 || reflect.DeepEqual(a.tasks, b.tasks)
}

func emptyEntry() entry {
	return entry{stat: domain.FileStat{}, hash: [sha256.Size]byte{}, tasks: nil, isUnreadable: false}
}
