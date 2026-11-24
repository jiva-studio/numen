package kitpdf

import (
	"strings"
	"testing"
)

type recordedRect struct{ x, y, w, h float64 }

type recordingSink struct {
	fills int
	rects []recordedRect
}

func (r *recordingSink) SetFillColor(_, _, _ int) { r.fills++ }

func (r *recordingSink) Rect(x, y, w, h float64, _ string) {
	r.rects = append(r.rects, recordedRect{x: x, y: y, w: w, h: h})
}

func getLongText() string {
	words := make([]string, 33)
	for i := range words {
		words[i] = strings.Repeat(string(rune('a'+i%26)), 8)
	}
	return getSheetText(words)
}

func TestGetSheetText_JoinsTheWordsWithSingleSpaces(t *testing.T) {
	t.Parallel()
	if got := getSheetText([]string{"alpha", "beta", "gamma"}); got != "alpha beta gamma" {
		t.Fatalf("the sheet text is %q", got)
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
	drawQR(sink, matrix, left, top)
	if sink.fills != 1 {
		t.Errorf("the fill color is set %d times, want 1", sink.fills)
	}
	if len(sink.rects) != runs {
		t.Errorf("%d rectangles are drawn, want %d", len(sink.rects), runs)
	}
	for _, r := range sink.rects {
		if r.x < left || r.y < top || r.x+r.w > left+qrSide+1e-9 || r.y+r.h > top+qrSide+1e-9 {
			t.Fatalf("the rectangle %+v is outside the square", r)
		}
	}
}
