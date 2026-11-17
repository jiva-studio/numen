// Package ports declares the interfaces the use cases of the plugin share.
package ports

import (
	"context"
	"image"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/scan"
)

// Recognizer reads the blocks of one page drawn as an image.
type Recognizer interface {
	GetRecognitionModel() scan.Model
	Recognize(ctx context.Context, page image.Image) ([]scan.Block, error)
}
