package transcription

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

// A model is fetched by its address, and a person may point any of them at another place.
//
// A downloaded file is kept under the platform's cache directory, beside what a
// reading fetches. Deleting it costs a download and no knowledge.
const cacheDir = "numen/models"

// opening is how long a host has to answer at all, and slowest is the rate a
// download has to keep up once it is answering. Together they are how long one file has.
const (
	opening = 2 * time.Minute
	slowest = 192 << 10
	assumed = 1 << 30
)

// patience is how long a file of one size has to arrive. A length nobody said
// is taken to be larger than any of these models.
func patience(size int64) time.Duration {
	if size <= 0 {
		size = assumed
	}
	return opening + time.Duration(size/slowest)*time.Second
}

// kept is where a downloaded file is put: the folder the settings name, or this
// platform's cache directory.
func kept(cfg Config) (string, error) {
	if cfg.Dir != "" {
		return cfg.Dir, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, filepath.FromSlash(cacheDir)), nil
}

// fetched is the file one address names, downloaded if it is not already here.
//
// The name it is kept under carries the address, so two models with one filename
// do not collide and a changed address is a different file rather than a stale
// one wearing the right name.
func fetched(ctx context.Context, cfg Config, address string, allowed bool) (string, error) {
	dir, err := kept(cfg)
	if err != nil {
		return "", err
	}
	at := filepath.Join(dir, cached(address))
	if _, err := os.Stat(at); err == nil {
		return at, nil
	}
	if !allowed {
		return "", fmt.Errorf(
			"%s is not on this machine: set transcription.dir to where it is, or transcription.download to fetch it",
			address)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return at, download(ctx, cfg, address, at)
}

// cached is what one address is kept under: the file it ends in, and enough of
// the address to tell two of them apart.
func cached(address string) string {
	sum := sha256.Sum256([]byte(address))
	base := path.Base(address)
	if base == "" || base == "." || base == "/" {
		base = "model.onnx"
	}
	return hex.EncodeToString(sum[:6]) + "-" + base
}

// download writes what one address holds, through a file beside it, so that a
// download interrupted leaves nothing that looks finished.
//
// The deadline is set twice: once for the host to answer, and once for the body
// when its length is known.
func download(ctx context.Context, cfg Config, address, at string) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	waited := time.AfterFunc(opening, stop)
	defer waited.Stop()
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
	waited.Reset(patience(answer.ContentLength))

	part := at + ".part"
	file, err := os.Create(part)
	if err != nil {
		return err
	}
	counted := &counting{
		to:    file,
		total: answer.ContentLength,
		say:   func(done, total int64) { cfg.say(path.Base(address), done, total) },
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
