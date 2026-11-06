package engine

import (
	"errors"
	"image"
	"testing"

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
