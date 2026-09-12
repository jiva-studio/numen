package onnxruntime

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

// A file is fetched by its address, and a person may point any of them at
// another place.
//
// A downloaded file is kept under the platform's cache directory. Deleting it
// costs a download and no knowledge.
const cacheDir = "numen/models"

// opening is how long a host has to answer at all, and slowest is the rate a
// download has to keep up once it is answering. Together they are how long one
// file has.
const (
	opening = 2 * time.Minute
	slowest = 192 << 10
	assumed = 1 << 30
)

// patience is how long a file of one size has to arrive. A length nobody said
// is taken to be larger than any of these files.
func patience(size int64) time.Duration {
	if size <= 0 {
		size = assumed
	}
	return opening + time.Duration(size/slowest)*time.Second
}

// fetching follows a redirect only to another https address.
var fetching = &http.Client{
	CheckRedirect: func(request *http.Request, via []*http.Request) error {
		if request.URL.Scheme != "https" {
			return fmt.Errorf("%s redirects to %s", via[len(via)-1].URL, request.URL)
		}
		if len(via) >= 10 {
			return fmt.Errorf("%s redirects ten times", via[0].URL)
		}
		return nil
	},
}

// CacheDir is where a downloaded file is put: the folder the settings name, or
// this platform's cache directory.
func CacheDir(s Settings) (string, error) {
	if s.Dir != "" {
		return s.Dir, nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, filepath.FromSlash(cacheDir)), nil
}

// Fetch is the file one address names, downloaded if it is not already here.
//
// Nothing checked here. The runtime's caller checks what it fetched against the
// sums this build carries, because the runtime's address is this build's: one
// release a platform, published once and never republished, so a sum is what
// this build says about a file it chose.
//
// A model's address is the person's. `indexing.recognition` and
// `indexing.transcription` name it and may be pointed anywhere, and a sum
// beside a name they chose is a number only they could supply and have nowhere
// to get. Most of the defaults name a branch rather than a revision, and a
// branch is written over, so a sum on those would refuse the model the day its
// publisher republished it. A model a person names is a model a person trusts,
// and it is fetched as named.
func Fetch(ctx context.Context, s Settings, address string) (string, error) {
	dir, err := CacheDir(s)
	if err != nil {
		return "", err
	}
	at := filepath.Join(dir, GetCacheName(address))
	if _, err := os.Stat(at); err == nil {
		return at, nil
	}
	if !s.Download {
		return "", fmt.Errorf("%s is not on this machine: set %s.dir to where it is, or %s.download to fetch it",
			address, s.Section, s.Section)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return at, download(ctx, s, address, at)
}

// GetCacheName is what one address is kept under: the file it ends in, and
// enough of the address to tell two of them apart.
//
// The name carries the address, so two files with one filename do not collide
// and a changed address is a different file.
func GetCacheName(address string) string {
	sum := sha256.Sum256([]byte(address))
	base := path.Base(address)
	if base == "" || base == "." || base == "/" {
		base = "model.onnx"
	}
	return hex.EncodeToString(sum[:6]) + "-" + base
}

// IsAddress says whether what the settings named is somewhere to fetch from
// rather than a file on this machine. What is fetched is fetched over https.
func IsAddress(name string) bool {
	return strings.HasPrefix(name, "https://")
}

// download writes what one address holds, through a file beside it, so that a
// download interrupted leaves nothing that looks finished.
//
// The deadline is set twice: once for the host to answer, and once for the body
// when its length is known.
func download(ctx context.Context, s Settings, address, at string) error {
	if !IsAddress(address) {
		return fmt.Errorf("%s is not an https address", address)
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	waited := time.AfterFunc(opening, stop)
	defer waited.Stop()
	s.say(path.Base(address), 0, 0)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return err
	}
	answer, err := fetching.Do(request)
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
		say:   func(done, total int64) { s.say(path.Base(address), done, total) },
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
