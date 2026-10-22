package pdf

import (
	"bytes"
	"math"
	"os"
	"testing"
)

func TestRenderPageDefaultDPIRendersAt300DPI(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := OpenReader(&trackingCloser{Reader: bytes.NewReader(raw)})
	if err != nil {
		t.Fatal(err)
	}
	defer scan.Close()

	width, height, err := scan.GetPageSize(0)
	if err != nil {
		t.Fatal(err)
	}

	wantDx := int(math.Ceil(width * 300.0 / pointsPerInch))
	wantDy := int(math.Ceil(height * 300.0 / pointsPerInch))

	drawn300, err := scan.RenderPage(0, 300)
	if err != nil {
		t.Fatalf("RenderPage(0, 300) failed: %v", err)
	}
	if drawn300.Bounds().Dx() != wantDx || drawn300.Bounds().Dy() != wantDy {
		t.Fatalf("RenderPage(0, 300) bounds = %v, want (%d, %d)", drawn300.Bounds(), wantDx, wantDy)
	}

	for _, dpi := range []int{0, -1, -72} {
		rendered, err := scan.RenderPage(0, dpi)
		if err != nil {
			t.Fatalf("RenderPage(0, %d) failed: %v", dpi, err)
		}
		if rendered.Bounds() != drawn300.Bounds() {
			t.Errorf("RenderPage(0, %d) bounds = %v, want %v", dpi, rendered.Bounds(), drawn300.Bounds())
		}
		if rendered.Bounds().Dx() != wantDx || rendered.Bounds().Dy() != wantDy {
			t.Errorf("RenderPage(0, %d) = (%d, %d), want (%d, %d)",
				dpi, rendered.Bounds().Dx(), rendered.Bounds().Dy(), wantDx, wantDy)
		}
	}
}

func TestRenderPageScanDefaultDPI(t *testing.T) {
	raw, err := os.ReadFile("testdata/scan.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := OpenReader(&trackingCloser{Reader: bytes.NewReader(raw)})
	if err != nil {
		t.Fatal(err)
	}
	defer scan.Close()

	drawnDefault, err := scan.RenderPage(0, 0)
	if err != nil {
		t.Fatalf("RenderPage(0, 0) failed: %v", err)
	}
	drawn300, err := scan.RenderPage(0, 300)
	if err != nil {
		t.Fatalf("RenderPage(0, 300) failed: %v", err)
	}
	if drawnDefault.Bounds() != drawn300.Bounds() {
		t.Errorf("RenderPage(0, 0) bounds = %v, want 300 DPI bounds %v", drawnDefault.Bounds(), drawn300.Bounds())
	}
	if drawnDefault.Bounds().Dx() != 834 || drawnDefault.Bounds().Dy() != 834 {
		t.Errorf("RenderPage(0, 0) bounds = %v, want 834x834", drawnDefault.Bounds())
	}
}
