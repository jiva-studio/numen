package onnx

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Models are fetched by their address rather than through a library that knows
// one place to look. Two of these come from one host and one from another,
// because the export that carries its own alphabet is not the export the first
// host publishes — and a person may point any of them at a fourth place.
//
// A downloaded file is kept under the platform's cache directory. Deleting it
// costs a download and no knowledge.
const cacheDir = "numen/models"

// fetching is how long one model has to arrive. The largest of them is 124 MB,
// and a download that has stalled is better said out loud than waited on.
const fetching = 10 * time.Minute

// fetched is the file one address names, downloaded if it is not already here.
//
// The name it is kept under carries the address, so two models with one filename
// do not collide and a changed address is a different file rather than a stale
// one wearing the right name.
func fetched(ctx context.Context, cfg Config, address string, allowed bool) (string, error) {
	dir := cfg.Dir
	if dir == "" {
		cache, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(cache, filepath.FromSlash(cacheDir))
	}
	at := filepath.Join(dir, named(address))
	if _, err := os.Stat(at); err == nil {
		return at, nil
	}
	if !allowed {
		return "", fmt.Errorf(
			"%s is not on this machine: set recognition.dir to where it is, or recognition.download to fetch it",
			address)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return at, download(ctx, cfg, address, at)
}

// named is what one address is kept under: the file it ends in, and enough of
// the address to tell two of them apart.
func named(address string) string {
	sum := sha256.Sum256([]byte(address))
	base := path.Base(address)
	if base == "" || base == "." || base == "/" {
		base = "model.onnx"
	}
	return hex.EncodeToString(sum[:6]) + "-" + base
}

// download writes what one address holds, through a file beside it, so that a
// download interrupted leaves nothing that looks finished.
func download(ctx context.Context, cfg Config, address, at string) error {
	ctx, stop := context.WithTimeout(ctx, fetching)
	defer stop()
	cfg.say(path.Base(address), 0, 0)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	answer, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", address, err)
	}
	defer answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", address, answer.Status)
	}

	part := at + ".part"
	file, err := os.Create(part)
	if err != nil {
		return err
	}
	counted := &counting{
		to:    file,
		total: answer.ContentLength,
		// A hundred and sixty megabytes is minutes, and a person watching a
		// number that never moves is a person watching a program that hung.
		say: func(done, total int64) { cfg.say(path.Base(address), done, total) },
	}
	if _, err := io.Copy(counted, answer.Body); err != nil {
		file.Close()
		os.Remove(part)
		return fmt.Errorf("fetching %s: %w", address, err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		os.Remove(part)
		return err
	}
	if err := file.Close(); err != nil {
		os.Remove(part)
		return err
	}
	return os.Rename(part, at)
}

// address says whether what the settings named is somewhere to fetch from
// rather than a file on this machine.
func address(name string) bool {
	return strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "http://")
}

// counting is a writer that says how far it has got.
//
// It says so no more than once a second, because what reads it is drawn on a
// screen and a number that changes faster than a person can read it is a number
// nobody reads.
type counting struct {
	to    io.Writer
	total int64
	done  int64
	last  time.Time
	say   func(done, total int64)
}

func (c *counting) Write(raw []byte) (int, error) {
	n, err := c.to.Write(raw)
	c.done += int64(n)
	if now := time.Now(); now.Sub(c.last) > time.Second {
		c.last = now
		c.say(c.done, c.total)
	}
	return n, err
}
