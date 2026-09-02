package webui

import (
	"strings"
	"testing"

	derived "github.com/jiva-studio/numen/modules/libs/core/text"
)

// A run that got no words out of a source wrote down what it got instead, and
// asking again says so rather than starting a run that answers the same.
func TestASourceAlreadyAnsweredSaysWhatCameOfIt(t *testing.T) {
	const said = "unopened: the mp3 recording: mp3: MPEG version 2.5 is not supported"
	talks := willRun()
	_, handler := running(t,
		stored{derived.Answer(listener, hashed): []byte(said + "\n")},
		indexed{talk: {Path: talk, From: listener, Hash: hashed}},
		willRun(), talks,
	)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeAnswered {
		t.Fatalf("it was answered %q: %s", back.Answer, back.Why)
	}
	if !strings.Contains(back.Why, said) {
		t.Errorf("it does not say what came of it: %q", back.Why)
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
			derived.Answer(listener, hashed):   []byte("silent\n"),
		},
		indexed{talk: {Path: talk, From: listener, Hash: hashed}},
		willRun(), talks,
	)

	back := answered(t, post(handler, transcribeAt(talk)))
	if back.Answer != outcomeDone {
		t.Errorf("it was answered %q: %s", back.Answer, back.Why)
	}
}
