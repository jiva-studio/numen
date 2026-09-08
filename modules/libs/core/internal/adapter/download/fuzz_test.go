package download

import "testing"

// The shapes captions arrive in: an event to a stretch of speech, one that says
// nothing, one appending to the stretch before it, a duration that runs
// backwards, and bytes no site publishes.
var captionSeeds = []string{
	`{"events":[{"tStartMs":0,"dDurationMs":1000,"segs":[{"utf8":"a word"}]}]}`,
	`{"events":[{"tStartMs":0,"dDurationMs":0,"segs":[{"utf8":"\n"}]}]}`,
	`{"events":[{"tStartMs":10,"dDurationMs":-5,"segs":[{"utf8":"a"}]},` +
		`{"tStartMs":1,"dDurationMs":1,"segs":[{"utf8":"b"}]}]}`,
	`{"events":[{"aAppend":1,"segs":[{"utf8":"again"}]}]}`,
	`{"events":null}`,
	`{}`,
	``,
	"\x00\xff",
}

// Captions are a stranger's bytes, fetched from a site over a network, and what
// comes out of them is cues or an error and never a cue that reads the wrong
// place.
func FuzzCued(f *testing.F) {
	for _, seed := range captionSeeds {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		cues, err := cued([]byte(raw))
		if err != nil {
			return
		}
		for i, one := range cues {
			switch {
			case one.Text == "":
				t.Fatalf("cue %d says nothing", i)
			case one.To < one.From:
				t.Fatalf("cue %d runs from %d to %d", i, one.From, one.To)
			case i+1 < len(cues) && one.To > cues[i+1].From:
				t.Fatalf("cue %d ends at %d, and the next begins at %d", i, one.To, cues[i+1].From)
			}
		}
	})
}
