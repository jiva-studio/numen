package onnxruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// downloadTimeout is how long the archive has to arrive.
	downloadTimeout = 15 * time.Minute

	// maxArchiveBytes is the most bytes an archive may hold.
	maxArchiveBytes = 1 << 30
)

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

// verifyFile returns an error when the file at name does not hold the sha256 sum.
func verifyFile(name, sum string) error {
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
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if found := hex.EncodeToString(hash.Sum(nil)); found != sum {
		return fmt.Errorf("%s holds %s, and %s is pinned", name, found, sum)
	}
	return nil
}

// download writes the archive at the address to at. The bytes go through a file
// beside at and are put in place only when they hold the pinned sum.
func download(ctx context.Context, client *http.Client, r Release, pos string, onProgress func(count, total int64)) error {
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
	defer answer.Body.Close()
	if answer.StatusCode != http.StatusOK {
		return fmt.Errorf("fetching %s: %s", r.Address, answer.Status)
	}

	file, err := os.CreateTemp(filepath.Dir(pos), filepath.Base(pos)+".part-*")
	if err != nil {
		return err
	}
	part := file.Name()
	hash := sha256.New()
	copied, err := io.Copy(io.MultiWriter(file, hash, &progressWriter{total: answer.ContentLength, onProgress: onProgress}), io.LimitReader(answer.Body, maxArchiveBytes+1))
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err == nil && copied > maxArchiveBytes {
		err = fmt.Errorf("the archive passes %d bytes", int64(maxArchiveBytes))
	}
	if found := hex.EncodeToString(hash.Sum(nil)); err == nil && found != r.Sum {
		err = fmt.Errorf("the download holds %s, and %s is pinned", found, r.Sum)
	}
	if err != nil {
		os.Remove(part)
		return fmt.Errorf("fetching %s: %w", r.Address, err)
	}
	if err := os.Rename(part, pos); err != nil {
		os.Remove(part)
		return err
	}
	return nil
}

// progressWriter tells onProgress how many bytes have been written, of a total
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
