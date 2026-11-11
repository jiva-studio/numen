package proofread

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/prompt"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/segments"
)

func TestGetProofreadKindsAcceptsEveryKind(t *testing.T) {
	got := slices.Sorted(maps.Keys(GetProofreadKinds()))
	want := []domain.TextKind{"*"}
	if !slices.Equal(got, want) {
		t.Errorf("kinds %v, want %v", got, want)
	}
}

func encodeSelector(t *testing.T, selector segments.Selector) domain.Selector {
	t.Helper()
	encoded, err := selector.Encode()
	if err != nil {
		t.Fatal(err)
	}
	return encoded
}

func TestProofreadReadsTheKindsItIsGiven(t *testing.T) {
	s := newScene(t)
	s.run.kinds = map[domain.TextKind]bool{"subtitle": true}
	cue := encodeSelector(t, segments.Selector{T: []float64{0, 1}})
	subtitle := s.hold(t, speechID, talkPath, "subtitle", "subtitle", "the recording", []domain.TextSegment{
		{Text: "a flywheel spinnig\n", Selector: cue},
	})
	scan := s.holdScan(t)
	s.model.script = func(prompt string) string {
		return getRows(prompt, func(_, text string) string { return strings.ReplaceAll(text, "spinnig", "spinning") })
	}

	if _, err := s.run.Proofread(t.Context(), Request{Path: talkPath, Source: subtitle.GetID()}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.run.Proofread(t.Context(), Request{Path: scanPath, Source: scan.GetID()}); !errors.Is(err, errFormat) {
		t.Errorf("a recognition outside the kinds: got %v, want %v", err, errFormat)
	}

	parts, live := s.readLive(t, talkPath)
	if live.Kind != ProofreadingKind || len(parts) != 1 || parts[0].Text != "a flywheel spinning\n" {
		t.Errorf("the live version %+v reads %+v", live, parts)
	}
	if instruction := s.model.instructions[0]; instruction != prompt.SpeechInstruction {
		t.Errorf("timed text was not asked about as speech: %q", instruction)
	}
}
