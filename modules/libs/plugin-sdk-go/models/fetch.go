package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// headerTimeout is how long a host has to answer at all, and minBytesPerSecond
// is the rate a download has to keep up once it is answering. Together they
// are how long one file has.
const (
	headerTimeout     = 2 * time.Minute
	minBytesPerSecond = 192 << 10
	unknownSizeBytes  = 1 << 30
)

// getDownloadTimeout returns how long a file of one size has to arrive. A size
// nobody said is taken to be larger than any model.
func getDownloadTimeout(size int64) time.Duration {
	if size <= 0 {
		size = unknownSizeBytes
	}
	return headerTimeout + time.Duration(size/minBytesPerSecond)*time.Second
}

// client follows a redirect only to another https address.
var client = &http.Client{
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

// IsAddress reports whether name is an https address.
func IsAddress(name string) bool {
	return strings.HasPrefix(name, "https://")
}

// GetCacheName returns the file name an address is kept under. A pinned
// address is kept under its sum, so a file that is present has the sum it was
// pinned to; any other is kept under a digest of the address.
func GetCacheName(address, sum string) string {
	base := path.Base(address)
	if base == "" || base == "." || base == "/" {
		base = "model.onnx"
	}
	if sum != "" {
		return sum[:min(len(sum), 12)] + "-" + base
	}
	digest := sha256.Sum256([]byte(address))
	return hex.EncodeToString(digest[:6]) + "-" + base
}

// VerifyFile returns an error when the file at name does not hold the sha256 sum.
func VerifyFile(name, sum string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if found := hex.EncodeToString(hash.Sum(nil)); found != strings.ToLower(sum) {
		return fmt.Errorf("%s holds %s, and %s is pinned", name, found, sum)
	}
	return nil
}

// Request is one file to find in a folder or to fetch into it.
type Request struct {
	// Dir is the folder the file is kept in.
	Dir string
	// Section is the settings block a person is told to change.
	Section string
	// Address is the https address the file is fetched from.
	Address string
	// Sum is the sha256 the file must hold. Empty means the file is fetched as named.
	Sum string
	// ShouldDownload allows fetching a file that is not in Dir.
	ShouldDownload bool
	// Progress is told how far a download has got.
	Progress func(what string, done, total int64)
}

func (r Request) reportProgress(done, total int64) {
	if r.Progress != nil {
		r.Progress(path.Base(r.Address), done, total)
	}
}

// Fetch returns the file Request names, downloading it into Dir when it is not
// there and downloads are allowed. A download is written beside its place,
// checked against the sum and the quota, and only then put in its place.
func Fetch(ctx context.Context, r Request) (string, error) {
	at := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	if _, err := os.Stat(at); err == nil {
		return at, nil
	}
	if !r.ShouldDownload {
		return "", fmt.Errorf("%s is not on this machine: set %s.dir to where it is, or %s.download to fetch it",
			r.Address, r.Section, r.Section)
	}
	if IsOffline() {
		return "", fmt.Errorf("%s is not on this machine, and %s forbids fetching it", r.Address, EnvOffline)
	}
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		return "", err
	}
	return at, download(ctx, r, at)
}

// download writes what one address holds through a file beside at.
//
// The deadline is set twice: once for the host to answer, and once for the body
// when its length is known.
func download(ctx context.Context, r Request, at string) error {
	if !IsAddress(r.Address) {
		return fmt.Errorf("%s is not an https address", r.Address)
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	waited := time.AfterFunc(headerTimeout, stop)
	defer waited.Stop()
	r.reportProgress(0, 0)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.Address, nil)
	if err != nil {
		return err
	}
	answer, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	defer answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", r.Address, answer.Status)
	}
	if answer.ContentLength > 0 {
		if err := CheckQuota(r.Dir, answer.ContentLength); err != nil {
			return fmt.Errorf("fetching %s: %w", r.Address, err)
		}
	}
	waited.Reset(getDownloadTimeout(answer.ContentLength))

	part := at + ".part"
	if err := write(part, answer.Body, r, answer.ContentLength); err != nil {
		os.Remove(part)
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	return os.Rename(part, at)
}

// write copies body into the file at part, and returns an error when the copy
// passes the quota, or holds other than the sum.
func write(part string, body io.Reader, r Request, total int64) error {
	file, err := os.Create(part)
	if err != nil {
		return err
	}
	room := max(GetQuota()-GetUsage(r.Dir), 0)
	hash := sha256.New()
	counted := &progressWriter{
		to:         io.MultiWriter(file, hash),
		total:      total,
		onProgress: r.reportProgress,
	}
	copied, err := io.Copy(counted, io.LimitReader(body, room+1))
	if err == nil && copied > room {
		err = errors.New("the download passes the quota " + EnvQuota)
	}
	if err == nil && r.Sum != "" {
		if found := hex.EncodeToString(hash.Sum(nil)); found != strings.ToLower(r.Sum) {
			err = fmt.Errorf("the download holds %s, and %s is pinned", found, r.Sum)
		}
	}
	if err == nil {
		err = file.Sync()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}

// progressWriter is a writer that says how far it has got, no more than once a second.
type progressWriter struct {
	to         io.Writer
	total      int64
	done       int64
	last       time.Time
	onProgress func(done, total int64)
}

func (c *progressWriter) Write(raw []byte) (int, error) {
	n, err := c.to.Write(raw)
	c.done += int64(n)
	if now := time.Now(); now.Sub(c.last) > time.Second {
		c.last = now
		c.onProgress(c.done, c.total)
	}
	return n, err
}
