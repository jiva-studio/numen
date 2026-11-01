package pdf

import (
	"bytes"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/cleanup"
	"math"
	"os"
	"testing"
)

func TestRenderPageDefaultDPIRendersAt300DPI(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := newWorkers(t).OpenReader(&trackingCloser{Reader: bytes.NewReader(raw)})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, scan)

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
	scan, err := newWorkers(t).OpenReader(&trackingCloser{Reader: bytes.NewReader(raw)})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, scan)

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

func TestRenderPageAtWidthDrawsThePageThatWide(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := newWorkers(t).OpenReader(&trackingCloser{Reader: bytes.NewReader(raw)})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, scan)
	width, height, err := scan.GetPageSize(0)
	if err != nil {
		t.Fatal(err)
	}

	drawn, err := scan.RenderPageAtWidth(0, 300, 1<<24)
	if err != nil {
		t.Fatal(err)
	}

	wantDy := int(math.Round(300 * height / width))
	if drawn.Bounds().Dx() != 300 || drawn.Bounds().Dy() != wantDy {
		t.Errorf("bounds = %v, want 300 by %d", drawn.Bounds(), wantDy)
	}
	if _, err := scan.RenderPageAtWidth(0, 300, 1000); err == nil {
		t.Error("a page over the most pixels was drawn")
	}
	if _, err := scan.RenderPageAtWidth(0, 0, 1<<24); err == nil {
		t.Error("a page no pixels across was drawn")
	}
}

func TestGetFittingWidthKeepsThePageWithinTheMostPixels(t *testing.T) {
	raw, err := os.ReadFile("testdata/tiny.pdf")
	if err != nil {
		t.Fatal(err)
	}
	scan, err := newWorkers(t).OpenReader(&trackingCloser{Reader: bytes.NewReader(raw)})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup.Close(t, scan)
	width, height, err := scan.GetPageSize(0)
	if err != nil {
		t.Fatal(err)
	}

	for _, most := range []int{1000, 250_000, 1 << 24} {
		fitting, err := scan.GetFittingWidth(0, 1<<20, most)
		if err != nil {
			t.Fatal(err)
		}
		if pixels := fitting * getHeightAt(fitting, width, height); pixels > most {
			t.Errorf("most=%d: %d wide has %d pixels", most, fitting, pixels)
		}
		if next := fitting + 1; next*getHeightAt(next, width, height) <= most {
			t.Errorf("most=%d: %d wide is not the widest, %d fits", most, fitting, next)
		}
	}
	if fitting, _ := scan.GetFittingWidth(0, 120, 1<<24); fitting != 120 {
		t.Errorf("a width that fits was changed to %d", fitting)
	}
}
