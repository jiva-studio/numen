package engine

import (
	"context"
	"image"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/settings"
)

// mockName names the mock layout model, recognizer and origin.
const mockName = "mock"

// MockText is what the mock recognizer reads on every page.
const MockText = "Mock recognized text. Regenerative braking recovers kinetic energy, " +
	"a flywheel keeps it in a rotor spinning in a vacuum, " +
	"and pumped storage lifts water to an upper reservoir."

// MockRecognizer reads MockText on every page as one block, in a box across the top of the page. It
// loads no model.
type MockRecognizer struct{}

// GetRecognitionModel names the mock, drawn at the resolution of a screen.
func (MockRecognizer) GetRecognitionModel() Model {
	return Model{Layout: mockName, Recognizer: mockName, DPI: 72, From: mockName}
}

// Recognize reads MockText in a box across the top tenth of the page.
func (MockRecognizer) Recognize(ctx context.Context, page image.Image) ([]Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bounds := page.Bounds()
	rect := image.Rect(
		bounds.Min.X+bounds.Dx()/10, bounds.Min.Y+bounds.Dy()/10,
		bounds.Max.X-bounds.Dx()/10, bounds.Min.Y+bounds.Dy()/5,
	)
	return []Block{{
		Label:     settings.LabelText,
		Text:      MockText,
		IsHeading: false,
		Depth:     0,
		Boxes:     []Box{{Rect: rect, Range: ByteRange{Start: 0, End: len(MockText)}}},
	}}, nil
}

// Close releases nothing.
func (MockRecognizer) Close() error { return nil }
