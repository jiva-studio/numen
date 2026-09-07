package editor

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
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

	// A scan carries its reading and that reading put right: what a proofreader
	// writes is another thing to ask about, and is asked about here.
	held := carrying(t, api, book)
	if len(held) != 2 {
		t.Fatalf("a book carries %v", held)
	}
	if held[readingOf] != v1.State_STATE_NONE {
		t.Errorf("the reading of a book nothing read is %s", held[readingOf])
	}
}

// A book a model has read carries that reading, and how long it is.
func TestABookAModelHasReadCarriesTheReading(t *testing.T) {
	const wrote = "what the model read off the pages"
	api, _ := running(t,
		stored{derived.Artifact(reader, scanned): []byte(wrote)},
		indexed{book: {Fingerprint: domain.Fingerprint{Path: book}, Producer: reader, Hash: scanned}},
		willRun(), willRun(),
	)

	out, err := api.ListArtifacts(t.Context(), connect.NewRequest(&v1.ListArtifactsRequest{Path: book}))
	if err != nil {
		t.Fatalf("asked what the book carries and was refused: %v", err)
	}
	held := out.Msg.GetArtifacts()
	if len(held) != 2 {
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
		named(api.Showing(), talk, transcriptID),
		named(api.Showing(), talk, transcriptCorrectedID),
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
// the question is refused.
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
				indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Producer: asr, Hash: hashed}},
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

// A note that points nowhere carries nothing: what is made from a file follows
// from what the file is.
func TestANoteThatPointsNowhereCarriesNothing(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	if held := carrying(t, api, idea); len(held) != 0 {
		t.Errorf("an ordinary note carries %v", held)
	}
	if code := refusedMaking(t, api, idea, transcriptOf); code != connect.CodeInvalidArgument {
		t.Errorf("fetching for an ordinary note was refused with %s", code)
	}
}

// A link note carries what is at its address, and it is nothing until something
// has been fetched.
func TestALinkNoteCarriesWhatIsAtItsAddress(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	held := carrying(t, api, pointed)
	if len(held) != 2 {
		t.Fatalf("a link note carries %v", held)
	}
	if held[transcriptOf] != v1.State_STATE_NONE {
		t.Errorf("a link note nothing has fetched for carries %s", held[transcriptOf])
	}
}

// Once the words are fetched, the note carries them, however the note was
// edited since: what came back is kept under the address and not under the
// note's bytes.
func TestALinkNoteCarriesTheWordsFetchedForIt(t *testing.T) {
	const words = "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nwhat was said\n"
	hash := derived.Fingerprint([]byte(pointsAt))
	api, _ := running(t,
		stored{derived.Artifact(derived.Captions, hash): []byte(words)},
		nothingRead(), willRun(), willRun(),
	)

	held := carrying(t, api, pointed)
	if held[transcriptOf] != v1.State_STATE_DONE {
		t.Errorf("a link note the words were fetched for carries %s", held[transcriptOf])
	}
}

// A build on a machine holding neither tool says so, and the window offers the
// fetch nowhere from then on.
func TestABuildThatCannotFetch(t *testing.T) {
	api, _ := running(t, stored{}, nothingRead(), willRun(), willRun())

	if code := refusedMaking(t, api, pointed, transcriptOf); code != connect.CodeUnimplemented {
		t.Errorf("a build with no fetcher refused with %s", code)
	}
}

// The words fetched for a link note are read back the way a recording's are:
// they are words with times in them, and one tab draws both.
func TestTheWordsOfALinkNoteAreReadBack(t *testing.T) {
	const words = "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nwhat was said\n"
	hash := derived.Fingerprint([]byte(pointsAt))
	api, _ := running(t,
		stored{derived.Artifact(derived.Captions, hash): []byte(words)},
		indexed{pointed: {
			Fingerprint: domain.Fingerprint{Path: pointed},
			Producer:    derived.Captions,
			Hash:        hash,
		}},
		willRun(), willRun(),
	)

	out, err := api.ReadTranscript(t.Context(),
		connect.NewRequest(&v1.ReadTranscriptRequest{Path: pointed}))
	if err != nil {
		t.Fatal(err)
	}
	cues := out.Msg.GetSpoken().GetCues()
	if len(cues) != 1 || cues[0].GetText() != "what was said" || cues[0].GetFrom() != 1000 {
		t.Errorf("the words read %+v", cues)
	}
}

// A copy of a video is asked for by hand, and taking it away leaves the note
// pointing where it pointed.
func TestACopyIsFetchedAndTakenAway(t *testing.T) {
	hash := derived.Fingerprint([]byte(pointsAt))
	held := stored{derived.Copy(hash): []byte("the bytes of a video")}
	api, _ := running(t, held, nothingRead(), willRun(), willRun())

	if carrying(t, api, pointed)[copyOf] != v1.State_STATE_DONE {
		t.Fatalf("a note with a copy carries %v", carrying(t, api, pointed))
	}

	if _, err := api.DeleteArtifact(t.Context(), connect.NewRequest(&v1.DeleteArtifactRequest{
		Path: pointed, Kind: v1.ArtifactKind_ARTIFACT_KIND_COPY,
	})); err != nil {
		t.Fatal(err)
	}
	if carrying(t, api, pointed)[copyOf] != v1.State_STATE_NONE {
		t.Errorf("the copy is still here: %v", carrying(t, api, pointed))
	}
	if _, held := held[derived.Copy(hash)]; held {
		t.Error("the bytes are still in the store")
	}
}

// A link note is asked what it is before its words are read, the way a
// recording is. A note pointing at a video answers, and one asked where no copy
// stands answers all the same: the frame plays it, and the words stand under it.
func TestALinkNoteIsAskedWhatItIsBeforeItsWords(t *testing.T) {
	const words = "WEBVTT\n\n00:00:01.000 --> 00:00:02.000\nwhat was said\n"
	hash := derived.Fingerprint([]byte(pointsAt))
	api, _ := running(t,
		stored{derived.Artifact(derived.Captions, hash): []byte(words)},
		indexed{pointed: {
			Fingerprint: domain.Fingerprint{Path: pointed},
			Producer:    derived.Captions,
			Hash:        hash,
		}},
		willRun(), willRun(),
	)

	out, err := api.GetRecording(t.Context(),
		connect.NewRequest(&v1.GetRecordingRequest{Path: pointed}))
	if err != nil {
		t.Fatalf("a link note was refused before its words were read: %v", err)
	}
	if out.Msg.GetLength() == 0 {
		t.Errorf("it runs %d ms, and the words reach further", out.Msg.GetLength())
	}
}
