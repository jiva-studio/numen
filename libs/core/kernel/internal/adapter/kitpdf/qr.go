package kitpdf

import (
	"fmt"
	"strings"

	"rsc.io/qr"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

const (
	qrSide      = 56.0
	qrQuietSize = 4
)

type rectSink interface {
	SetFillColor(r, g, b int)
	Rect(x, y, w, h float64, styleStr string)
}

// getSheetText is the text of the code on a sheet: a versioned link of the kit, the sheet and its words, nothing else.
func getSheetText(kitID string, sheet identitydomain.Sheet) string {
	return fmt.Sprintf("numen://recover?v=1&kit=%s&sheet=%d&words=%s", kitID, sheet.Number, strings.Join(sheet.Words, "+"))
}

func getQRMatrix(text string) ([][]bool, error) {
	code, err := qr.Encode(text, qr.M)
	if err != nil {
		return nil, fmt.Errorf("kitpdf: encode the QR code: %w", err)
	}
	matrix := make([][]bool, code.Size)
	for y := range matrix {
		matrix[y] = make([]bool, code.Size)
		for x := range matrix[y] {
			matrix[y][x] = code.Black(x, y)
		}
	}
	return matrix, nil
}

func drawQR(sink rectSink, matrix [][]bool, left, top float64) {
	module := qrSide / float64(len(matrix)+2*qrQuietSize)
	sink.SetFillColor(0, 0, 0)
	for y, row := range matrix {
		for x := 0; x < len(row); x++ {
			if !row[x] {
				continue
			}
			end := x
			for end+1 < len(row) && row[end+1] {
				end++
			}
			sink.Rect(
				left+float64(x+qrQuietSize)*module,
				top+float64(y+qrQuietSize)*module,
				float64(end-x+1)*module, module, "F",
			)
			x = end
		}
	}
}
