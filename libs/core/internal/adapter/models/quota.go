// Package models manages model quota reservations and folder paths.
package models

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// EnvDir names the folder plugins keep their models in.
	EnvDir = "NUMEN_MODELS_DIR"
	// EnvOffline forbids fetching a model when set.
	EnvOffline = "NUMEN_OFFLINE"
	// EnvQuota is the maximum bytes the models folder may hold.
	EnvQuota = "NUMEN_MODELS_QUOTA"
	// DefaultQuota is the size of the models folder when EnvQuota names none.
	DefaultQuota int64 = 32 << 30
	homeDir            = "numen/models"
)

const (
	lockName         = ".quota.lock"
	staleLock        = 30 * time.Second
	lockRetry        = 50 * time.Millisecond
	reserveSuffix    = ".reserve"
	staleReservation = 10 * time.Minute
	touchInterval    = staleReservation / 4
)

// GetHome returns the shared models folder.
func GetHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, filepath.FromSlash(homeDir)), nil
}

// IsOffline reports whether EnvOffline forbids fetching.
func IsOffline() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(EnvOffline))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// GetQuota returns the byte cap of the models folder.
func GetQuota() int64 {
	if n, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(EnvQuota)), 10, 64); err == nil && n > 0 {
		return n
	}
	return DefaultQuota
}

// GetUsage returns the bytes under dir and active reservations.
func GetUsage(dir string) (int64, error) {
	var sum int64
	err := filepath.WalkDir(resolvePath(dir), func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		sum += info.Size()
		if strings.HasSuffix(name, reserveSuffix) {
			sum += getUnwritten(name, info)
		}
		return nil
	})
	return sum, err
}

func getUnwritten(reservation string, info fs.FileInfo) int64 {
	if time.Since(info.ModTime()) > staleReservation {
		return 0
	}
	raw, err := os.ReadFile(reservation)
	if err != nil {
		return 0
	}
	first, part, _ := strings.Cut(string(raw), "\n")
	promised, err := strconv.ParseInt(strings.TrimSpace(first), 10, 64)
	if err != nil {
		return 0
	}
	var content int64
	if part != "" {
		if target, err := os.Stat(part); err == nil {
			content = target.Size()
		}
	}
	return max(promised-content, 0)
}

// CheckQuota returns an error when need bytes exceeds remaining space under dir.
func CheckQuota(dir string, need int64) error {
	limit := GetQuota()
	used, err := GetUsage(dir)
	if err != nil {
		return fmt.Errorf("counting bytes under %s: %w", dir, err)
	}
	if used+need > limit {
		return fmt.Errorf("%s holds %d bytes, and %d more exceeds %d limit from %s", dir, used, need, limit, EnvQuota)
	}
	return nil
}

func resolvePath(name string) string {
	name = filepath.Clean(name)
	var rest []string
	for current := name; ; {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			for i := len(rest) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, rest[i])
			}
			return resolved
		}
		parent := filepath.Dir(current)
		if parent == current {
			return name
		}
		rest = append(rest, filepath.Base(current))
		current = parent
	}
}

func lockQuota(ctx context.Context, root string) (func() error, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	lock := filepath.Join(root, lockName)
	for {
		err := os.Mkdir(lock, 0o755)
		if err == nil {
			return func() error { return os.Remove(lock) }, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return nil, err
		}
		if info, statErr := os.Stat(lock); statErr == nil && time.Since(info.ModTime()) > staleLock {
			if err := os.Remove(lock); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
			continue
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockRetry):
		}
	}
}

// Reservation tracks bytes claimed under quota until release.
type Reservation struct {
	Room int64
	file string
	stop chan struct{}
	wg   sync.WaitGroup
	once sync.Once
	// touchErr is the first error of refreshing the file; the keep-alive
	// goroutine writes it and Release reads it after the goroutine ends.
	touchErr error
}

// Reserve claims want bytes under root for a download.
func Reserve(ctx context.Context, root string, want int64, part string) (*Reservation, error) {
	return reserve(ctx, root, want, part, touchInterval)
}

func reserve(ctx context.Context, root string, want int64, part string, touch time.Duration) (_ *Reservation, err error) {
	unlock, err := lockQuota(ctx, root)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, unlock()) }()

	used, err := GetUsage(root)
	if err != nil {
		return nil, fmt.Errorf("counting bytes under %s: %w", root, err)
	}
	remaining := max(GetQuota()-used, 0)
	if want < 0 {
		want = remaining
	}
	if want > remaining {
		return nil, fmt.Errorf("%s holds %d bytes, and %d more exceeds %d limit from %s", root, used, want, GetQuota(), EnvQuota)
	}
	file, err := os.CreateTemp(root, "reservation-*"+reserveSuffix)
	if err != nil {
		return nil, err
	}
	_, err = file.WriteString(strconv.FormatInt(want, 10) + "\n" + part)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, errors.Join(err, os.Remove(file.Name()))
	}
	claim := &Reservation{Room: want, file: file.Name(), stop: make(chan struct{})}
	claim.wg.Add(1)
	go claim.keepAlive(touch)
	return claim, nil
}

func (r *Reservation) keepAlive(interval time.Duration) {
	defer r.wg.Done()
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for {
		select {
		case <-r.stop:
			return
		case now := <-tick.C:
			if err := os.Chtimes(r.file, now, now); err != nil && r.touchErr == nil {
				r.touchErr = err
			}
		}
	}
}

// Release surrenders the reservation claim and returns the first error of
// refreshing or removing its file. A second call returns nil.
func (r *Reservation) Release() (err error) {
	r.once.Do(func() {
		close(r.stop)
		r.wg.Wait()
		err = errors.Join(r.touchErr, os.Remove(r.file))
	})
	return err
}
