package textversion_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// riverPages are three segments of invented Russian text. Their whole text is 53 bytes: the second
// segment begins at byte 20 and the third at byte 33; every letter is two bytes.
var riverPages = []domain.TextSegment{
	{Text: "Река течёт ", Heading: "Река", Selector: `{"page":1}`},
	{Text: "к морю. ", Selector: `{"page":2}`},
	{Text: "Ветер дует.", Heading: "Ветер", Selector: `{"page":3}`},
}

const riverSize = 53

// newRiverScene is the scene whose reader gives the scan the text of riverPages.
func newRiverScene(t *testing.T) *scene {
	t.Helper()
	s := newScene(t)
	s.reader.make = func(string, domain.VersionText, bool) []domain.TextSegment { return slices.Clone(riverPages) }
	s.requestText(t)
	return s
}

func TestGetTextRangeReturnsTheSegmentsLimitedToTheRange(t *testing.T) {
	for name, tc := range map[string]struct {
		start, length int
		wantStart     int
		want          []domain.TextSegment
	}{
		"one whole segment": {start: 20, length: 13, wantStart: 20, want: []domain.TextSegment{riverPages[1]}},
		"across two segments": {start: 25, length: 10, wantStart: 25, want: []domain.TextSegment{
			{Text: "орю. ", Selector: `{"page":2}`},
			{Text: "В", Heading: "Ветер", Selector: `{"page":3}`},
		}},
		"to the end with length zero": {start: 33, length: 0, wantStart: 33, want: []domain.TextSegment{riverPages[2]}},
		"the whole text":              {start: 0, length: 0, wantStart: 0, want: riverPages},
	} {
		t.Run(name, func(t *testing.T) {
			s := newRiverScene(t)

			got, err := s.versions.GetTextRange(t.Context(), s.vault, scanPath, s.getReaderVersion(), tc.start, tc.length)
			if err != nil {
				t.Fatal(err)
			}

			if got.Start != tc.wantStart || got.Size != riverSize || !slices.Equal(got.Segments, tc.want) {
				t.Errorf("the range from %d of %d bytes is %+v, want start %d, size %d and %+v",
					tc.start, tc.length, got, tc.wantStart, riverSize, tc.want)
			}
		})
	}
}

func TestGetTextRangeMovesAStartInsideACharacterToTheNextCharacter(t *testing.T) {
	s := newRiverScene(t)

	// Byte 26 is the second byte of "о" in "морю".
	got, err := s.versions.GetTextRange(t.Context(), s.vault, scanPath, s.getReaderVersion(), 26, 0)
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.TextSegment{{Text: "рю. ", Selector: `{"page":2}`}, riverPages[2]}
	if got.Start != 27 || got.Size != riverSize || !slices.Equal(got.Segments, want) {
		t.Errorf("the range from byte 26 is %+v, want start 27, size %d and %+v", got, riverSize, want)
	}
}

func TestGetTextRangeEndsBeforeACharacterTheLengthSplits(t *testing.T) {
	s := newRiverScene(t)

	// Three bytes from the start end inside "е", the second letter.
	got, err := s.versions.GetTextRange(t.Context(), s.vault, scanPath, s.getReaderVersion(), 0, 3)
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.TextSegment{{Text: "Р", Heading: "Река", Selector: `{"page":1}`}}
	if got.Start != 0 || got.Size != riverSize || !slices.Equal(got.Segments, want) {
		t.Errorf("the range of 3 bytes is %+v, want start 0, size %d and %+v", got, riverSize, want)
	}
}
