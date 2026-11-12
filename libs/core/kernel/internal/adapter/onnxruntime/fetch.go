package onnxruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/atomicfile"
)

const (
	// downloadTimeout is how long the archive has to arrive.
	downloadTimeout = 15 * time.Minute

	// maxArchiveBytes is the most bytes an archive may hold.
	maxArchiveBytes = 1 << 30
)

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

func verifyFile(name, sum string) (err error) {
	info, err := os.Stat(name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%s is not a file", name)
	}
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer closeAndJoin(&err, file)
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if found := hex.EncodeToString(hash.Sum(nil)); found != sum {
		return fmt.Errorf("%s holds %s, and %s is pinned", name, found, sum)
	}
	return nil
}

func download(ctx context.Context, client *http.Client, r Release, pos string, onProgress func(count, total int64)) (err error) {
	ctx, cancel := context.WithTimeout(ctx, downloadTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, r.Address, nil)
	if err != nil {
		return err
	}
	answer, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	defer closeAndJoin(&err, answer.Body)
	if answer.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", r.Address, answer.Status)
	}

	check := &archiveReader{
		source:   io.LimitReader(answer.Body, maxArchiveBytes+1),
		digest:   sha256.New(),
		progress: progressWriter{total: answer.ContentLength, onProgress: onProgress},
		wanted:   r.Sum,
	}
	if err := atomicfile.WriteFile(ctx, pos, check, 0o644); err != nil {
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	return nil
}

// archiveReader hands on the bytes of an archive and, at their end, refuses an archive that is too
// large or does not hash to the pinned sum, so the write that reads it never replaces the file with
// it.
type archiveReader struct {
	source io.Reader
	digest interface {
		io.Writer
		Sum(prefix []byte) []byte
	}
	progress progressWriter
	wanted   string
	count    int64
}

func (a *archiveReader) Read(chunk []byte) (int, error) {
	n, err := a.source.Read(chunk)
	a.count += int64(n)
	if _, writeErr := a.digest.Write(chunk[:n]); writeErr != nil {
		return n, writeErr
	}
	if _, progressErr := a.progress.Write(chunk[:n]); progressErr != nil {
		return n, progressErr
	}
	if a.count > maxArchiveBytes {
		return n, fmt.Errorf("the archive passes %d bytes", int64(maxArchiveBytes))
	}
	if errors.Is(err, io.EOF) {
		if found := hex.EncodeToString(a.digest.Sum(nil)); found != a.wanted {
			return n, fmt.Errorf("the download holds %s, and %s is pinned", found, a.wanted)
		}
	}
	return n, err
}

// progressWriter reports to onProgress how many bytes have been written, of a total
// that is negative when unknown.
type progressWriter struct {
	total      int64
	count      int64
	onProgress func(count, total int64)
}

func (w *progressWriter) Write(chunk []byte) (int, error) {
	w.count += int64(len(chunk))
	if w.onProgress != nil {
		w.onProgress(w.count, w.total)
	}
	return len(chunk), nil
}
