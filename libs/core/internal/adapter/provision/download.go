package provision

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jiva-studio/numen/libs/core/port/machineport"
)

// ErrChecksum is a download whose bytes do not hold the checksum the requirement pins.
var ErrChecksum = errors.New("the download does not hold the pinned checksum")

const (
	tempPrefix       = ".tmp."
	progressInterval = 100 * time.Millisecond
)

// transfer is one file fetched from an address.
type transfer struct {
	client     *http.Client
	address    string
	dest       string
	sum        string
	backoff    backoff
	clock      machineport.Clock
	onProgress func(count, total int64)
}

// newClient returns a client that follows a redirect only to another https address.
func newClient() *http.Client {
	return &http.Client{
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
}

// fetch leaves at dest a file that holds the checksum. The bytes arrive in a
// .tmp.<uuid> file beside dest; one a failed run left is continued with an HTTP
// Range request, and the file is renamed into place only once its checksum
// matches.
func (t transfer) fetch(ctx context.Context) error {
	dir := filepath.Dir(t.dest)
	part, err := claimPartial(dir)
	if err != nil {
		return err
	}
	err = retry(ctx, t.backoff, isTransient, func() error { return t.append(ctx, part) })
	if err != nil {
		if errors.Is(err, ErrChecksum) {
			return errors.Join(err, os.Remove(part))
		}
		return err
	}
	return renameWithRetry(ctx, part, t.dest, t.backoff)
}

// claimPartial returns the file the bytes of this run go to: the first
// partial a failed run left, or a new one. Any other partial is removed.
func claimPartial(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var partial string
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), tempPrefix) {
			continue
		}
		if partial == "" {
			partial = filepath.Join(dir, entry.Name())
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return "", err
		}
	}
	if partial != "" {
		return partial, nil
	}
	return filepath.Join(dir, tempPrefix+uuid.NewString()), nil
}

// append continues the partial from where it ends, then checks the checksum.
func (t transfer) append(ctx context.Context, part string) (err error) {
	hasher := sha256.New()
	offset, err := hashExisting(part, hasher)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, t.address, nil)
	if err != nil {
		return err
	}
	if offset > 0 {
		request.Header.Set("Range", "bytes="+strconv.FormatInt(offset, 10)+"-")
	}
	answer, err := t.client.Do(request)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", t.address, err)
	}
	defer closeAndJoin(&err, answer.Body)

	switch {
	case answer.StatusCode == http.StatusRequestedRangeNotSatisfiable:
		return t.finish(hasher, offset)
	case answer.StatusCode == http.StatusPartialContent && offset > 0:
	case answer.StatusCode == http.StatusOK:
		offset = 0
		hasher.Reset()
	default:
		return &statusError{address: t.address, status: answer.Status, code: answer.StatusCode}
	}
	total := answer.ContentLength
	if total >= 0 {
		total += offset
	}
	flags := os.O_WRONLY | os.O_CREATE
	if offset == 0 {
		flags |= os.O_TRUNC
	} else {
		flags |= os.O_APPEND
	}
	file, err := os.OpenFile(part, flags, 0o644)
	if err != nil {
		return err
	}
	byteCount, err := copyProgress(t.clock, io.MultiWriter(file, hasher), answer.Body, offset, total, t.onProgress)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("fetching %s: %w", t.address, err)
	}
	return t.finish(hasher, offset+byteCount)
}

// finish checks the checksum of the whole file. A file that does not hold it is
// an error that ends the run.
func (t transfer) finish(hasher hash.Hash, size int64) error {
	if found := hex.EncodeToString(hasher.Sum(nil)); found != t.sum {
		return fmt.Errorf("%w: %s holds %s, and %s is pinned", ErrChecksum, t.address, found, t.sum)
	}
	if t.onProgress != nil {
		t.onProgress(size, size)
	}
	return nil
}

// hashExisting feeds the bytes the file holds to the hash and returns how many there are.
func hashExisting(path string, hasher hash.Hash) (_ int64, err error) {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	defer closeAndJoin(&err, file)
	return io.Copy(hasher, file)
}

// copyProgress copies the body to the writer and reports how far it has got no
// more often than progressInterval.
func copyProgress(clock machineport.Clock, dst io.Writer, src io.Reader, offset, total int64, onProgress func(count, total int64)) (int64, error) {
	buffer := make([]byte, 64*1024)
	var byteCount int64
	last := clock.Now()
	for {
		count, readErr := src.Read(buffer)
		if count > 0 {
			if _, err := dst.Write(buffer[:count]); err != nil {
				return byteCount, err
			}
			byteCount += int64(count)
			if onProgress != nil && clock.Now().Sub(last) >= progressInterval {
				onProgress(offset+byteCount, total)
				last = clock.Now()
			}
		}
		if errors.Is(readErr, io.EOF) {
			return byteCount, nil
		}
		if readErr != nil {
			return byteCount, readErr
		}
	}
}

// statusError is an HTTP answer that is not the file.
type statusError struct {
	address string
	status  string
	code    int
}

func (e *statusError) Error() string { return fmt.Sprintf("fetching %s: %s", e.address, e.status) }

// isTransient reports whether another attempt may succeed: a broken
// connection or a server that is busy, and never a checksum that failed.
func isTransient(err error) bool {
	if errors.Is(err, ErrChecksum) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var status *statusError
	if errors.As(err, &status) {
		return status.code >= 500 || status.code == http.StatusTooManyRequests
	}

	return !isAccessDenied(err)
}
