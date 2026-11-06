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

// envFont names a TrueType font that draws Latin, Cyrillic, polytonic Greek and Chinese in one file,
// for a machine that has none of the files getUnicodeFonts lists.
const envFont = "NUMEN_OCR_FONT"

// getUnicodeFonts lists fonts that draw Latin, Cyrillic, polytonic Greek and Chinese in one file: the
// one envFont names, then Arial Unicode where macOS and Microsoft Office install it.
func getUnicodeFonts() []string {
	fonts := []string{
		"/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
		`C:\Windows\Fonts\ARIALUNI.TTF`,
	}
	if path := os.Getenv(envFont); path != "" {
		return append([]string{path}, fonts...)
	}
	return fonts
}

// openFace is the first of paths on this machine at the size of 12 pt print scanned at 300 dpi, the
// resolution the plugin renders a page at.
func openFace(t *testing.T, paths []string) font.Face {
	t.Helper()
	for _, path := range paths {
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
	t.Skipf("no font that draws every script is on this machine: looked at %v; name one in %s", paths, envFont)
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

// openMultilingual opens the four recognisers of multilingual from NUMEN_OCR_DIR, which holds the model
// files under the names the manifest gives them and the ONNX Runtime used when findRuntime finds none.
func openMultilingual(t *testing.T) *Recogniser {
	t.Helper()
	dir := os.Getenv("NUMEN_OCR_DIR")
	if dir == "" {
		t.Skip("NUMEN_OCR_DIR names no folder of real models")
	}
	cfg := Defaults()
	cfg.Dir, cfg.ShouldDownload = dir, false
	// A process loads one ONNX Runtime: a second library fails every session on a tensor type mismatch.
	cfg.Runtime = findRuntime(t)
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
	face := openFace(t, getUnicodeFonts())
	for _, page := range []struct {
		language string
		line     string
		// Each group is one letter of the script, read as any of its spellings.
		letters [][]string
		// foreign is a script none of whose letters the page holds.
		foreign *unicode.RangeTable
	}{
		{"Russian", "Съешь же ещё этих мягких французских булок", [][]string{{"ж"}, {"ё"}, {"щ"}}, unicode.Latin},
		// The PP-OCR recognisers read ź as ż.
		{"Polish", "Zażółć gęślą jaźń", [][]string{{"ż"}, {"ó"}, {"ł"}, {"ę"}, {"ś"}, {"ń"}}, unicode.Cyrillic},
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
	text := readPage(t, r, drawPage(openFace(t, getUnicodeFonts()), "ВНИМАНИЕ: высокое напряжение", "Caution: high voltage"))
	for _, want := range []string{"ВНИМАНИЕ", "напряжение", "Caution", "voltage"} {
		if !strings.Contains(text, want) {
			t.Errorf("the page came back as %q, without %q", text, want)
		}
	}
	if strings.Contains(text, "BHMMAHME") {
		t.Errorf("the Russian line came back in Latin look-alikes: %q", text)
	}
}

// readWords reads a page that prints line and fails for every word that is not returned as it is
// printed and for any letter of the scripts in foreign.
func readWords(t *testing.T, r *Recogniser, face font.Face, line string, words []string, foreign ...*unicode.RangeTable) {
	t.Helper()
	text := readPage(t, r, drawPage(face, line))
	for _, word := range words {
		if !strings.Contains(text, word) {
			t.Errorf("%q came back as %q, without %q", line, text, word)
		}
	}
	for _, script := range foreign {
		if hasLetterOf(text, script) {
			t.Errorf("%q came back as %q, with letters of another script", line, text)
		}
	}
}

func TestMultilingualKeepsTheCyrillicOfRussianLinesWithLatinLetters(t *testing.T) {
	r := openMultilingual(t)
	face := openFace(t, getUnicodeFonts())
	for _, page := range []struct {
		line  string
		words []string
	}{
		{"Программа Numen читает PDF", []string{"Программа", "читает", "PDF"}},
		{"Версия Python 3.12 вышла осенью", []string{"Версия", "3.12", "вышла", "осенью"}},
		// Every letter of this line has a Latin look-alike.
		{"сорок рек и сухое море", []string{"сорок", "рек", "сухое", "море"}},
	} {
		t.Run(page.line, func(t *testing.T) {
			readWords(t, r, face, page.line, page.words, unicode.Greek, unicode.Han)
		})
	}
}

func TestMultilingualKeepsAGreekLineGreekAndAnEnglishLineLatin(t *testing.T) {
	r := openMultilingual(t)
	face := openFace(t, getUnicodeFonts())
	for _, page := range []struct {
		line  string
		words []string
	}{
		{"Τα παιδιά παίζουν στον κήπο", []string{"παιδιά", "παίζουν", "κήπο"}},
		{"TAKE HOME COMET", []string{"TAKE", "HOME", "COMET"}},
		{"Caution: high voltage", []string{"Caution", "high", "voltage"}},
	} {
		t.Run(page.line, func(t *testing.T) {
			readWords(t, r, face, page.line, page.words, unicode.Cyrillic)
		})
	}
}

// The Cyrillic model alone reads these lines as "Программа Нитен читает PDF" and "Версия Руthоn 3.12
// вышла осенью": the Latin word of each comes from another model.
func TestMultilingualKeepsTheLatinWordsOfRussianLines(t *testing.T) {
	r := openMultilingual(t)
	face := openFace(t, getUnicodeFonts())
	for _, page := range []struct {
		line  string
		words []string
	}{
		{"Программа Numen читает PDF", []string{"Программа", "Numen", "читает", "PDF"}},
		{"Версия Python 3.12 вышла осенью", []string{"Версия", "Python", "3.12", "вышла", "осенью"}},
	} {
		t.Run(page.line, func(t *testing.T) {
			readWords(t, r, face, page.line, page.words, unicode.Greek, unicode.Han)
		})
	}
}
