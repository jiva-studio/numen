package engine

import (
	"errors"
	"image"
	"slices"
	"strings"
	"testing"
	"unicode"

	read "github.com/getcharzp/go-ocr"
)

// upperBox and lowerBox are the two lines of one region, as the detector boxes them for every recogniser.
func upperBox() [4]int { return [4]int{0, 0, 300, 40} }

func lowerBox() [4]int { return [4]int{0, 60, 300, 100} }

func readingOf(lines ...read.RecResult) func(image.Image) ([]read.RecResult, error) {
	return func(image.Image) ([]read.RecResult, error) { return lines, nil }
}

func failingWith(err error) func(image.Image) ([]read.RecResult, error) {
	return func(image.Image) ([]read.RecResult, error) { return nil, err }
}

// latinReading is a Latin recogniser over a Russian word and an English one.
func latinReading() func(image.Image) ([]read.RecResult, error) {
	return readingOf(
		read.RecResult{Box: upperBox(), Text: "BHMMAHME", Score: 0.62},
		read.RecResult{Box: lowerBox(), Text: "Hello", Score: 0.97},
	)
}

// cyrillicReading is a Cyrillic recogniser over the same two lines.
func cyrillicReading() func(image.Image) ([]read.RecResult, error) {
	return readingOf(
		read.RecResult{Box: upperBox(), Text: "ВНИМАНИЕ", Score: 0.95},
		read.RecResult{Box: lowerBox(), Text: "Нello", Score: 0.71},
	)
}

func recogniseOnePart(t *testing.T, reads ...func(image.Image) ([]read.RecResult, error)) ([]Block, error) {
	t.Helper()
	page := image.NewRGBA(image.Rect(0, 0, 600, 400))
	r := NewTestRecogniser(partsOf(image.Rect(20, 20, 580, 380)), reads, Model{})
	return r.Recognise(t.Context(), page)
}

func readText(t *testing.T, blocks []Block, err error) string {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("the page came back as %d blocks: %+v", len(blocks), blocks)
	}
	return blocks[0].Text
}

func TestEachLineKeepsTheReadingOfTheMostConfidentRecogniser(t *testing.T) {
	blocks, err := recogniseOnePart(t, latinReading(), cyrillicReading())
	if got := readText(t, blocks, err); got != "ВНИМАНИЕ Hello" {
		t.Errorf("the part reads %q, want %q", got, "ВНИМАНИЕ Hello")
	}
}

func TestALineReadAsConfidentlyByTwoRecognisersKeepsTheFirstReading(t *testing.T) {
	first := readingOf(
		read.RecResult{Box: upperBox(), Text: "Hello", Score: 0.9},
		read.RecResult{Box: lowerBox(), Text: "world", Score: 0.8},
	)
	second := readingOf(
		read.RecResult{Box: upperBox(), Text: "Нello", Score: 0.9},
		read.RecResult{Box: lowerBox(), Text: "wоrld", Score: 0.8},
	)
	blocks, err := recogniseOnePart(t, first, second)
	if got := readText(t, blocks, err); got != "Hello world" {
		t.Errorf("the part reads %q, want %q", got, "Hello world")
	}
}

func TestRecognisersThatBoxTheLinesDifferentlyKeepTheFirstReading(t *testing.T) {
	for name, second := range map[string]func(image.Image) ([]read.RecResult, error){
		"other boxes": readingOf(
			read.RecResult{Box: [4]int{0, 0, 150, 40}, Text: "ВНИМАНИЕ", Score: 0.95},
			read.RecResult{Box: [4]int{0, 60, 150, 100}, Text: "Нello", Score: 0.99},
		),
		"fewer lines": readingOf(
			read.RecResult{Box: upperBox(), Text: "ВНИМАНИЕ", Score: 0.95},
		),
	} {
		t.Run(name, func(t *testing.T) {
			blocks, err := recogniseOnePart(t, latinReading(), second)
			if got := readText(t, blocks, err); got != "BHMMAHME Hello" {
				t.Errorf("the part reads %q, want the first reading %q", got, "BHMMAHME Hello")
			}
		})
	}
}

func TestAPartIsReadByTheOtherRecognisersWhenOneFails(t *testing.T) {
	broken := errors.New("the session is gone")

	blocks, err := recogniseOnePart(t, failingWith(broken), cyrillicReading())
	if got := readText(t, blocks, err); got != "ВНИМАНИЕ Нello" {
		t.Errorf("with the first recogniser failing the part reads %q, want %q", got, "ВНИМАНИЕ Нello")
	}

	blocks, err = recogniseOnePart(t, latinReading(), failingWith(broken))
	if got := readText(t, blocks, err); got != "BHMMAHME Hello" {
		t.Errorf("with the second recogniser failing the part reads %q, want %q", got, "BHMMAHME Hello")
	}
}

