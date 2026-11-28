// Package kitpdf writes a recovery kit as a PDF of three A4 pages, one sheet to a page, and saves it
// in the file the person chooses.
package kitpdf

import (
	"bytes"
	_ "embed"
	"fmt"
	"strconv"

	"github.com/go-pdf/fpdf"

	identitydomain "github.com/jiva-studio/numen/libs/core/kernel/domain/identity"
)

const (
	pageWidth  = 210.0
	margin     = 20.0
	right      = pageWidth - margin
	contentW   = right - margin
	gridRows   = 11
	gridRowH   = 8.0
	gridColumn = contentW / 3
	gridGap    = 5.0
	boxGap     = 4.0
	boxPad     = 4.0
	boxHeight  = 35.0
	stepIndent = 5.0
	pointToMM  = 25.4 / 72
	logoName   = "logo"
	logoSize   = 12.0
	logoGap    = 3.0
	logoTop    = 20.7
)

// logoPNG is the application icon, the same 256 px picture the desktop build ships.
//
//go:embed logo.png
var logoPNG []byte

type color int

const (
	ink    color = 0x000000
	hushed color = 0x666666
	rule   color = 0xBBBBBB
	shade  color = 0xF0F0F0
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
	pdf.RegisterImageOptionsReader(logoName, getLogoOptions(), bytes.NewReader(logoPNG))
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

// segment is a piece of flowing text; a bold one names a menu or stresses a phrase.
type segment struct {
	text   string
	isBold bool
}

// flow is a block of wrapped text between two x positions.
type flow struct {
	left, right, top, lineH float64
	size                    float64
	color                   color
}

type way struct {
	title string
	steps [][]segment
}

func getMenuStep(before, menu string) []segment {
	return []segment{{text: before, isBold: false}, {text: menu, isBold: true}, {text: ".", isBold: false}}
}

func getWays() []way {
	open := getMenuStep("In Numen open ", "Settings > Identity")
	return []way{
		{title: "Restore by scanning", steps: [][]segment{
			open,
			getMenuStep("Choose ", "Restore > Recovery kit: scan"),
			{{text: "Scan the QR code of this sheet, then of another.", isBold: false}},
		}},
		{title: "Restore by typing", steps: [][]segment{
			open,
			getMenuStep("Choose ", "Restore > Recovery kit: type"),
			{{text: "Type the 33 words of this sheet, then of another.", isBold: false}},
		}},
	}
}

func (p sheetPage) draw(sheet identitydomain.Sheet, matrix [][]bool) {
	p.pdf.AddPage()
	p.drawHeader(fmt.Sprintf("Sheet %d of %d", sheet.Number, len(p.kit.Sheets)))
	p.drawLead()
	p.drawWays()
	p.drawQR(sheet.Number, matrix)
	p.drawWords(sheet)
	p.drawFooter()
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

func (p sheetPage) drawLine(x1, x2, y, width float64, c color) {
	p.pdf.SetDrawColor(c.getRed(), c.getGreen(), c.getBlue())
	p.pdf.SetLineWidth(width)
	p.pdf.Line(x1, y, x2, y)
}

func (p sheetPage) writeFlow(f flow, segments []segment) float64 {
	p.pdf.SetLeftMargin(f.left)
	p.pdf.SetRightMargin(pageWidth - f.right)
	p.pdf.SetXY(f.left, f.top)
	for _, s := range segments {
		style := ""
		if s.isBold {
			style = "B"
		}
		p.setFont("Helvetica", style, f.size, f.color)
		p.pdf.Write(f.lineH, p.toCodePage(s.text))
	}
	p.pdf.SetLeftMargin(margin)
	p.pdf.SetRightMargin(margin)
	return p.pdf.GetY() + f.lineH
}

func getBaseline(top, lineH, size float64) float64 { return top + lineH/2 + 0.3*size*pointToMM }

func (p sheetPage) drawHeader(of string) {
	p.setFont("Helvetica", "B", 26, ink)
	p.pdf.ImageOptions(logoName, margin, logoTop, logoSize, logoSize, false, getLogoOptions(), 0, "")
	p.write(margin+logoSize+logoGap, 30, "Numen")
	p.setFont("Helvetica", "B", 13, ink)
	p.writeRight(right, 25.5, "Recovery kit")
	p.setFont("Helvetica", "", 10, hushed)
	p.writeRight(right, 31, of)
	p.drawLine(margin, right, 34, 0.5, ink)
}

func (p sheetPage) drawLead() {
	p.writeFlow(flow{left: margin, right: right, top: 40, lineH: 5.4, size: 11, color: ink}, []segment{
		{text: "This sheet helps you get your ", isBold: false},
		{text: "identity", isBold: true},
		{text: " back if you lose your computer. ", isBold: false},
		{text: "Any two of the three sheets are enough.", isBold: true},
		{text: " One sheet alone cannot be used by anyone. Keep each sheet in a different place.", isBold: false},
	})
}

func (p sheetPage) drawWays() {
	const top = 62.0
	width := (contentW - boxGap) / 2
	p.pdf.SetFillColor(shade.getRed(), shade.getGreen(), shade.getBlue())
	for i, w := range getWays() {
		left := margin + float64(i)*(width+boxGap)
		p.pdf.RoundedRect(left, top, width, boxHeight, 1.5, "1234", "F")
		p.setFont("Helvetica", "B", 10.5, ink)
		p.write(left+boxPad, top+boxPad+2.6, w.title)
		p.drawSteps(w.steps, left+boxPad, left+width-boxPad, top+boxPad+5)
	}
}

func (p sheetPage) drawSteps(steps [][]segment, left, edge, top float64) {
	const size, lineH = 9.0, 4.4
	for i, step := range steps {
		p.setFont("Helvetica", "", size, ink)
		p.write(left, getBaseline(top, lineH, size), strconv.Itoa(i+1)+".")
		top = p.writeFlow(flow{left: left + stepIndent, right: edge, top: top, lineH: lineH, size: size, color: ink}, step) + 1.2
	}
}

func (p sheetPage) drawQR(number int, matrix [][]bool) {
	const top = 102.0
	offset := getQuietOffset(matrix)
	p.pdf.SetFillColor(0, 0, 0)
	drawQR(func(x, y, w, h float64) { p.pdf.Rect(x, y, w, h, "F") }, matrix, margin-offset, top-offset)
	left := margin + contentW/3
	p.setFont("Helvetica", "B", 8, hushed)
	p.write(left, getBaseline(top, 4, 8), fmt.Sprintf("QR CODE OF SHEET %d", number))
	p.writeFlow(flow{left: left, right: right, top: top + 5, lineH: 4.6, size: 9.5, color: ink}, []segment{
		{text: "It holds the same 33 words as the list below. Both ways give the same result.", isBold: false},
	})
}

func (p sheetPage) drawWords(sheet identitydomain.Sheet) {
	const captionY, gridTop = 152.0, 155.0
	p.setFont("Helvetica", "B", 8, hushed)
	p.write(margin, captionY, fmt.Sprintf("THE 33 WORDS OF SHEET %d", sheet.Number))
	for i, word := range sheet.Words {
		x := margin + float64(i/gridRows)*gridColumn
		top := gridTop + float64(i%gridRows)*gridRowH
		y := getBaseline(top, gridRowH, 11)
		p.setFont("Helvetica", "", 9, hushed)
		p.writeRight(x+7, y, strconv.Itoa(i+1))
		p.setFont("Courier", "", 11, ink)
		p.write(x+10, y, word)
		p.drawLine(x, x+gridColumn-gridGap, top+gridRowH, 0.1, rule)
	}
}

func (p sheetPage) drawFooter() {
	const ruleY, textY = 270.0, 275.0
	p.drawLine(margin, right, ruleY, 0.1, rule)
	p.setFont("Helvetica", "", 8, hushed)
	p.write(margin, textY, "numen.md")
	p.writeRight(right, textY, "Do not photograph or share. Anyone with two sheets can restore your identity.")
}

func getLogoOptions() fpdf.ImageOptions {
	return fpdf.ImageOptions{ImageType: "png", ReadDpi: false, AllowNegativePosition: false}
}
