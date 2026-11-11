package numbered

import (
	"strings"
	"testing"
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

// A batch without a digest is the lines alone.
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
