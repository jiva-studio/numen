// Package models is where the models a plugin runs on are kept, how a plugin
// fetches one, and what a fetched file is checked against.
package models

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// The environment variables a person sets to move, cap or switch off the
// models a plugin keeps.
const (
	// EnvDir names the folder every plugin keeps its models in.
	EnvDir = "NUMEN_MODELS_DIR"
	// EnvOffline, when true, forbids any plugin to fetch a model.
	EnvOffline = "NUMEN_OFFLINE"
	// EnvQuota is the most bytes the models folder may hold.
	EnvQuota = "NUMEN_MODELS_QUOTA"
)

// DefaultQuota is the size of the models folder when EnvQuota names none.
const DefaultQuota int64 = 32 << 30

// homeDir is the models folder under the application data directory.
const homeDir = "numen/models"

// GetHome returns the one folder the plugins share their models in: EnvDir, or
// numen/models under the application data directory.
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

// GetQuota returns the most bytes the models folder may hold.
func GetQuota() int64 {
	if n, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(EnvQuota)), 10, 64); err == nil && n > 0 {
		return n
	}
	return DefaultQuota
}

// reserveSuffix ends the name of a reservation, the file that holds the bytes a
// download will hold and, on a second line, the file it writes.
// staleReservation is how long a reservation that is not touched keeps counting.
const (
	reserveSuffix    = ".reserve"
	staleReservation = 10 * time.Minute
)

// GetUsage returns the bytes the files under dir hold, and the bytes the
// downloads in progress under it have reserved and not yet written. A folder
// that does not exist holds none.
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

// getUnwritten returns the bytes a live reservation has promised and its
// download has not yet written. The reservation file holds the bytes promised,
// and on a second line the file the download writes.
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

// CheckQuota returns an error when need more bytes would take dir past the quota.
func CheckQuota(dir string, need int64) error {
	limit := GetQuota()
	used, err := GetUsage(dir)
	if err != nil {
		return fmt.Errorf("counting the bytes under %s: %w", dir, err)
	}
	if used+need > limit {
		return fmt.Errorf("%s holds %d bytes, and %d more would pass the %d the %s allows", dir, used, need, limit, EnvQuota)
	}
	return nil
}

// resolvePath returns name with the symbolic links of its existing part
// resolved. The part that does not exist yet is kept as written.
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