func TestAPageEveryRecogniserFailsOnIsAFailure(t *testing.T) {
	broken := errors.New("the session is gone")
	if _, err := recogniseOnePart(t, failingWith(broken), failingWith(broken)); !errors.Is(err, broken) {
		t.Errorf("a page every recogniser failed on answered with %v", err)
	}
}

// readLine is one recogniser's reading of a page that holds one line.
func readLine(text string, score float32) func(image.Image) ([]read.RecResult, error) {
	return readingOf(read.RecResult{Box: upperBox(), Text: text, Score: score})
}

// The four readings of each line are those of multilingual, in its order: Latin, Cyrillic, Greek,
// PP-OCRv6 tiny. The Greek and tiny models read Latin, and they read Cyrillic as Latin, Greek or
// Chinese look-alikes as confidently as the Cyrillic model reads it right, or more. The Cyrillic
// model reads a Latin word of a Russian line as Cyrillic or as a word of both scripts.
func TestEachLineKeepsTheReadingInTheScriptOfItsLetters(t *testing.T) {
	for _, line := range []struct {
		name  string
		reads []func(image.Image) ([]read.RecResult, error)
		// words are returned as they are printed, and no letter of foreign is returned.
		words   []string
		foreign []*unicode.RangeTable
	}{
		{
			name: "a Russian line of letters Latin also has",
			reads: []func(image.Image) ([]read.RecResult, error){
				readLine("κ κ   pe", 0.781),
				readLine("сорок рек и сухое море", 0.977),
				readLine("copok pek i cyxoe mope", 0.990),
				readLine("copok pek n cyxoe Mope", 0.920),
			},
			words:   []string{"сорок", "рек", "и", "сухое", "море"},
			foreign: []*unicode.RangeTable{unicode.Latin, unicode.Greek},
		},
		{
			name: "a Russian line with a Latin word",
			reads: []func(image.Image) ([]read.RecResult, error){
				readLine("Nporpamma Numen yntaet PDF", 0.700),
				readLine("Программа Нитен читает PDF", 0.955),
				readLine("ΠporpaMMa Numen 4uτaet PDF", 0.975),
				readLine("NporpaMMa Numen 4uTaet PDF", 0.930),
			},
			words:   []string{"Программа", "читает", "PDF"},
			foreign: []*unicode.RangeTable{unicode.Greek},
		},
		{
			name: "a Russian line with a Latin word and a version",
			reads: []func(image.Image) ([]read.RecResult, error){
				readLine("Bepcna Python 3.12 Bblwna oceHblo", 0.720),
				readLine("Версия Руthоn 3.12 вышла осенью", 0.950),
				readLine("Bepcna Python 3.12 Bbl山na oceHb1o", 0.960),
				readLine("Bepcna Python 3.12 Bbl山na oceHb1o", 0.965),
			},
			words:   []string{"Версия", "3.12", "вышла", "осенью"},
			foreign: []*unicode.RangeTable{unicode.Han},
		},
		{
			name: "an English line",
			reads: []func(image.Image) ([]read.RecResult, error){
				readLine("TAKE HOME COMET", 0.950),
				readLine("ТАКЕ НОМЕ СОМЕТ", 0.930),
				readLine("TAKE HOME COMET", 0.940),
				readLine("TAKE HOME COMET", 0.945),
			},
			words:   []string{"TAKE", "HOME", "COMET"},
			foreign: []*unicode.RangeTable{unicode.Cyrillic},
		},
		{
			name: "an English line the Cyrillic model reads with look-alikes",
			reads: []func(image.Image) ([]read.RecResult, error){
				readLine("Caution: high voltage", 0.960),
				readLine("Сaution: high vоltage", 0.990),
				readLine("Caution: high voltage", 0.950),
				readLine("Caution: high voltage", 0.955),
			},
			words:   []string{"Caution:", "high", "voltage"},
			foreign: []*unicode.RangeTable{unicode.Cyrillic},
		},
		{
			name: "a Greek line",
			reads: []func(image.Image) ([]read.RecResult, error){
				readLine("Ta(eec kai uuptiec", 0.800),
				readLine("Гaзeec кai мypтиec", 0.850),
				readLine("Γαζέες καὶ μυρτιὲς", 0.960),
				readLine("Ta3eec kai uuptiec", 0.700),
			},
			words:   []string{"Γαζέες", "καὶ", "μυρτιὲς"},
			foreign: []*unicode.RangeTable{unicode.Cyrillic, unicode.Latin},
		},
	} {
		t.Run(line.name, func(t *testing.T) {
			blocks, err := recogniseOnePart(t, line.reads...)
			got := readText(t, blocks, err)
			for _, word := range line.words {
				if !slices.Contains(strings.Fields(got), word) {
					t.Errorf("the line reads %q, without %q", got, word)
				}
			}
			for _, script := range line.foreign {
				if hasLetterOf(got, script) {
					t.Errorf("the line reads %q, with letters of another script", got)
				}
			}
		})
	}
}
