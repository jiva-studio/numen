package recognition

import (
	"image"
	"image/color"
	"testing"
)

// Planes returns three planar color channels normalized between zero and one.
func TestPlanesDimensions(t *testing.T) {
	img := documentImage(200, 300)
	got := planes(img)
	want := 3 * layoutSide * layoutSide
	if len(got) != want {
		t.Fatalf("planes length = %d, want %d", len(got), want)
	}
	for i, v := range got {
		if v < 0 || v > 1 {
			t.Fatalf("planes[%d] = %f, want within [0, 1]", i, v)
		}
	}
}

func BenchmarkPlanes(b *testing.B) {
	img := documentImage(2000, 3000)
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = planes(img)
	}
}

func documentImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8((x*31 + y*17) & 0xff)
			img.SetRGBA(x, y, color.RGBA{R: v, G: v, B: v, A: 255})
		}
	}
	return img
}
