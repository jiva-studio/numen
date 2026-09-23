package download

import (
	"testing"
)

// What a site publishes for a video a machine listened to: one event to a
// stretch of speech, a blank between two of them, and the tail of a stretch
// drawn again while the next is being said.
const machineWritten = `{"events":[
  {"tStartMs":0,"dDurationMs":1000,"segs":[{"utf8":"\n"}]},
  {"tStartMs":1500,"dDurationMs":4000,"segs":[{"utf8":"what"},{"utf8":" was"},{"utf8":" said"}]},
  {"tStartMs":4200,"dDurationMs":100,"aAppend":1,"segs":[{"utf8":"\n"}]},
  {"tStartMs":4200,"dDurationMs":4900,"segs":[{"utf8":"what  was\nsaid next"}]}
]}`

// The words of a video stand once, whatever a player was going to draw. A site
// that scrolls two lines at a time says one stretch of speech once, and a cue
// that stood twice would put the words into the index twice.
func TestWhatWasSaidStandsOnce(t *testing.T) {
	cues, err := parseCues([]byte(machineWritten))
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 {
		t.Fatalf("the words are %d cues: %+v", len(cues), cues)
	}
	if cues[0].Text != "what was said" || cues[1].Text != "what was said next" {
		t.Errorf("the words read %q and %q", cues[0].Text, cues[1].Text)
	}
}

// A cue ends where the next begins, so the moment on the player belongs to one
// stretch of speech.
func TestACueEndsWhereTheNextBegins(t *testing.T) {
	cues, err := parseCues([]byte(machineWritten))
	if err != nil {
		t.Fatal(err)
	}
	for i, one := range cues {
		if one.To < one.From {
			t.Errorf("cue %d runs from %d to %d", i, one.From, one.To)
		}
		if i+1 < len(cues) && one.To > cues[i+1].From {
			t.Errorf("cue %d ends at %d, and the next begins at %d", i, one.To, cues[i+1].From)
		}
	}
	if cues[0].From != 1500 || cues[0].To != 4200 {
		t.Errorf("the first cue runs from %d to %d", cues[0].From, cues[0].To)
	}
}

// A person's own captions arrive as they were published: one event to a cue,
// nothing overlapping and nothing repeated.
func TestCaptionsAPersonPublished(t *testing.T) {
	cues, err := parseCues([]byte(
		`{"events":[{"tStartMs":0,"dDurationMs":2000,"segs":[{"utf8":"The first line."}]},` +
			`{"tStartMs":2000,"dDurationMs":3000,"segs":[{"utf8":"The second."}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(cues) != 2 || cues[0].To != 2000 || cues[1].From != 2000 {
		t.Errorf("the cues are %+v", cues)
	}
}

// Bytes that are not what they claim are the file's fault and not the run's.
func TestCaptionsThatWillNotRead(t *testing.T) {
	if _, err := parseCues([]byte("this is not what a site publishes")); err == nil {
		t.Error("anything at all was read as captions")
	}
	cues, err := parseCues([]byte(`{"events":[]}`))
	if err != nil || len(cues) != 0 {
		t.Errorf("a video with nothing said in it is %+v (%v)", cues, err)
	}
}
