package cloud

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	syncclient "github.com/jiva-studio/numen/libs/shared/syncprotocol/client"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

const (
	partURLBatch  = 50
	partURLMargin = 5 * time.Minute
)

// Uploader sends a body to the sync server in one request or in parts, the way the server wants it for its size.
type Uploader struct {
	cloud *Cloud
	now   func() time.Time

	mu   sync.Mutex
	urls map[string]*syncv1.PartUpload
}

// NewUploader returns the uploader over the server. now reads the clock the expiry of a part URL is judged by.
func NewUploader(cloud *Cloud, now func() time.Time) *Uploader {
	return &Uploader{cloud: cloud, now: now, mu: sync.Mutex{}, urls: map[string]*syncv1.PartUpload{}}
}

// Start reserves the blob of a body of the stored size.
func (u *Uploader) Start(ctx context.Context, vault []byte, storedSize int64) (transfer.Upload, error) {
	if storedSize <= syncprotocol.MaxBlobSize {
		reserved, err := u.cloud.ReserveBlobs(ctx, vault, []uint64{uint64(storedSize)})
		if err != nil {
			return transfer.Upload{}, err
		}
		if len(reserved) != 1 {
			return transfer.Upload{}, fmt.Errorf("reserve a blob: the server made %d reservations for one", len(reserved))
		}
		return transfer.Upload{Vault: vault, URL: reserved[0].GetUrl(), Blob: reserved[0].GetBlobId(), PartSize: storedSize, PartCount: 1, Done: nil}, nil
	}
	partSize := syncprotocol.GetPartSize(storedSize)
	reserved, err := call(ctx, u.cloud, func() (*syncv1.ReserveMultipartBlobResponse, error) {
		return u.cloud.blobs.ReserveMultipartBlob(ctx, vault, uint64(storedSize), uint64(partSize))
	})
	if err != nil {
		return transfer.Upload{}, err
	}
	return transfer.Upload{
		Vault: vault, URL: "", Blob: reserved.GetBlobId(), PartSize: int64(reserved.GetPartSize()), PartCount: int(reserved.GetPartCount()), Done: nil,
	}, nil
}

// Resume asks the server which parts of the blob it holds. A blob the server does not know is transfer.ErrUploadGone.
func (u *Uploader) Resume(ctx context.Context, vault, blob []byte) (transfer.Upload, error) {
	parts, err := call(ctx, u.cloud, func() ([]*syncv1.PartInfo, error) { return u.cloud.blobs.ListParts(ctx, vault, blob) })
	if errors.Is(err, syncprotocol.ErrNoVault) {
		return transfer.Upload{}, transfer.ErrUploadGone
	}
	if err != nil {
		return transfer.Upload{}, err
	}
	done := make([]int, len(parts))
	for i, part := range parts {
		done[i] = int(part.GetPartNumber())
	}
	return transfer.Upload{Vault: vault, URL: "", Blob: blob, PartSize: 0, PartCount: 0, Done: done}, nil
}

func (u *Uploader) PutPart(ctx context.Context, up transfer.Upload, number int, body []byte) error {
	if up.URL != "" {
		return syncclient.PutBlob(ctx, u.cloud.storage, up.URL, body)
	}
	url, err := u.getPartURL(ctx, up, number)
	if err != nil {
		return err
	}
	return syncclient.PutPart(ctx, u.cloud.storage, url, body)
}

func (u *Uploader) getPartURL(ctx context.Context, up transfer.Upload, number int) (string, error) {
	key := fmt.Sprintf("%x/%d", up.Blob, number)
	u.mu.Lock()
	defer u.mu.Unlock()
	if found, ok := u.urls[key]; ok && found.GetExpiresAt()-u.now().Unix() > int64(partURLMargin/time.Second) {
		return found.GetUrl(), nil
	}
	last := min(up.PartCount, number+partURLBatch-1)
	if number < 1 || last >= math.MaxUint32 {
		return "", fmt.Errorf("part %d of %d is not a part number of a blob", number, up.PartCount)
	}
	numbers := make([]uint32, 0, partURLBatch)
	for next := uint32(number); next <= uint32(last); next++ {
		numbers = append(numbers, next)
	}
	uploads, err := call(ctx, u.cloud, func() ([]*syncv1.PartUpload, error) {
		return u.cloud.blobs.GetPartUrls(ctx, up.Vault, up.Blob, numbers)
	})
	if err != nil {
		return "", err
	}
	for _, part := range uploads {
		u.urls[fmt.Sprintf("%x/%d", up.Blob, part.GetPartNumber())] = part
	}
	found, ok := u.urls[key]
	if !ok {
		return "", fmt.Errorf("the server gave no URL for part %d", number)
	}
	return found.GetUrl(), nil
}

// Finish joins the parts into the blob. A body sent in one request needs nothing.
func (u *Uploader) Finish(ctx context.Context, up transfer.Upload) error {
	if up.URL != "" {
		return nil
	}
	_, err := call(ctx, u.cloud, func() (struct{}, error) {
		return struct{}{}, u.cloud.blobs.CompleteMultipartBlob(ctx, up.Vault, up.Blob)
	})
	return err
}

// Abort drops the blob the upload reserved.
func (u *Uploader) Abort(ctx context.Context, up transfer.Upload) error {
	if up.URL != "" {
		return nil
	}
	_, err := call(ctx, u.cloud, func() (struct{}, error) {
		return struct{}{}, u.cloud.blobs.AbortMultipartBlob(ctx, up.Vault, up.Blob)
	})
	return err
}

// Blobs reads blobs from the sync server whole or from a byte on.
type Blobs struct{ cloud *Cloud }

func NewBlobs(cloud *Cloud) *Blobs { return &Blobs{cloud: cloud} }

func (b *Blobs) getURL(ctx context.Context, vault, blob []byte) (string, error) {
	downloads, err := b.cloud.GetBlobUrls(ctx, vault, [][]byte{blob})
	if err != nil {
		return "", err
	}
	if len(downloads) != 1 || !downloads[0].GetFound() {
		return "", transfer.ErrBlobMissing
	}
	return downloads[0].GetUrl(), nil
}

// GetBlob downloads a blob of up to syncprotocol.MaxBlobSize bytes.
func (b *Blobs) GetBlob(ctx context.Context, vault, blob []byte) ([]byte, error) {
	url, err := b.getURL(ctx, vault, blob)
	if err != nil {
		return nil, err
	}
	return b.cloud.GetBlob(ctx, url)
}

func (b *Blobs) OpenBlob(ctx context.Context, vault, blob []byte, from int64) (io.ReadCloser, error) {
	url, err := b.getURL(ctx, vault, blob)
	if err != nil {
		return nil, err
	}
	return syncclient.GetBlobRange(ctx, b.cloud.storage, url, from)
}
