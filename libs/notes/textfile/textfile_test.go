package textfile_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/textfile"
)

func TestEncodeWritesOneSegmentPerLine(t *testing.T) {
	raw, err := textfile.Encode([]domain.TextSegment{
		{Text: "Good morning.\n", Selector: `{"t":[0,1.5]}`},
		{Text: "Chapter", Heading: "One"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"text":"Good morning.\n","selector":{"t":[0,1.5]}}` + "\n" + `{"text":"Chapter","heading":"One"}` + "\n"
	if string(raw) != want {
		t.Fatalf("encoded\n%s\nwant\n%s", raw, want)
	}
}

func TestDecodeReadsWhatEncodeWrote(t *testing.T) {
	segments := []domain.TextSegment{
		{Text: "first ", Heading: "Intro", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: "second"},
	}
	raw, err := textfile.Encode(segments)
	if err != nil {
		t.Fatal(err)
	}
	got, err := textfile.Decode(append(raw, '\n'))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, segments) {
		t.Fatalf("decoded %+v, want %+v", got, segments)
	}
}

func TestEncodeRejectsASelectorThatIsNotJSON(t *testing.T) {
	_, err := textfile.Encode([]domain.TextSegment{{Text: "a", Selector: "page 3"}})
	if !errors.Is(err, textfile.ErrInvalidSelector) {
		t.Fatalf("got %v, want ErrInvalidSelector", err)
	}
}

func TestDecodeRejectsALineThatIsNotJSON(t *testing.T) {
	if _, err := textfile.Decode([]byte("{\"text\":\"a\"}\nWEBVTT\n")); err == nil {
		t.Fatal("a WebVTT line was read as a segment")
	}
}
