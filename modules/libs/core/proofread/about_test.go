package proofread_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/transcript"
	"github.com/jiva-studio/numen/modules/libs/core/proofread"
)

// makeCues is a transcript of the words, one cue to each of them.
func makeCues(said ...string) []transcript.Cue {
	out := make([]transcript.Cue, 0, len(said))
	for at, text := range said {
		out = append(out, transcript.Cue{Text: text, From: at * 1000, To: at*1000 + 800})
	}
	return out
}

// getListedWords is the words a digest lists, and nothing for a digest listing
// none.
func getListedWords(t *testing.T, about string) string {
	t.Helper()
	for _, line := range strings.Split(about, "\n") {
		if said, listed := strings.CutPrefix(line, "Words recurring through it, as the machine transcribed them: "); listed {
			return said
		}
	}
	return ""
}

// A digest says how the speech opens and which of its words recur.
func TestADigestSaysWhatTheRecordingHolds(t *testing.T) {
	about := proofread.Describe(makeCues(
		"The assembly at Mithila heard Ganaka.",
		"The teacher listened. Then Ganaka spoke of Mithila",
		"as a city nobody had named before him.",
	))

	if !strings.Contains(about, "The speech opens: The assembly at Mithila heard Ganaka.") {
		t.Errorf("the digest says %q", about)
	}
	if said := getListedWords(t, about); said != "Mithila, Ganaka" {
		t.Errorf("the digest lists %q", said)
	}
}

// A word said once is not a word that recurs.
func TestADigestListsOnlyWhatRecurs(t *testing.T) {
	about := proofread.Describe(makeCues("The teacher named Ganaka once.", "Nothing else was said of him."))
	if said := getListedWords(t, about); said != "" {
		t.Errorf("the digest lists %q", said)
	}
}

// A word standing capitalised only where a sentence opens is a sentence's first
// word and not a name, however often it is said.
func TestAWordOpeningEverySentenceIsNotAName(t *testing.T) {
	about := proofread.Describe(makeCues("Ganaka spoke. Ganaka listened. Ganaka left."))
	if said := getListedWords(t, about); said != "" {
		t.Errorf("the digest lists %q", said)
	}
}

// A shortening ends no sentence, so the name it stands before is a name.
func TestANameBehindAShorteningIsStillAName(t *testing.T) {
	about := proofread.Describe(makeCues("Mr. Ganaka arrived. Mr. Ganaka left."))
	if said := getListedWords(t, about); said != "Ganaka" {
		t.Errorf("the digest lists %q", said)
	}
}

// A speaker talking about himself is not a recording full of one name.
func TestTheWordAPersonCallsHimselfIsNotAName(t *testing.T) {
	about := proofread.Describe(makeCues(
		"So I was saying that I think Ganaka is here.",
		"And I told him that I would go. I'm sure I've said so.",
	))
	if said := getListedWords(t, about); said != "" {
		t.Errorf("the digest lists %q", said)
	}
}

// The opening is as much of the speech as the digest quotes, and says it was
// cut short.
func TestTheOpeningIsCutShort(t *testing.T) {
	var said []string
	for at := range 200 {
		said = append(said, "word"+strconv.Itoa(at))
	}
	about := proofread.Describe(makeCues(strings.Join(said, " ")))

	if !strings.Contains(about, "word0 word1 ") || strings.Contains(about, "word199") {
		t.Errorf("the digest says %q", about)
	}
	if !strings.Contains(about, " …\n") {
		t.Errorf("the digest does not say the opening was cut short: %q", about)
	}
}

// A transcript saying nothing is described as nothing.
func TestATranscriptSayingNothingIsDescribedAsNothing(t *testing.T) {
	if about := proofread.Describe(makeCues("", "")); about != "" {
		t.Errorf("the digest says %q", about)
	}
	if about := proofread.Describe(nil); about != "" {
		t.Errorf("the digest says %q", about)
	}
}

// The digest stands before the first line of the question, and the lines are
// asked about as they always were.
func TestTheDigestStandsBeforeTheFirstLine(t *testing.T) {
	batch := proofread.Batch{Number: 0, IsJoinable: true, Context: "The speech opens: Ganaka spoke.", Lines: []proofread.Line{
		{Number: 0, Text: "Ganaka spoke"},
		{Number: 1, Text: "to the assembly."},
	}}

	asked := proofread.Ask(batch)
	if !strings.HasPrefix(asked, "The speech opens: Ganaka spoke.\n\n") {
		t.Fatalf("the question is %q", asked)
	}
	if !strings.HasSuffix(asked, proofread.Opens+"0"+proofread.Closes+"Ganaka spoke "+
		proofread.Opens+"1"+proofread.Closes+"to the assembly.") {
		t.Errorf("the question is %q", asked)
	}
}

// A batch nothing was said about is the lines alone.
func TestABatchWithNoDigestIsTheLinesAlone(t *testing.T) {
	batch := proofread.Batch{Number: 0, Lines: []proofread.Line{{Number: 0, Text: "Ganaka spoke"}}}
	if asked := proofread.Ask(batch); asked != proofread.Opens+"0"+proofread.Closes+"Ganaka spoke" {
		t.Errorf("the question is %q", asked)
	}
}
