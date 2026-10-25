package engine

import (
	"context"
	"image"
)

// MockText is what the mock recogniser reads on every page.
const MockText = "Mock recognised text. Regenerative braking recovers kinetic energy, " +
	"a flywheel keeps it in a rotor spinning in a vacuum, " +
	"and pumped storage lifts water to an upper reservoir."

// MockRecogniser reads MockText on every page as one block, in a box across the top of the page. It
// loads no model.
type MockRecogniser struct{}

// GetRecognitionModel names the mock, drawn at the resolution of a screen.
func (MockRecogniser) GetRecognitionModel() Model {
	return Model{Layout: "mock", Recogniser: "mock", DPI: 72, From: "mock"}
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
		Label: "text",
		Text:  MockText,
		Boxes: []Box{{Rect: rect, Span: ByteSpan{To: len(MockText)}}},
	}}, nil
}

// Close releases nothing.
func (MockRecogniser) Close() error { return nil }
