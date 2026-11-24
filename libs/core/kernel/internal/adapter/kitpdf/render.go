// Package kitpdf writes a recovery kit as a PDF of three A4 pages, one sheet to a page, and saves it
// in the file the person chooses.
package kitpdf

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/go-pdf/fpdf"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

const (
	pageWidth  = 210.0
	margin     = 20.0
	right      = pageWidth - margin
	gridTop    = 40.0
	gridRows   = 11
	gridRowH   = 9.0
	gridPad    = 2.5
	gridColumn = (right - margin) / 3
)

type color int

const (
	ink    color = 0x18181B
	hushed color = 0x71717A
	rule   color = 0xD4D4D8
	shade  color = 0xF4F4F5
)

func (c color) getRed() int   { return int(c>>16) & 0xFF }
func (c color) getGreen() int { return int(c>>8) & 0xFF }
func (c color) getBlue() int  { return int(c) & 0xFF }

// Render writes the kit as a PDF: one A4 page for each sheet.
func Render(kit identitydomain.Kit) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle("Numen recovery kit", true)
	pdf.SetCreator("Numen", true)
	pdf.SetCreationDate(kit.CreatedAt)
	pdf.SetModificationDate(kit.CreatedAt)
	page := sheetPage{pdf: pdf, toCodePage: pdf.UnicodeTranslatorFromDescriptor(""), kit: kit}
	for _, sheet := range kit.Sheets {
		matrix, err := getQRMatrix(getSheetText(kit.ID, sheet))
		if err != nil {
			return nil, err
		}
		page.draw(sheet, matrix)
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("kitpdf: write the kit: %w", err)
	}
	return out.Bytes(), nil
}

type sheetPage struct {
	pdf        *fpdf.Fpdf
	toCodePage func(string) string
	kit        identitydomain.Kit
}

func (p sheetPage) draw(sheet identitydomain.Sheet, matrix [][]bool) {
	p.pdf.AddPage()
	p.drawHeader(fmt.Sprintf("Sheet %d of %d", sheet.Number, len(p.kit.Sheets)))
	p.drawWords(sheet.Words)
	p.drawQR(matrix)
}

func (p sheetPage) setFont(family, style string, size float64, c color) {
	p.pdf.SetFont(family, style, size)
	p.pdf.SetTextColor(c.getRed(), c.getGreen(), c.getBlue())
}

func (p sheetPage) write(x, y float64, s string) { p.pdf.Text(x, y, p.toCodePage(s)) }

func (p sheetPage) writeRight(x, y float64, s string) {
	s = p.toCodePage(s)
	p.pdf.Text(x-p.pdf.GetStringWidth(s), y, s)
}

func (p sheetPage) drawLine(y float64) {
	p.pdf.SetDrawColor(rule.getRed(), rule.getGreen(), rule.getBlue())
	p.pdf.SetLineWidth(0.2)
	p.pdf.Line(margin, y, right, y)
}

func (p sheetPage) drawHeader(of string) {
	p.setFont("Helvetica", "B", 22, ink)
	p.write(margin, 26, "Numen recovery kit")
	p.setFont("Helvetica", "", 11, hushed)
	p.writeRight(right, 26, of)
	p.drawLine(32)
}

func (p sheetPage) drawWords(words []string) {
	height := gridRows*gridRowH + 2*gridPad
	p.pdf.SetFillColor(shade.getRed(), shade.getGreen(), shade.getBlue())
	for row := 1; row < gridRows; row += 2 {
		p.pdf.Rect(margin, gridTop+gridPad+float64(row)*gridRowH, right-margin, gridRowH, "F")
	}
	p.pdf.SetDrawColor(rule.getRed(), rule.getGreen(), rule.getBlue())
	p.pdf.SetLineWidth(0.2)
	p.pdf.RoundedRect(margin, gridTop, right-margin, height, 2, "1234", "D")
	for i, word := range words {
		x := margin + float64(i/gridRows)*gridColumn
		y := gridTop + gridPad + (float64(i%gridRows)+0.5)*gridRowH + 1.6
		p.setFont("Helvetica", "", 9, hushed)
		p.writeRight(x+12, y, strconv.Itoa(i+1))
		p.setFont("Courier", "B", 13, ink)
		p.write(x+15, y, word)
	}
}

func (p sheetPage) drawQR(matrix [][]bool) {
	const top = 160.0
	p.pdf.SetFillColor(0, 0, 0)
	drawQR(func(x, y, w, h float64) { p.pdf.Rect(x, y, w, h, "F") }, matrix, margin, top)
	p.setFont("Helvetica", "", 10, ink)
	p.write(margin+qrSide+6, top+qrSide/2-2, "To restore, scan this code in Numen under")
	p.write(margin+qrSide+6, top+qrSide/2+3, "Settings, Restore. Any 2 of 3 sheets are enough.")
}
