package source

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestRunProgressIsReadOffTheNamesTheRunWrites(t *testing.T) {
	partial := domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, talkHash)
	for name, tc := range map[string]struct {
		set  func(textScene)
		want RunProgress
	}{
		"nothing is left": {func(textScene) {}, RunProgress{State: RunNone}},
		"a finished text": {
			func(s textScene) {
				s.write(t, domain.NewArtifactName(domain.ArtifactText, domain.ASR, talkHash), "words")
			},
			RunProgress{State: RunComplete, Size: 5},
		},
		"a run holding the name": {
			func(s textScene) { s.write(t, partial, "wo"); s.claim(t, partial) },
			RunProgress{State: RunActive, Size: 2},
		},
		"a run that stopped": {
			func(s textScene) { s.write(t, partial, "wo") },
			RunProgress{State: RunStopped, Size: 2},
		},
		"a recording with no speech": {
			func(s textScene) {
				s.write(t, domain.NewArtifactName(domain.ArtifactAnswer, domain.ASR, talkHash), text.Silent)
			},
			RunProgress{State: RunNoSpeech},
		},
	} {
		t.Run(name, func(t *testing.T) {
			s := newTextScene(t)
			tc.set(s)
			got, err := s.texts.GetRunProgress.Execute(t.Context(), s.vault, recordingPath, domain.KindRecording)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Errorf("progress is %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestRunProgressOfARecordingTheIndexKnowsNothingOfExistsOnItsBytes(t *testing.T) {
	s := newTextScene(t).withoutIndex()
	s.write(t, domain.NewArtifactName(domain.ArtifactText, domain.ASR, text.Fingerprint([]byte("a recording"))), "words")

	got, err := s.texts.GetRunProgress.Execute(t.Context(), s.vault, recordingPath, domain.KindRecording)
	if err != nil || got.State != RunComplete {
		t.Errorf("progress is %+v, %v, want the text standing under the fingerprint of the bytes", got, err)
	}
}

func TestRunProgressOfAScanNoModelProducedIsNone(t *testing.T) {
	s := newTextScene(t)

	got, err := s.texts.GetRunProgress.Execute(t.Context(), s.vault, scanPath, domain.KindBook)
	if err != nil || got != (RunProgress{}) {
		t.Errorf("progress is %+v, %v, want none", got, err)
	}
}

func TestRunProgressOfALinkNoteIsReadBelowItsAddress(t *testing.T) {
	s := newTextScene(t)
	read := func() RunState {
		got, err := s.texts.GetRunProgress.Execute(t.Context(), s.vault, linkPath, domain.KindURL)
		if err != nil {
			t.Fatal(err)
		}
		return got.State
	}
	if got := read(); got != RunNone {
		t.Fatalf("nothing fetched: %v", got)
	}
	s.write(t, domain.NewArtifactName(domain.ArtifactText, domain.GetAddressProducers()[0], text.Fingerprint([]byte(string(linkedURL)))), "page")
	if got := read(); got != RunComplete {
		t.Fatalf("a page fetched: %v", got)
	}
}

func TestRunProgressOfAVaultWhoseIndexIsNotUpIsNone(t *testing.T) {
	texts := NewTexts(vaults{first.ID: newLibrary()}, nil, nil, nil)

	if got, err := texts.GetRunProgress.Execute(t.Context(), first, recordingPath, domain.KindRecording); err != nil || got != (RunProgress{}) {
		t.Errorf("progress is %+v, %v", got, err)
	}
}
