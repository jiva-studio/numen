package models

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
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
// address is kept under its sum; any other is kept under a digest of the address.
func GetCacheName(address, sum string) string {
	base := getBaseName(address)
	if sum != "" {
		return sum[:min(len(sum), 12)] + "-" + base
	}
	digest := sha256.Sum256([]byte(address))
	return hex.EncodeToString(digest[:6]) + "-" + base
}

// getBaseName returns the last element of the address path, without its query.
func getBaseName(address string) string {
	name := address
	if u, err := url.Parse(address); err == nil {
		name = u.Path
	}
	base := path.Base(name)
	if base == "" || base == "." || base == "/" {
		return "model.onnx"
	}
	return base
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
		r.Progress(getBaseName(r.Address), done, total)
	}
}

// Fetch returns the file Request names, downloading it into Dir when it is not
// there, or is there with another sum, and downloads are allowed. A download is
// written beside its place, checked against the sum and the quota, and only
// then put in its place.
func Fetch(ctx context.Context, r Request) (string, error) {
	at := filepath.Join(r.Dir, GetCacheName(r.Address, r.Sum))
	var corrupt error
	if _, err := os.Stat(at); err == nil {
		if r.Sum == "" {
			return at, nil
		}
		if corrupt = VerifyFile(at, r.Sum); corrupt == nil {
			return at, nil
		}
	}
	if !r.ShouldDownload {
		if corrupt != nil {
			return "", corrupt
		}
		return "", fmt.Errorf("%s is not on this machine: set %s.dir to where it is, or %s.download to fetch it",
			r.Address, r.Section, r.Section)
	}
	if IsOffline() {
		if corrupt != nil {
			return "", corrupt
		}
		return "", fmt.Errorf("%s is not on this machine, and %s forbids fetching it", r.Address, EnvOffline)
	}
	if err := os.MkdirAll(r.Dir, 0o755); err != nil {
		return "", err
	}
	if corrupt != nil {
		if err := os.Remove(at); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
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

	file, err := os.CreateTemp(r.Dir, filepath.Base(at)+".part-*")
	if err != nil {
		return err
	}
	part := file.Name()
	claim, err := Reserve(ctx, getQuotaRoot(r), answer.ContentLength, part)
	if err != nil {
		file.Close()
		os.Remove(part)
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	defer claim.Release()
	waited.Reset(getDownloadTimeout(answer.ContentLength))

	if err := write(file, answer.Body, r, claim, answer.ContentLength); err != nil {
		os.Remove(part)
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	if err := os.Rename(part, at); err != nil {
		os.Remove(part)
		return err
	}
	return nil
}

// write copies body into file and closes it. It returns an error when the copy
// passes the room the claim holds, or holds other than the sum.
func write(file *os.File, body io.Reader, r Request, claim *Reservation, total int64) error {
	hash := sha256.New()
	counted := &progressWriter{
		to:         io.MultiWriter(file, hash),
		total:      total,
		onProgress: r.reportProgress,
	}
	copied, err := io.Copy(counted, io.LimitReader(body, claim.Room+1))
	if err == nil && copied > claim.Room {
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

// getQuotaRoot returns the folder the quota counts: the shared models folder
// when the request keeps its files in it, and the request's own folder
// otherwise.
func getQuotaRoot(r Request) string {
	home, err := GetHome()
	if err != nil {
		return r.Dir
	}
	relative, err := filepath.Rel(resolvePath(home), resolvePath(r.Dir))
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return r.Dir
	}
	return home
}
