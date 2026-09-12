package editor

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	derived "github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/source"
)

// The fingerprint of the recording a test asks about. A run keeps what it wrote
// about a recording under it, and whoever asks works it out the same way.
func sounded() string { return derived.Fingerprint([]byte(sound)) }

// A run that got no words out of a source wrote down what came of it, and
// asking again says that.
func TestASourceAlreadyAnsweredSaysWhatCameOfIt(t *testing.T) {
	const said = "the mp3 recording: mp3: MPEG version 2.5 is not supported"
	talks := willRun()
	api, _ := running(t,
		stored{derived.Answer(asr, hashed): []byte(derived.Unopened + ": " + said + "\n")},
		indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Producer: asr, Hash: hashed}},
		willRun(), talks,
	)

	made := making(t, api, talk, transcriptOf)
	if made.GetState() != v1.State_STATE_FAILED {
		t.Fatalf("it was answered %s", made.GetState())
	}
	if made.GetError() != said {
		t.Errorf("it does not say what came of it: %q", made.GetError())
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
		name  string
		gave  string
		state v1.State
		why   string
	}{
		{"heard no speech", derived.Silent + "\n", v1.State_STATE_EMPTY, ""},
		{"could not be opened", derived.Unopened + ": " + said + "\n", v1.State_STATE_FAILED, said},
	} {
		t.Run(one.name, func(t *testing.T) {
			talks := willRun()
			api, _ := running(t,
				stored{derived.Answer(derived.ASR, sounded()): []byte(one.gave)},
				indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Hash: sounded()}},
				willRun(), talks,
			)

			made := making(t, api, talk, transcriptOf)
			if made.GetState() != one.state {
				t.Fatalf("it was answered %s", made.GetState())
			}
			if made.GetError() != one.why {
				t.Errorf("it says %q came of it", made.GetError())
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
	api, _ := running(t,
		stored{derived.Answer(derived.ASR, sounded()): []byte(derived.Silent + "\n")},
		nothingRead(),
		willRun(), talks,
	)

	if made := making(t, api, talk, transcriptOf); made.GetState() != v1.State_STATE_EMPTY {
		t.Fatalf("it was answered %s", made.GetState())
	}
	if talks.times != 0 {
		t.Error("a run was given a recording already answered")
	}
}

// What a run finished stands over what it once answered.
func TestASourceDoneIsDoneEvenWhereAnAnswerStands(t *testing.T) {
	talks := willRun()
	api, _ := running(t,
		stored{
			derived.Artifact(asr, hashed): []byte("what the model heard"),
			derived.Answer(asr, hashed):   []byte(derived.Silent + "\n"),
		},
		indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Producer: asr, Hash: hashed}},
		willRun(), talks,
	)

	if made := making(t, api, talk, transcriptOf); made.GetState() != v1.State_STATE_DONE {
		t.Errorf("it was answered %s", made.GetState())
	}
}

// A finished transcript stands over an answer where the index names no producer
// either.
func TestARecordingDoneIsDoneWhereTheIndexNamesNoProducer(t *testing.T) {
	const heard = "what the model heard"
	talks := willRun()
	api, _ := running(t,
		stored{
			derived.Artifact(derived.ASR, sounded()): []byte(heard),
			derived.Answer(derived.ASR, sounded()):   []byte(derived.Silent + "\n"),
		},
		indexed{talk: {Fingerprint: domain.Fingerprint{Path: talk}, Hash: sounded()}},
		willRun(), talks,
	)

	made := making(t, api, talk, transcriptOf)
	if made.GetState() != v1.State_STATE_DONE {
		t.Fatalf("it was answered %s", made.GetState())
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

func (unrecorded) SaveSource(context.Context, domain.VaultID, domain.Source) error { return nil }
func (unrecorded) SaveExtraction(context.Context, domain.VaultID, domain.SourceChunks) error {
	return nil
}

func (unrecorded) RemoveSources(context.Context, domain.VaultID, domain.SourceKind, []string) error {
	return nil
}

func (unrecorded) MoveSources(context.Context, domain.VaultID, string, string) error { return nil }

// What a run wrote about a recording it got no words out of is what the facet
// finds, over one vault and one store: both name the file by the fingerprint of
// the bytes.
func TestWhatARunAnsweredIsWhatTheFacetFinds(t *testing.T) {
	vault := testsupport.NewVault(t, map[string]string{talk: sound})
	stores := filesystem.DerivedStores{Area: filesystem.TranscriptDir}

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
	runningBehind(api, func(on *passes) { on.transcribes = talks })

	made := making(t, api, talk, transcriptOf)
	if made.GetState() != v1.State_STATE_FAILED {
		t.Fatalf("it was answered %s", made.GetState())
	}
	if made.GetError() != "mp3: MPEG version 2.5 is not supported" {
		t.Errorf("it does not say what came of it: %q", made.GetError())
	}
	if talks.times != 0 {
		t.Error("a run was given a recording already answered")
	}
}

// A recording nothing has answered is put through a run.
func TestARecordingNothingAnsweredIsRun(t *testing.T) {
	talks := willRun()
	api, _ := running(t, stored{}, nothingRead(), willRun(), talks)

	if made := making(t, api, talk, transcriptOf); made.GetState() != v1.State_STATE_RUNNING {
		t.Fatalf("it was answered %s", made.GetState())
	}
	if talks.times != 1 {
		t.Errorf("the run was asked for %d times", talks.times)
	}
}
