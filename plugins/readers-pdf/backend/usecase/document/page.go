package document

import (
	"bytes"
	"context"
	"errors"
	"image/jpeg"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// MostPixels is the most pixels one page image has. A page asked for wider is drawn as wide as fits,
// and the window scales the image up.
const MostPixels = 1 << 24

// RenderError is a failure to draw a page of a file that was read.
type RenderError struct {
	Err error
}

func (e *RenderError) Error() string { return e.Err.Error() }

func (e *RenderError) Unwrap() error { return e.Err }

// GaveUpError is the end of a request before its page was drawn.
type GaveUpError struct {
	Err error
}

func (e *GaveUpError) Error() string { return e.Err.Error() }

func (e *GaveUpError) Unwrap() error { return e.Err }

// RenderPage draws the page at index of the file in the vault as a JPEG image pixelWidth pixels
// across, or as wide as fits within MostPixels.
func (r *Reader) RenderPage(ctx context.Context, vault domain.VaultID, path string, index, pixelWidth int) ([]byte, error) {
	raw, _, err := r.files(vault).Read(ctx, path)
	if err != nil {
		return nil, err
	}
	// A window that asked for another width, or left the page, no longer wants this one.
	// gomutants:disable-next-line BRANCH_IF reason="vault.Read already rejects canceled contexts before reaching this check"
	if err := ctx.Err(); err != nil {
		return nil, &GaveUpError{Err: err}
	}
	image, err := r.drawPage(ctx, raw, index, pixelWidth)
	if err != nil {
		return nil, &RenderError{Err: err}
	}
	return image, nil
}

func (r *Reader) drawPage(ctx context.Context, raw []byte, index, pixelWidth int) (encoded []byte, err error) {
	scan, err := r.workers.OpenBytes(raw)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, scan.Close()) }()
	// The wait for a worker can outlast the window's interest in the page.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	fitting, err := scan.GetFittingWidth(index, pixelWidth, MostPixels)
	if err != nil {
		return nil, err
	}
	drawing, err := scan.RenderPageAtWidth(index, fitting, MostPixels)
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, drawing, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
