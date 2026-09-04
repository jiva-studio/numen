package webui

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	derived "github.com/jiva-studio/numen/modules/libs/core/text"
)

// What a file carries is asked before anything is offered over it. A window
// that cannot ask has only one way to find out whether a book has been read: to
// ask for it to be read again and be refused.

// A book nothing has read carries a reading all the same, and it is nothing.
// The row is the answer, and its absence would be a build that makes no
// readings at all.
func TestABookNothingHasReadCarriesAReadingThatIsNothing(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	held := carrying(t, api, book)
	if len(held) != 1 {
		t.Fatalf("a book carries %v", held)
	}
	if held[readingID] != v1.State_STATE_NONE {
		t.Errorf("the reading of a book nothing read is %s", held[readingID])
	}
}

// A book a model has read carries that reading, and how long it is.
func TestABookAModelHasReadCarriesTheReading(t *testing.T) {
	const wrote = "what the model read off the pages"
	api, _ := running(t,
		stored{derived.Artifact(reader, scanned): []byte(wrote)},
		indexed{book: {Path: book, Producer: reader, Hash: scanned}},
		willRun(), willRun(),
	)

	out, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: book}))
	if err != nil {
		t.Fatalf("asked what the book carries and was refused: %v", err)
	}
	held := out.Msg.GetArtifacts()
	if len(held) != 1 {
		t.Fatalf("a book carries %v", held)
	}
	if held[0].GetState() != v1.State_STATE_DONE {
		t.Errorf("a book a model read is %s", held[0].GetState())
	}
	if held[0].GetSize() != int64(len(wrote)) {
		t.Errorf("the reading is %d bytes long", held[0].GetSize())
	}
	if held[0].GetName() != named(api.Showing(), book, readingID) {
		t.Errorf("the reading stands at %q", held[0].GetName())
	}
}

// A recording carries two: the words a model heard, and those words put right.
// They stand in the order they are made.
func TestARecordingCarriesWhatWasHeardAndWhatWasPutRight(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	out, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: talk}))
	if err != nil {
		t.Fatalf("asked what the recording carries and was refused: %v", err)
	}
	want := []string{
		named(api.Showing(), talk, heardID),
		named(api.Showing(), talk, correctedID),
	}
	held := out.Msg.GetArtifacts()
	if len(held) != len(want) {
		t.Fatalf("a recording carries %v", held)
	}
	for at, one := range held {
		if one.GetName() != want[at] {
			t.Errorf("the artifact at %d stands at %q, want %q", at, one.GetName(), want[at])
		}
		if one.GetState() != v1.State_STATE_NONE {
			t.Errorf("%s of a recording nothing heard is %s", one.GetName(), one.GetState())
		}
	}
}

// A file nothing is made from carries nothing, and the window offers nothing
// over it.
func TestAFileNothingIsMadeFromCarriesNothing(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	if held := carrying(t, api, idea); len(held) != 0 {
		t.Errorf("a note carries %v", held)
	}
}

// A path the vault does not hold carries nothing that can be asked about, and
// is refused rather than answered with an empty list.
func TestWhatAPathTheVaultDoesNotHoldCarriesIsNotFound(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	_, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{
		Path: "library/nothing.pdf",
	}))
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("asked about nothing and was refused %v", err)
	}
}

// A run that got no words out of a recording is what the recording carries, and
// what the run said about it stands with it.
func TestWhatARunAnsweredIsWhatARecordingCarries(t *testing.T) {
	const said = "mp3: MPEG version 2.5 is not supported"
	for _, one := range []struct {
		name  string
		gave  string
		state v1.State
		why   string
	}{
		{"heard no speech", derived.Silent + "\n", v1.State_STATE_EMPTY, ""},
		{"could not be opened", derived.Unopened + ": " + said + "\n", v1.State_STATE_FAILED, said},
	} {
		t.Run(one.name, func(t *testing.T) {
			api, _ := running(t,
				stored{derived.Answer(asr, hashed): []byte(one.gave)},
				indexed{talk: {Path: talk, Producer: asr, Hash: hashed}},
				willRun(), willRun(),
			)

			out, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{
				Path: talk,
			}))
			if err != nil {
				t.Fatalf("asked what the recording carries and was refused: %v", err)
			}
			heard := out.Msg.GetArtifacts()[0]
			if heard.GetState() != one.state {
				t.Fatalf("the recording carries %s", heard.GetState())
			}
			if heard.GetError() != one.why {
				t.Errorf("it says %q came of it", heard.GetError())
			}
		})
	}
}

// A window standing on nothing holds no file to carry anything.
func TestAWindowStandingOnNothingCarriesNothing(t *testing.T) {
	api := &API{}

	_, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: book}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Errorf("a window with no vault was refused %v", err)
	}
}
