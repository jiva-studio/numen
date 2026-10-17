// Package models is where the models a plugin runs on are kept, how a plugin
// fetches one, and what a fetched file is checked against.
package models

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

// homeDir is the models folder under the platform's cache directory.
const homeDir = "numen/models"

// GetHome returns the one folder the plugins share their models in: EnvDir, or
// numen/models under the platform's cache directory.
func GetHome() (string, error) {
	if dir := os.Getenv(EnvDir); dir != "" {
		return dir, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, filepath.FromSlash(homeDir)), nil
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

// GetUsage returns the bytes the files under dir hold. A folder that does not
// exist holds none.
func GetUsage(dir string) int64 {
	var sum int64
	_ = filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, err := entry.Info(); err == nil {
			sum += info.Size()
		}
		return nil
	})
	return sum
}

// CheckQuota returns an error when need more bytes would take dir past the quota.
func CheckQuota(dir string, need int64) error {
	limit := GetQuota()
	if used := GetUsage(dir); used+need > limit {
		return fmt.Errorf("%s holds %d bytes, and %d more would pass the %d the %s allows", dir, used, need, limit, EnvQuota)
	}
	return nil
}
