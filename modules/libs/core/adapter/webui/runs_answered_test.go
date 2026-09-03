package webui

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// The fingerprint of the recording a test asks about. A run keeps what it wrote
// about a recording under it, and whoever asks works it out the same way.
func sounded() string { return derived.Fingerprint([]byte(sound)) }

// A run that got no words out of a source wrote down what it got instead, and
// asking again says so rather than starting a run that answers the same.
func TestASourceAlreadyAnsweredSaysWhatCameOfIt(t *testing.T) {
	const said = "the mp3 recording: mp3: MPEG version 2.5 is not supported"
	talks := willRun()
	_, handler := running(t,
		stored{derived.Answer(listener, hashed): []byte(derived.Unopened + ": " + said + "\n")},
		indexed{talk: {Path: talk, Producer: listener, Hash: hashed}},
		willRun(), talks,
	)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeAnswered {
		t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
	}
	if back.Why != hearingUnopened+" "+said {
		t.Errorf("it does not say what came of it: %q", back.Why)
	}
	if talks.times != 0 {
		t.Error("a run was given a recording already answered")
	}
}

// A recording no run got words out of is named by no producer in the index, and
// what the run wrote is found by the bytes it wrote about.
func TestARecordingAnsweredIsFoundWhereTheIndexNamesNoProducer(t *testing.T) {
	const said = "the mp3 recording: mp3: MPEG version 2.5 is not supported"
	for _, one := range []struct {
		name string
		gave string
		why  string
	}{
		{"heard no speech", derived.Silent + "\n", hearingSilent},
		{"could not be opened", derived.Unopened + ": " + said + "\n", hearingUnopened + " " + said},
	} {
		t.Run(one.name, func(t *testing.T) {
			talks := willRun()
			_, handler := running(t,
				stored{derived.Answer(derived.ASR, sounded()): []byte(one.gave)},
				indexed{talk: {Path: talk, Hash: sounded()}},
				willRun(), talks,
			)

			back := answered(t, post(handler, transcribeAt(talk)))
			if back.Answer != outcomeAnswered {
				t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
			}
			if back.Why != one.why {
				t.Errorf("it was told %q", back.Why)
			}
			if talks.times != 0 {
				t.Error("a run was given a recording already answered")
			}
		})
	}
}

// A recording the index holds nothing at all about is answered the same way:
// the answer stands under the bytes, and nothing else names them.
func TestARecordingAnsweredIsFoundWhereTheIndexHoldsNothing(t *testing.T) {
	talks := willRun()
	_, handler := running(t,
		stored{derived.Answer(derived.ASR, sounded()): []byte(derived.Silent + "\n")},
		nothingRead(),
		willRun(), talks,
	)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeAnswered {
		t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
	}
	if talks.times != 0 {
		t.Error("a run was given a recording already answered")
	}
}

// What a run finished stands over what it once answered.
func TestASourceDoneIsDoneEvenWhereAnAnswerStands(t *testing.T) {
	talks := willRun()
	_, handler := running(t,
		stored{
			derived.Artifact(listener, hashed): []byte("what the model heard"),
			derived.Answer(listener, hashed):   []byte(derived.Silent + "\n"),
		},
		indexed{talk: {Path: talk, Producer: listener, Hash: hashed}},
		willRun(), talks,
	)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeDone {
		t.Errorf("it was answered %q: %s", back.Answer, back.Why)
	}
}

// A finished transcript stands over an answer where the index names no producer
// either.
func TestARecordingDoneIsDoneWhereTheIndexNamesNoProducer(t *testing.T) {
	talks := willRun()
	_, handler := running(t,
		stored{
			derived.Artifact(derived.ASR, sounded()): []byte("what the model heard"),
			derived.Answer(derived.ASR, sounded()):   []byte(derived.Silent + "\n"),
		},
		indexed{talk: {Path: talk, Hash: sounded()}},
		willRun(), talks,
	)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeDone {
		t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
	}
	if back.Why != heardAlready {
		t.Errorf("it was told %q", back.Why)
	}
}

// deaf is a transcriber that opens nothing, which is a recording in a form
// nothing here decodes.
type deaf struct{}

func (deaf) Transcription() port.TranscriptionModel { return port.TranscriptionModel{} }

func (deaf) Open(context.Context, []byte) (port.Recording, error) {
	return nil, errors.New("mp3: MPEG version 2.5 is not supported")
}

func (deaf) Transcribe(context.Context, port.Audio) (string, error) { return "", nil }
func (deaf) Close() error                                           { return nil }

// unrecorded is an index a run writes to and nothing reads back.
type unrecorded struct{}

func (unrecorded) SaveSource(context.Context, string, port.Source) error           { return nil }
func (unrecorded) SaveExtraction(context.Context, string, port.SourceChunks) error { return nil }

func (unrecorded) RemoveSources(context.Context, string, domain.SourceKind, []string) error {
	return nil
}

func (unrecorded) MoveSources(context.Context, string, string, string) error { return nil }

// What a run wrote about a recording it got no words out of is what the facet
// finds, over one vault and one store: both name the file by the fingerprint of
// the bytes.
func TestWhatARunAnsweredIsWhatTheFacetFinds(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	stores := filesystem.DerivedStores{Area: filesystem.SpeechDir}

	run := source.Transcribe{
		Readers: filesystem.VaultReaders{},
		Sources: unrecorded{},
		Derived: stores,
		By:      deaf{},
	}
	res, err := run.Execute(t.Context(), vault, talk)
	if err != nil {
		t.Fatalf("the run failed: %v", err)
	}
	if !res.Unopened {
		t.Fatal("the run opened a recording nothing decodes")
	}

	talks := willRun()
	api := &API{
		Readers:   filesystem.VaultReaders{},
		Highlight: &source.Highlight{Sources: nothingRead(), Derived: stores},
	}
	api.show(vault)
	runningBehind(api, func(on *showing) { on.transcribes = talks })

	back := answered(t, post(api.Serving(http.NotFoundHandler()), transcribeAt(talk)))
	if back.Answer != outcomeAnswered {
		t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
	}
	if back.Why != hearingUnopened+" mp3: MPEG version 2.5 is not supported" {
		t.Errorf("it does not say what came of it: %q", back.Why)
	}
	if talks.times != 0 {
		t.Error("a run was given a recording already answered")
	}
}

// A recording nothing has answered is put through a run.
func TestARecordingNothingAnsweredIsRun(t *testing.T) {
	talks := willRun()
	_, handler := running(t, stored{}, nothingRead(), willRun(), talks)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeStarted {
		t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
	}
	if talks.times != 1 {
		t.Errorf("the run was asked for %d times", talks.times)
	}
}
