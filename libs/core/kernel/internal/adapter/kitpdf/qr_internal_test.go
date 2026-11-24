package kitpdf

import (
	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
	"strings"
	"testing"
)

type recordedRect struct{ x, y, w, h float64 }

type recordingSink struct {
	rects []recordedRect
}

func (r *recordingSink) fill(x, y, w, h float64) {
	r.rects = append(r.rects, recordedRect{x: x, y: y, w: w, h: h})
}

func getLongText() string {
	words := make([]string, 33)
	for i := range words {
		words[i] = strings.Repeat(string(rune('a'+i%26)), 8)
	}
	return getSheetText("C0DE", identitydomain.Sheet{Number: 2, Words: words})
}

func TestGetSheetText_IsAVersionedLinkOfTheKitTheSheetAndItsWords(t *testing.T) {
	t.Parallel()
	sheet := identitydomain.Sheet{Number: 3, Words: []string{"alpha", "beta", "gamma"}}
	want := "numen://recover?v=1&kit=C0DE&sheet=3&words=alpha+beta+gamma"
	if got := getSheetText("C0DE", sheet); got != want {
		t.Fatalf("the sheet text is %q, want %q", got, want)
	}
}

func TestGetQRMatrix_EncodesTheLongestSheetText(t *testing.T) {
	t.Parallel()
	matrix, err := getQRMatrix(getLongText())
	if err != nil {
		t.Fatal(err)
	}
	if len(matrix) < 53 || (len(matrix)-17)%4 != 0 {
		t.Fatalf("the matrix side is %d modules, want a QR version of 9 or above", len(matrix))
	}
	for y, row := range matrix {
		if len(row) != len(matrix) {
			t.Fatalf("row %d has %d modules, want %d", y, len(row), len(matrix))
		}
	}
}

func TestDrawQR_FillsEachRunOfBlackModulesInsideTheSquare(t *testing.T) {
	t.Parallel()
	matrix, err := getQRMatrix(getLongText())
	if err != nil {
		t.Fatal(err)
	}
	runs := 0
	for _, row := range matrix {
		for x, black := range row {
			if black && (x == 0 || !row[x-1]) {
				runs++
			}
		}
	}
	sink := &recordingSink{}
	const left, top = 150.0, 209.0
	drawQR(sink.fill, matrix, left, top)
	if len(sink.rects) != runs {
		t.Errorf("%d rectangles are drawn, want %d", len(sink.rects), runs)
	}
	for _, r := range sink.rects {
		if r.x < left || r.y < top || r.x+r.w > left+qrSide+1e-9 || r.y+r.h > top+qrSide+1e-9 {
			t.Fatalf("the rectangle %+v is outside the square", r)
		}
	}
}

func TestGetQuietOffset_IsTheBorderBeforeTheFirstModule(t *testing.T) {
	t.Parallel()
	matrix, err := getQRMatrix(getLongText())
	if err != nil {
		t.Fatal(err)
	}
	sink := &recordingSink{}
	drawQR(sink.fill, matrix, 0, 0)
	minX := sink.rects[0].x
	for _, r := range sink.rects {
		minX = min(minX, r.x)
	}
	if got := getQuietOffset(matrix); minX < got-1e-9 || minX > got+1e-9 {
		t.Errorf("the first module starts at %v, the offset is %v", minX, got)
	}
}
