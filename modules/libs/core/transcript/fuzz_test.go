package transcript_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// vttSeeds are the shapes a transcript arrives in: what the transcriber writes,
// a cue named on the line above its timing, cue settings after the second
// timing, a note, a timing in minutes and one in hours, carriage returns, a
// line somebody was still writing, and bytes that are no transcript at all.
var vttSeeds = []string{
	"WEBVTT\n\n00:00:00.000 --> 00:00:02.500\nthe first thing said\n\n" +
		"00:00:02.500 --> 00:00:05.000\nand the second\n",
	"WEBVTT\n\ncue-1\n00:00:00.000 --> 00:00:02.500\nsaid\n",
	"WEBVTT\n\n00:00:00.000 --> 00:00:02.500 line:0 position:20%\nsaid\n",
	"WEBVTT\n\nNOTE this file was written by hand\n\n00:01.000 --> 00:02.000\nsaid\n",
	"WEBVTT\r\n\r\n00:00:00.000 --> 00:00:01.000\r\nsaid\r\n",
	"WEBVTT\n\n00:00:00.000 -->\nsaid\n",
	"WEBVTT\n\n00:00:02.500 --> 00:00:00.000\nbackwards\n",
	"WEBVTT\n\n99:59:59.999 --> 99:59:59.999\nsaid\n",
	"WEBVTT\n",
	"",
	"\x00\xff\xfe not a transcript at all",
}

// Every cue a transcript is read as stands at its own words: the offset it
// carries is where its text is to be found in the words the transcript reads
// as, and the cues are in the order they were spoken.
//
// A transcript arrives from the transcriber and may be edited by hand, so the
// bytes are a stranger's.
func FuzzParse(f *testing.F) {
	for _, seed := range vttSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		said, cues := transcript.Parse([]byte(raw))
		at := -1
		for _, cue := range cues {
			if cue.Offset < 0 || cue.Offset+len(cue.Text) > len(said) {
				t.Fatalf("a cue of %d bytes stands at %d in %d bytes of words",
					len(cue.Text), cue.Offset, len(said))
			}
			if got := said[cue.Offset : cue.Offset+len(cue.Text)]; got != cue.Text {
				t.Fatalf("a cue reading %q stands at %q", cue.Text, got)
			}
			if cue.Offset <= at {
				t.Fatalf("a cue at %d follows one at %d", cue.Offset, at)
			}
			at = cue.Offset
			if strings.TrimSpace(cue.Text) != cue.Text || cue.Text == "" {
				t.Fatalf("a cue carries %q, which is not something said", cue.Text)
			}
		}
	})
}

// The cues come back as they went in. A cue carrying no words is not written,
// and neither is one whose words are a silence in the middle of it: a blank
// line ends a cue in the format, so a cue holding one is two blocks and not one
// cue, and it is not written as one.
//
// A transcript is WebVTT, and the artifact is what a person opens.
func FuzzMarshal(f *testing.F) {
	f.Add("the first thing said", "and the second", 0, 2500, 5000)
	f.Add(" ", "said", 0, 0, 1)
	f.Add("said\nover two lines", "-->", 3599999, 3600000, 3600001)
	f.Add("NOTE", "WEBVTT", 0, 1, 2)

	f.Fuzz(func(t *testing.T, first, second string, a, b, c int) {
		times := []int{a, b, c}
		for _, ms := range times {
			if ms < 0 || ms >= 100*60*60*1000 {
				return
			}
		}
		// The format ends a line one way: a carriage return in a cue's words is
		// the reader's to drop, so a cue carrying one comes back as itself with
		// the endings settled and is no fixed point of the pair.
		if strings.ContainsRune(first+second, '\r') {
			return
		}
		want := []transcript.Cue{
			{Text: strings.TrimSpace(first), From: times[0], To: times[1]},
			{Text: strings.TrimSpace(second), From: times[1], To: times[2]},
		}
		var written []transcript.Cue
		for _, cue := range want {
			// A cue whose words hold a blank line is two blocks of the format
			// and comes back as something else. Nothing writes one: a cue is
			// one stretch of speech.
			if cue.Text == "" || strings.Contains(cue.Text, "\n\n") {
				return
			}
			written = append(written, cue)
		}

		_, got := transcript.Parse(transcript.Marshal(written))
		if len(got) != len(written) {
			t.Fatalf("%d cues written came back as %d", len(written), len(got))
		}
		for at, cue := range got {
			if cue.Text != written[at].Text ||
				cue.From != written[at].From || cue.To != written[at].To {
				t.Fatalf("a cue %+v came back as %+v", written[at], cue)
			}
		}
	})
}
