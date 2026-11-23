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
	gridTop    = 70.0
	gridRows   = 11
	gridRowH   = 9.0
	gridPad    = 2.5
	gridColumn = (right - margin) / 3
)

type color struct{ r, g, b int }

var (
	ink    = color{24, 24, 27}
	hushed = color{113, 113, 122}
	rule   = color{212, 212, 216}
	shade  = color{244, 244, 245}
)

// Render writes the kit as a PDF: one A4 page for each sheet.
func Render(kit identitydomain.Kit) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetTitle("Numen recovery kit", true)
	pdf.SetCreator("Numen", true)
	pdf.SetCreationDate(kit.CreatedAt)
	pdf.SetModificationDate(kit.CreatedAt)
	page := sheetPage{pdf: pdf, text: pdf.UnicodeTranslatorFromDescriptor(""), kit: kit}
	for _, sheet := range kit.Sheets {
		page.draw(sheet)
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, fmt.Errorf("kitpdf: write the kit: %w", err)
	}
	return out.Bytes(), nil
}

type sheetPage struct {
	pdf  *fpdf.Fpdf
	text func(string) string
	kit  identitydomain.Kit
}

func (p sheetPage) draw(sheet identitydomain.Sheet) {
	p.pdf.AddPage()
	of := fmt.Sprintf("Sheet %d of %d", sheet.Number, len(p.kit.Sheets))
	p.drawHeader(of)
	p.drawFacts()
	p.drawWords(sheet.Words, of)
	p.drawRule()
	p.drawRestore()
	p.drawKeptAt()
	p.drawFooter(of)
}

func (p sheetPage) setFont(family, style string, size float64, c color) {
	p.pdf.SetFont(family, style, size)
	p.pdf.SetTextColor(c.r, c.g, c.b)
}

func (p sheetPage) write(x, y float64, s string) { p.pdf.Text(x, y, p.text(s)) }

func (p sheetPage) writeRight(x, y float64, s string) {
	s = p.text(s)
	p.pdf.Text(x-p.pdf.GetStringWidth(s), y, s)
}

func (p sheetPage) drawLine(y float64) {
	p.pdf.SetDrawColor(rule.r, rule.g, rule.b)
	p.pdf.SetLineWidth(0.2)
	p.pdf.Line(margin, y, right, y)
}

func (p sheetPage) drawHeader(of string) {
	p.setFont("Helvetica", "B", 22, ink)
	p.write(margin, 26, "Numen recovery kit")
	p.setFont("Helvetica", "", 11, hushed)
	p.writeRight(right, 26, of)
	p.setFont("Helvetica", "", 9, hushed)
	p.write(margin, 32, "A paper key to your Numen account. Keep it offline, in a safe place.")
	p.drawLine(37)
}

func (p sheetPage) drawFacts() {
	const label, value = 44.0, 50.5
	p.setFont("Helvetica", "B", 7, hushed)
	p.write(margin, label, "ACCOUNT FINGERPRINT")
	p.write(130, label, "KIT")
	p.write(152, label, "CREATED")
	p.setFont("Courier", "B", 11, ink)
	p.write(margin, value, p.kit.AccountFingerprint)
	p.write(130, value, p.kit.ID)
	p.setFont("Helvetica", "", 10, ink)
	p.write(152, value, p.kit.CreatedAt.Format("2 January 2006"))
	p.drawLine(56)
}

func (p sheetPage) drawWords(words []string, of string) {
	p.setFont("Helvetica", "B", 11, ink)
	p.write(margin, 65, "Recovery words")
	p.setFont("Helvetica", "", 9, hushed)
	p.writeRight(right, 65, fmt.Sprintf("%s · %d words", of, len(words)))
	height := gridRows*gridRowH + 2*gridPad
	p.pdf.SetFillColor(shade.r, shade.g, shade.b)
	for row := 1; row < gridRows; row += 2 {
		p.pdf.Rect(margin, gridTop+gridPad+float64(row)*gridRowH, right-margin, gridRowH, "F")
	}
	p.pdf.SetDrawColor(rule.r, rule.g, rule.b)
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

func (p sheetPage) drawRule() {
	const top = 189.0
	p.pdf.SetFillColor(shade.r, shade.g, shade.b)
	p.pdf.RoundedRect(margin, top, right-margin, 17, 2, "1234", "F")
	p.setFont("Helvetica", "B", 11, ink)
	p.write(margin+5, top+7, "Any 2 of 3 sheets restore your account. One sheet alone reveals nothing.")
	p.setFont("Helvetica", "", 9, hushed)
	p.write(margin+5, top+12.5, "Keep each sheet in a different place, so one lost or found sheet is never enough.")
}

func (p sheetPage) drawRestore() {
	p.setFont("Helvetica", "B", 11, ink)
	p.write(margin, 219, "To restore your account")
	p.setFont("Helvetica", "", 10, ink)
	p.write(margin, 226, "1.  In Numen, open Settings › Identity and licenses › Restore, then choose Recovery kit.")
	p.write(margin, 232, "2.  Type the words of any two sheets, one sheet in each field, then choose Restore.")
	p.write(margin, 238, "3.  In the field Chain mark, type the mark below. A server that went back in time is refused.")
	p.setFont("Courier", "B", 11, ink)
	p.write(margin+7, 244.5, p.kit.ChainMark)
}

func (p sheetPage) drawKeptAt() {
	const y = 252.0
	p.setFont("Helvetica", "B", 10, ink)
	p.write(margin, y, "Kept at:")
	p.pdf.SetDrawColor(hushed.r, hushed.g, hushed.b)
	p.pdf.SetLineWidth(0.2)
	p.pdf.Line(margin+17, y+0.8, right, y+0.8)
}

func (p sheetPage) drawFooter(of string) {
	p.drawLine(277)
	p.setFont("Helvetica", "", 8, hushed)
	p.write(margin, 282, "Numen cannot restore your account without the 12 recovery words or two sheets of a kit.")
	p.writeRight(right, 282, fmt.Sprintf("Kit %s · %s", p.kit.ID, of))
}
