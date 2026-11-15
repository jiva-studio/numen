package numbered

import (
	"strconv"
	"strings"
	"testing"
)

func makeLines(response ...string) []Line {
	out := make([]Line, 0, len(response))
	for item, text := range response {
		out = append(out, Line{Number: item, Text: text})
	}
	return out
}

func getListedWords(t *testing.T, about string) string {
	t.Helper()
	for line := range strings.SplitSeq(about, "\n") {
		if response, items := strings.CutPrefix(line, "Words recurring through it, as the machine transcribed them: "); items {
			return response
		}
	}
	return ""
}

// A digest reports how the speech opens and which of its words recur.
func TestADigestReturnsWhatTheRecordingContains(t *testing.T) {
	about := DescribeSpeech(makeLines(
		"The assembly at Mithila heard Ganaka.",
		"The teacher listened. Then Ganaka spoke of Mithila",
		"as a city nobody had named before him.",
	))

	if !strings.Contains(about, "The speech opens: The assembly at Mithila heard Ganaka.") {
		t.Errorf("the digest says %q", about)
	}
	if response := getListedWords(t, about); response != "Mithila, Ganaka" {
		t.Errorf("the digest lists %q", response)
	}
}

// A word that occurs once is not a recurring word.
func TestADigestListsOnlyWhatRecurs(t *testing.T) {
	about := DescribeSpeech(makeLines("The teacher named Ganaka once.", "Nothing else was said of him."))
	want := "The speech opens: The teacher named Ganaka once. Nothing else was said of him.\n"
	if about != want {
		t.Errorf("the digest says %q, want %q", about, want)
	}
	if response := getListedWords(t, about); response != "" {
		t.Errorf("the digest lists %q", response)
	}
	withSkippedFirst := DescribeSpeech(makeLines("The Once and Twice and Twice."))
	if response := getListedWords(t, withSkippedFirst); response != "Twice" {
		t.Errorf("the digest lists %q, want Twice", response)
	}
}

// A word capitalized only where a sentence opens is a sentence's first word and not
// a name, however often it occurs.
func TestAWordOpeningEverySentenceIsNotAName(t *testing.T) {
	about := DescribeSpeech(makeLines(
		"Ganaka spoke. Ganaka listened. Ganaka left.",
		"It is so. Ganaka spoke so. Ganaka left.",
		"See Bob. Ganaka and Tom. Ganaka.",
		"The (end.) Alpha (end.) Alpha “end.” Beta “end.” Beta \"end.\" Gamma \"end.\" Gamma 'end.' Delta 'end.' Delta.",
	))
	if response := getListedWords(t, about); response != "" {
		t.Errorf("the digest lists %q", response)
	}
}

// An abbreviation does not end a sentence, so the name following it is a name.
func TestANamePriorAShorteningIsStillAName(t *testing.T) {
	about := DescribeSpeech(makeLines("Mr. Ganaka arrived. Mr. Ganaka left. We saw e.g. Mithila and e.g. Mithila."))
	if response := getListedWords(t, about); response != "Ganaka, Mithila" {
		t.Errorf("the digest lists %q", response)
	}
}

// A speaker talking about himself is not a recording full of one name.
func TestTheWordAPersonCallsHimselfIsNotAName(t *testing.T) {
	about := DescribeSpeech(makeLines(
		"So I was saying that I think Ganaka is here.",
		"And I told him that I would go. I'm sure I've said so and I've done so, I’ll go and I’ll stay.",
	))
	if response := getListedWords(t, about); response != "" {
		t.Errorf("the digest lists %q", response)
	}
	withNames := DescribeSpeech(makeLines("The Om and Om and D'Arc and D'Arc and D’Arc and D’Arc and Ganak\u0301 and Ganak\u0301."))
	if response := getListedWords(t, withNames); response != "Om, D'Arc, D’Arc, Ganak\u0301" {
		t.Errorf("the digest lists %q", response)
	}
}

// The opening is as much of the speech as the digest quotes, and it is marked as
// truncated.
func TestTheOpeningIsSplitShort(t *testing.T) {
	var response []string
	for item := range 61 {
		response = append(response, "word"+strconv.Itoa(item))
	}
	exact := DescribeSpeech(makeLines(strings.Join(response[:60], " ")))
	if !strings.Contains(exact, "word59\n") || strings.Contains(exact, " …") {
		t.Errorf("60-word digest says %q", exact)
	}

	about := DescribeSpeech(makeLines(strings.Join(response, " ")))
	if !strings.Contains(about, "word0 word1 ") || !strings.Contains(about, "word59 …\n") || strings.Contains(about, "word60") {
		t.Errorf("61-word digest says %q", about)
	}
}

func TestTheDigestCapsListedNamesAtTwentyFour(t *testing.T) {
	words := []string{"The"}
	for i := range 25 {
		name := "Name" + string(rune('a'+i/26)) + string(rune('a'+i%26))
		words = append(words, name, name)
	}
	about := DescribeSpeech(makeLines(strings.Join(words, " ")))
	listed := strings.Split(getListedWords(t, about), ", ")
	if len(listed) != 24 || listed[0] != "Nameaa" || listed[23] != "Nameax" {
		t.Errorf("listed %d names: %v", len(listed), listed)
	}
}

func TestATranscriptSayingNothingIsDescribedAsNothing(t *testing.T) {
	if about := DescribeSpeech(makeLines("", "")); about != "" {
		t.Errorf("the digest says %q", about)
	}
	if about := DescribeSpeech(nil); about != "" {
		t.Errorf("the digest says %q", about)
	}
}
