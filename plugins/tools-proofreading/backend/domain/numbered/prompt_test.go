package numbered

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// The digest precedes the first line of the question, and the lines follow it
// unchanged.
func TestTheDigestExistsBeforeTheFirstLine(t *testing.T) {
	batch := Batch{Number: 0, IsJoinable: true, Context: "The speech opens: Ganaka spoke.", Lines: []Line{
		{Number: 0, Text: "Ganaka spoke"},
		{Number: 1, Text: "to the assembly."},
	}}

	request := BuildPrompt(batch)
	if !strings.HasPrefix(request, "The speech opens: Ganaka spoke.\n\n") {
		t.Fatalf("the question is %q", request)
	}
	if !strings.HasSuffix(request, Opens+"0"+Closes+"Ganaka spoke "+
		Opens+"1"+Closes+"to the assembly.") {
		t.Errorf("the question is %q", request)
	}
}

func TestABatchWithNoDigestIsTheLinesAlone(t *testing.T) {
	batch := Batch{Number: 0, Lines: []Line{{Number: 0, Text: "Ganaka spoke"}}}
	if request := BuildPrompt(batch); request != Opens+"0"+Closes+"Ganaka spoke" {
		t.Errorf("the question is %q", request)
	}
}

// The two instructions ask for different corrections: one is about what a
// machine read, the other about what it heard.
func TestSpeechIsProofreadForDifferentThingsThanAScan(t *testing.T) {
	if ScanInstruction == SpeechInstruction {
		t.Fatal("a scan and speech are proofread with the same instruction")
	}
	for _, word := range []string{"read off the pages", "diacritics"} {
		if !strings.Contains(ScanInstruction, word) {
			t.Errorf("the scan instruction does not mention %q", word)
		}
		if strings.Contains(SpeechInstruction, word) {
			t.Errorf("the speech instruction mentions %q", word)
		}
	}
	for _, word := range []string{"heard in a recording", "sounds like it", "capitalization", "hesitation", "summarize"} {
		if !strings.Contains(SpeechInstruction, word) {
			t.Errorf("the speech instruction does not mention %q", word)
		}
	}
	// Both are answered the same way.
	for _, one := range []string{ScanInstruction, SpeechInstruction} {
		for _, word := range []string{
			"12|the corrected line",
			"A line you would leave alone is a line you do not answer with.",
			Opens,
			Closes,
		} {
			if !strings.Contains(one, word) {
				t.Errorf("an instruction does not say %q", word)
			}
		}
	}

	// A page's printed lines keep the positions they were printed at. A sentence broken
	// across lines of speech is answered for as one line.
	if !strings.Contains(ScanInstruction, "Nothing moves from one line to\n  another") {
		t.Error("the scan instruction lets a word move between lines")
	}
	if !strings.Contains(SpeechInstruction, "12-14|the corrected sentence") {
		t.Error("the speech instruction does not say how a broken sentence is answered for")
	}
}

func TestReadPagesAndApplyCorrections(t *testing.T) {
	if IsTimed(nil) {
		t.Error("empty segments reported timed")
	}
	if IsTimed([]domain.TextSegment{{Selector: "{bad"}}) {
		t.Error("invalid selector reported timed")
	}
	if IsTimed([]domain.TextSegment{{Selector: `{"t":[1]}`}}) {
		t.Error("single-timestamp selector reported timed")
	}
	if !IsTimed([]domain.TextSegment{{Selector: `{"t":[1,2]}`}}) {
		t.Error("two-timestamp selector not reported timed")
	}
	if sel, err := ReadSelector(""); err != nil || sel.Page != 0 {
		t.Errorf("ReadSelector empty = %+v, %v", sel, err)
	}

	parts := []domain.TextSegment{
		{Text: "  first ", Heading: "H1", Selector: "{bad"},
		{Text: " second\n", Selector: `{"page":1,"boxes":[[0.1,0.2,0.3,0.4]]}`},
		{Text: "third\n", Selector: `{"page":1,"boxes":[[0.5,0.6,0.7,0.8]]}`},
		{Text: "fourth\n", Selector: `{"page":2,"boxes":[[0.9,0.9,1.0,1.0]]}`},
	}
	lines := ReadLines(parts)
	pages := ReadPages(parts, lines)
	if len(pages) != 3 || pages[0].Number != 0 || pages[1].Number != 1 || len(pages[1].Lines) != 2 || pages[2].Number != 2 {
		t.Errorf("ReadPages = %+v", pages)
	}

	applied := ApplyCorrections(parts, []Line{
		{Number: 1, Last: 2, Text: "joined with bad first selector"},
		{Number: 3, Last: 9, Text: "joined across pages"},
	})
	if len(applied) != 2 || applied[0].Selector != "{bad" || applied[0].Text != "  joined with bad first selector\n" {
		t.Errorf("applied[0] = %+v", applied[0])
	}
	if applied[1].Selector != `{"page":1,"boxes":[[0.5,0.6,0.7,0.8]]}` {
		t.Errorf("applied[1] selector = %q", applied[1].Selector)
	}

	joinedWithBadLater := joinSelectors([]domain.TextSegment{
		{Selector: `{"t":[1,2],"page":1,"boxes":[[0.1,0.1,0.2,0.2]]}`},
		{Selector: "{bad"},
		{Selector: `{"t":[2,5],"page":1,"boxes":[[0.3,0.3,0.4,0.4]]}`},
	})
	if joinedWithBadLater != `{"t":[1,5],"page":1,"boxes":[[0.1,0.1,0.2,0.2],[0.3,0.3,0.4,0.4]]}` {
		t.Errorf("joinedWithBadLater = %q", joinedWithBadLater)
	}
}
