package service

import (
	"strconv"
	"strings"
	"testing"
)

// makeLines is a transcript of the words, one line to each of them, numbered by
// place. A line of no words is left in so the numbering shows the gap.
func makeLines(response ...string) []Line {
	out := make([]Line, 0, len(response))
	for item, text := range response {
		out = append(out, Line{Number: item, Text: text})
	}
	return out
}

// getListedWords is the words a digest lists, and nothing for a digest listing
// none.
func getListedWords(t *testing.T, about string) string {
	t.Helper()
	for _, line := range strings.Split(about, "\n") {
		if response, items := strings.CutPrefix(line, "Words recurring through it, as the machine transcribed them: "); items {
			return response
		}
	}
	return ""
}

// A digest says how the speech opens and which of its words recur.
func TestADigestReturnsWhatTheRecordingContains(t *testing.T) {
	about := describeSpeech(makeLines(
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

// A word said once is not a word that recurs.
func TestADigestListsOnlyWhatRecurs(t *testing.T) {
	about := describeSpeech(makeLines("The teacher named Ganaka once.", "Nothing else was said of him."))
	if response := getListedWords(t, about); response != "" {
		t.Errorf("the digest lists %q", response)
	}
}

// A word standing capitalised only where a sentence opens is a sentence's first
// word and not a name, however often it is said.
func TestAWordOpeningEverySentenceIsNotAName(t *testing.T) {
	about := describeSpeech(makeLines("Ganaka spoke. Ganaka listened. Ganaka left."))
	if response := getListedWords(t, about); response != "" {
		t.Errorf("the digest lists %q", response)
	}
}

// A shortening ends no sentence, so the name it stands before is a name.
func TestANamePriorAShorteningIsStillAName(t *testing.T) {
	about := describeSpeech(makeLines("Mr. Ganaka arrived. Mr. Ganaka left."))
	if response := getListedWords(t, about); response != "Ganaka" {
		t.Errorf("the digest lists %q", response)
	}
}

// A speaker talking about himself is not a recording full of one name.
func TestTheWordAPersonCallsHimselfIsNotAName(t *testing.T) {
	about := describeSpeech(makeLines(
		"So I was saying that I think Ganaka is here.",
		"And I told him that I would go. I'm sure I've said so.",
	))
	if response := getListedWords(t, about); response != "" {
		t.Errorf("the digest lists %q", response)
	}
}

// The opening is as much of the speech as the digest quotes, and says it was
// cut short.
func TestTheOpeningIsSplitShort(t *testing.T) {
	var response []string
	for item := range 200 {
		response = append(response, "word"+strconv.Itoa(item))
	}
	about := describeSpeech(makeLines(strings.Join(response, " ")))

	if !strings.Contains(about, "word0 word1 ") || strings.Contains(about, "word199") {
		t.Errorf("the digest says %q", about)
	}
	if !strings.Contains(about, " …\n") {
		t.Errorf("the digest does not say the opening was cut short: %q", about)
	}
}

// A transcript saying nothing is described as nothing.
func TestATranscriptSayingNothingIsDescribedAsNothing(t *testing.T) {
	if about := describeSpeech(makeLines("", "")); about != "" {
		t.Errorf("the digest says %q", about)
	}
	if about := describeSpeech(nil); about != "" {
		t.Errorf("the digest says %q", about)
	}
}
