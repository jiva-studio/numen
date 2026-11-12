package hosttest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
)

func TestReadRangeReadsTheBytesFromStartAsTextWithTheFileStat(t *testing.T) {
	v := hosttest.New("Geology")
	v.File("Rocks/Slate.md", "Slate splits into thin sheets.\n")

	got, err := v.ReadRange(t.Context(), "Rocks/Slate.md", 6, 6)
	if err != nil {
		t.Fatal(err)
	}

	if got.Text != "splits" || got.Start != 6 {
		t.Errorf("the range is %+v, want splits from 6", got)
	}
	if stat := statOf(t, v, "Rocks/Slate.md"); got.Stat.Size != stat.Size || !got.Stat.ModTime.Equal(stat.ModTime) {
		t.Errorf("the range has the stat %+v, want the stat of the file, %+v", got.Stat, stat)
	}
}

func TestReadRangeSnapsStartForwardAndEndsOnAWholeCharacter(t *testing.T) {
	v := hosttest.New("Geology")
	// "ab" is bytes 0 and 1, "é" bytes 2 and 3, "cd" bytes 4 and 5.
	v.File("Rocks/Gneiss.txt", "abécd")

	inside, err := v.ReadRange(t.Context(), "Rocks/Gneiss.txt", 3, 10)
	if err != nil {
		t.Fatal(err)
	}
	if inside.Text != "cd" || inside.Start != 4 {
		t.Errorf("a range from the middle of é is %+v, want cd from 4", inside)
	}

	cut, err := v.ReadRange(t.Context(), "Rocks/Gneiss.txt", 0, 3)
	if err != nil {
		t.Fatal(err)
	}
	if cut.Text != "ab" || cut.Start != 0 {
		t.Errorf("a range ending in the middle of é is %+v, want ab from 0", cut)
	}
}

func TestReadRangeOfZeroOrOverMaxReads16000Bytes(t *testing.T) {
	v := hosttest.New("Geology")
	v.File("Rocks/Sandstone.txt", strings.Repeat("s", 20000))

	for _, length := range []int{0, 50000} {
		got, err := v.ReadRange(t.Context(), "Rocks/Sandstone.txt", 0, length)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Text) != 16000 {
			t.Errorf("a range of %d read %d bytes, want 16000", length, len(got.Text))
		}
	}
	tail, err := v.ReadRange(t.Context(), "Rocks/Sandstone.txt", 19990, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail.Text) != 10 || tail.Start != 19990 {
		t.Errorf("a range past the end is %d bytes from %d, want 10 bytes from 19990", len(tail.Text), tail.Start)
	}
}

func TestReadRangeOfBytesThatAreNotUTF8IsNotText(t *testing.T) {
	v := hosttest.New("Geology")
	v.File("Rocks/Core.bin", "CORE\xff\xfe\xfd\xfc")

	_, err := v.ReadRange(t.Context(), "Rocks/Core.bin", 0, 100)

	if !errors.Is(err, host.ErrNotText) {
		t.Errorf("a range of binary bytes returned %v, want ErrNotText", err)
	}
}

func TestReadRangeOfAFolderOrAMissingFileIsNotExist(t *testing.T) {
	v := hosttest.New("Geology")
	v.File("Rocks/Slate.md", "Slate.\n")

	for _, target := range []string{"Rocks", "Rocks/Chalk.md"} {
		if _, err := v.ReadRange(t.Context(), target, 0, 100); !errors.Is(err, host.ErrNotExist) {
			t.Errorf("a range of %s returned %v, want ErrNotExist", target, err)
		}
	}
}

func newRangeTexts(t *testing.T, segments []domain.TextSegment) (*hosttest.Texts, domain.TextVersionID) {
	t.Helper()
	ctx := t.Context()
	file := []byte("%PDF")
	texts := hosttest.NewTexts("v1", "org.example.ocr")
	texts.File("Strata.pdf", file)
	if err := host.WriteText(ctx, texts.GetStore("org.example.ocr"), "k.jsonl", segments); err != nil {
		t.Fatal(err)
	}
	version, err := texts.PutVersion(ctx, host.TextPut{
		Path: "Strata.pdf", Key: "k", Kind: "recognition", InputHash: domain.HashContent(file), File: "k.jsonl",
	})
	if err != nil {
		t.Fatal(err)
	}
	return texts, version.GetID()
}

func getStrataSegments() []domain.TextSegment {
	return []domain.TextSegment{
		{Text: "the ", Selector: `{"page":1}`},
		{Text: "end of ", Selector: `{"page":2}`},
		{Text: "ice", Selector: `{"page":3}`},
	}
}

func TestReadTextRangeReturnsTheSegmentsLimitedToTheRange(t *testing.T) {
	texts, id := newRangeTexts(t, getStrataSegments())

	got, err := texts.ReadTextRange(t.Context(), "Strata.pdf", id, 2, 6)
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.TextSegment{{Text: "e ", Selector: `{"page":1}`}, {Text: "end ", Selector: `{"page":2}`}}
	if !slices.Equal(got.Segments, want) || got.Start != 2 || got.Size != 14 {
		t.Errorf("the range is %+v, want %+v from 2 of a text of 14 bytes", got, want)
	}
}

func TestReadTextRangeOfLengthZeroReadsToTheEnd(t *testing.T) {
	texts, id := newRangeTexts(t, getStrataSegments())

	got, err := texts.ReadTextRange(t.Context(), "Strata.pdf", id, 5, 0)
	if err != nil {
		t.Fatal(err)
	}

	want := []domain.TextSegment{{Text: "nd of ", Selector: `{"page":2}`}, {Text: "ice", Selector: `{"page":3}`}}
	if !slices.Equal(got.Segments, want) || got.Start != 5 || got.Size != 14 {
		t.Errorf("the range is %+v, want %+v from 5 of a text of 14 bytes", got, want)
	}
}

func TestReadTextRangeMovesTheStartForwardToAWholeCharacter(t *testing.T) {
	// "a" is byte 0, "é" bytes 1 and 2, "b" byte 3.
	texts, id := newRangeTexts(t, []domain.TextSegment{{Text: "aé"}, {Text: "b"}})

	got, err := texts.ReadTextRange(t.Context(), "Strata.pdf", id, 2, 0)
	if err != nil {
		t.Fatal(err)
	}

	if want := []domain.TextSegment{{Text: "b"}}; !slices.Equal(got.Segments, want) || got.Start != 3 || got.Size != 4 {
		t.Errorf("a range from the middle of é is %+v, want %+v from 3 of a text of 4 bytes", got, want)
	}
}

func TestReadTextRangeOfAnUnknownVersionIsNotExist(t *testing.T) {
	texts, _ := newRangeTexts(t, getStrataSegments())

	if _, err := texts.ReadTextRange(t.Context(), "Strata.pdf", "org.example.ocr/none", 0, 0); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("a range of an unknown version returned %v, want ErrNotExist", err)
	}
}
