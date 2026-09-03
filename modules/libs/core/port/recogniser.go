package port

import (
	"context"
	"fmt"
	"image"

	"github.com/jiva-studio/numen/modules/libs/core/ocr"
)

// RecognitionModel names what read a page. It is recorded beside what it produced,
// as an EmbeddingModel is recorded with a vector: a text kept beyond the run
// that made it is claimed again by what made it, and anything left out of that
// name is something that can change while the name does not.
type RecognitionModel struct {
	// Layout is the model that divides a page into its parts.
	Layout string
	// Recogniser is the model that reads a line.
	Recogniser string
	// DPI is what the page was read at, which decides what the models saw.
	DPI int

	// From is where the models were loaded from. Two models answering to one
	// name from two places are two models.
	From string
}

// String is the identity as one value, for saying which models are in use.
func (r RecognitionModel) String() string {
	return fmt.Sprintf("%s+%s@%ddpi", r.Layout, r.Recogniser, r.DPI)
}

// Recipe is everything about this recognition that decides what a text is, as
// one value. It is kept beside an artifact so that a person can ask what read
// the text they are looking at.
func (r RecognitionModel) Recipe() string {
	return fmt.Sprintf("%s|%s|%s|%d", r.From, r.Layout, r.Recogniser, r.DPI)
}

// Recogniser turns the image of a page into the text it carries. The core asks
// for one and does not know whether the models run on this machine or a service
// answered.
type Recogniser interface {
	// Recognition is what every page this recogniser reads was read by.
	Recognition() RecognitionModel

	// Recognise is one page: its parts, in the order the page is read, and what
	// each of them says. A page that carries nothing readable is an empty page
	// and not an error.
	Recognise(ctx context.Context, page image.Image) ([]ocr.Block, error)

	// Close releases whatever the models hold.
	Close() error
}
