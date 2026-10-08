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

func TestLayoutPlanesReuse(t *testing.T) {
	l := &Layout{}
	img1 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img1.SetRGBA(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}
	out1 := l.planes(img1)
	plane := layoutSide * layoutSide
	for i := 0; i < plane; i++ {
		if out1[i] < 0.99 {
			t.Fatalf("red plane[%d] = %f, want ~1.0", i, out1[i])
		}
		if out1[plane+i] > 0.01 || out1[2*plane+i] > 0.01 {
			t.Fatalf("green/blue plane[%d] = %f / %f, want 0", i, out1[plane+i], out1[2*plane+i])
		}
	}

	img2 := image.NewRGBA(image.Rect(0, 0, 100, 100))
	for y := 0; y < 100; y++ {
		for x := 0; x < 100; x++ {
			img2.SetRGBA(x, y, color.RGBA{R: 0, G: 0, B: 255, A: 255})
		}
	}
	out2 := l.planes(img2)
	for i := 0; i < plane; i++ {
		if out2[2*plane+i] < 0.99 {
			t.Fatalf("blue plane[%d] = %f, want ~1.0", i, out2[2*plane+i])
		}
		if out2[i] > 0.01 || out2[plane+i] > 0.01 {
			t.Fatalf("red/green plane[%d] = %f / %f, want 0", i, out2[i], out2[plane+i])
		}
	}
}

func BenchmarkPlanes(b *testing.B) {
	img := documentImage(2000, 3000)
	l := &Layout{}
	b.ResetTimer()
	b.ReportAllocs()
	for b.Loop() {
		_ = l.planes(img)
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
