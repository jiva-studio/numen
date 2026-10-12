package recognition

import (
	"image"
	"image/color"
	"testing"
)

// Planes returns three planar color channels normalized between zero and one.
func TestPlanesDimensions(t *testing.T) {
	img := documentImage(200, 300)
	got := getPagePlanes(img)
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
	out1 := l.getPlanes(img1)
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
	out2 := l.getPlanes(img2)
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
		_ = l.getPlanes(img)
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

// The page is scaled to 800 by 800 and written one plane a colour, each
// channel a fraction of 255, row after row.
func TestPlanesHoldEachChannelOfEachPixelInPlace(t *testing.T) {
	const side = 800
	corners := []color.RGBA{
		{R: 255, G: 0, B: 128, A: 255},
		{R: 0, G: 255, B: 64, A: 255},
		{R: 64, G: 128, B: 255, A: 255},
		{R: 200, G: 100, B: 50, A: 255},
	}
	page := image.NewRGBA(image.Rect(0, 0, 400, 400))
	for y := 0; y < 400; y++ {
		for x := 0; x < 400; x++ {
			quarter := 0
			if x >= 200 {
				quarter++
			}
			if y >= 200 {
				quarter += 2
			}
			page.SetRGBA(x, y, corners[quarter])
		}
	}

	got := getPagePlanes(page)
	if len(got) != 3*side*side {
		t.Fatalf("planes length = %d, want %d", len(got), 3*side*side)
	}
	for y := 5; y < side; y += 37 {
		for x := 5; x < side; x += 41 {
			if x > 395 && x < 405 || y > 395 && y < 405 {
				continue
			}
			quarter := 0
			if x >= side/2 {
				quarter++
			}
			if y >= side/2 {
				quarter += 2
			}
			want := corners[quarter]
			at := y*side + x
			for plane, channel := range []uint8{want.R, want.G, want.B} {
				wantValue := float32(channel) / 255
				gotValue := got[plane*side*side+at]
				if d := gotValue - wantValue; d > 0.0005 || d < -0.0005 {
					t.Fatalf("plane %d at (%d, %d) = %f, want %f", plane, x, y, gotValue, wantValue)
				}
			}
		}
	}
}
