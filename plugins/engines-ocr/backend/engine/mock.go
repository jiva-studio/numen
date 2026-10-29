package engine

import (
	"context"
	"image"
)

// mockName names the mock layout model, recogniser and origin.
const mockName = "mock"

// MockText is what the mock recogniser reads on every page.
const MockText = "Mock recognised text. Regenerative braking recovers kinetic energy, " +
	"a flywheel keeps it in a rotor spinning in a vacuum, " +
	"and pumped storage lifts water to an upper reservoir."

// MockRecogniser reads MockText on every page as one block, in a box across the top of the page. It
// loads no model.
type MockRecogniser struct{}

// GetRecognitionModel names the mock, drawn at the resolution of a screen.
func (MockRecogniser) GetRecognitionModel() Model {
	return Model{Layout: mockName, Recogniser: mockName, DPI: 72, From: mockName}
}

// Recognise reads MockText in a box across the top tenth of the page.
func (MockRecogniser) Recognise(ctx context.Context, page image.Image) ([]Block, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bounds := page.Bounds()
	rect := image.Rect(
		bounds.Min.X+bounds.Dx()/10, bounds.Min.Y+bounds.Dy()/10,
		bounds.Max.X-bounds.Dx()/10, bounds.Min.Y+bounds.Dy()/5,
	)
	return []Block{{
		Label:     labelText,
		Text:      MockText,
		IsHeading: false,
		Depth:     0,
		Boxes:     []Box{{Rect: rect, Range: ByteRange{Start: 0, End: len(MockText)}}},
	}}, nil
}

// Close releases nothing.
func (MockRecogniser) Close() error { return nil }
