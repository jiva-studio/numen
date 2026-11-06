package engine

import (
	"image"
	"image/color"
	"image/draw"
	"os"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// fontPaths are fonts that draw Latin, Cyrillic, polytonic Greek and Chinese in one file.
var fontPaths = []string{
	"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
	"/Library/Fonts/Arial Unicode.ttf",
}

// openFace is the first font of fontPaths at the size of 12 pt print scanned at 300 dpi, the
// resolution the plugin renders a page at.
func openFace(t *testing.T) font.Face {
	t.Helper()
	for _, path := range fontPaths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		parsed, err := opentype.Parse(raw)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 50, DPI: 72, Hinting: font.HintingFull})
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		return face
	}
	t.Fatalf("no font that draws every script is on this machine: looked at %v", fontPaths)
	return nil
}

// drawPage is an A4 page at 300 dpi with the lines printed in black from its upper left margin.
func drawPage(face font.Face, lines ...string) image.Image {
	page := image.NewRGBA(image.Rect(0, 0, 2480, 3508))
	draw.Draw(page, page.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	writer := font.Drawer{Dst: page, Src: image.NewUniform(color.Black), Face: face, Dot: fixed.Point26_6{}}
	for i, line := range lines {
		writer.Dot = fixed.P(300, 400+i*90)
		writer.DrawString(line)
	}
	return page
}

func readPage(t *testing.T, r *Recogniser, page image.Image) string {
	t.Helper()
	blocks, err := r.Recognise(t.Context(), page)
	if err != nil {
		t.Fatal(err)
	}
	texts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		texts = append(texts, block.Text)
	}
	return strings.Join(texts, "\n")
}

func hasLetterOf(text string, script *unicode.RangeTable) bool {
	for _, letter := range text {
		if unicode.Is(script, letter) {
			return true
		}
	}
	return false
}

// openMultilingual opens the four recognisers of multilingual from NUMEN_OCR_DIR, which holds the ONNX
// Runtime and the model files under the names the manifest gives them.
func openMultilingual(t *testing.T) *Recogniser {
	t.Helper()
	dir := os.Getenv("NUMEN_OCR_DIR")
	if dir == "" {
		t.Skip("NUMEN_OCR_DIR names no folder of real models")
	}
	cfg := Defaults()
	cfg.Dir, cfg.ShouldDownload = dir, false
	r, err := Open(t.Context(), NewLibrary(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	want := "rec-pp-ocrv5-latin+rec-pp-ocrv5-cyrillic+rec-pp-ocrv5-greek+rec-pp-ocrv6-tiny"
	if got := r.GetRecognitionModel().Recogniser; got != want {
		t.Errorf("the recognition is recorded as read with %q, want %q", got, want)
	}
	return r
}

func TestMultilingualReadsTheLettersOfEachScript(t *testing.T) {
	r := openMultilingual(t)
	face := openFace(t)
	for _, page := range []struct {
		language string
		line     string
		// Each group is one letter of the script, read as any of its spellings.
		letters [][]string
		// foreign is a script none of whose letters the page holds.
		foreign *unicode.RangeTable
	}{
		{"Russian", "Съешь же ещё этих мягких французских булок", [][]string{{"ж"}, {"ё"}, {"щ"}}, unicode.Latin},
		{"Polish", "Zażółć gęślą jaźń", [][]string{{"ż"}, {"ó"}, {"ł"}, {"ę"}, {"ś"}, {"ź"}, {"ń"}}, unicode.Cyrillic},
		{"Czech", "Příliš žluťoučký kůň", [][]string{{"ř"}, {"ž"}, {"ů"}}, unicode.Cyrillic},
		{"German", "Größe, Übermaß", [][]string{{"ö"}, {"ü"}, {"ß"}}, unicode.Cyrillic},
		{"Greek", "Γαζέες καὶ μυρτιὲς δὲν θὰ βρῶ", [][]string{{"γ"}, {"ζ"}, {"έ", "ε"}, {"ὲ", "έ", "ε"}, {"β"}}, unicode.Latin},
		{"Chinese", "汉字识别是一项有用的技术", [][]string{{"汉"}, {"字"}}, unicode.Cyrillic},
	} {
		t.Run(page.language, func(t *testing.T) {
			text := readPage(t, r, drawPage(face, page.line))
			lower := strings.ToLower(text)
			for _, spellings := range page.letters {
				found := false
				for _, one := range spellings {
					found = found || strings.Contains(lower, one)
				}
				if !found {
					t.Errorf("%q came back as %q, without %s", page.line, text, strings.Join(spellings, " or "))
				}
			}
			if hasLetterOf(text, page.foreign) {
				t.Errorf("%q came back as %q, with letters of another script", page.line, text)
			}
		})
	}
}

func TestMultilingualReadsARussianLineInCyrillicAndAnEnglishLineInLatin(t *testing.T) {
	r := openMultilingual(t)
	text := readPage(t, r, drawPage(openFace(t), "ВНИМАНИЕ: высокое напряжение", "Caution: high voltage"))
	for _, want := range []string{"ВНИМАНИЕ", "напряжение", "Caution", "voltage"} {
		if !strings.Contains(text, want) {
			t.Errorf("the page came back as %q, without %q", text, want)
		}
	}
	if strings.Contains(text, "BHMMAHME") {
		t.Errorf("the Russian line came back in Latin look-alikes: %q", text)
	}
}
