package service

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	proofreading "github.com/jiva-studio/numen/plugins/tools-proofreading"
)

func TestReadKindsReturnsTheKindsOfTheProofreadCommand(t *testing.T) {
	kinds, err := readKinds(proofreading.Manifest)
	if err != nil {
		t.Fatal(err)
	}

	got := slices.Sorted(maps.Keys(kinds))
	want := []domain.TextKind{"*"}
	if !slices.Equal(got, want) {
		t.Errorf("kinds %v, want %v", got, want)
	}
}

func TestReadKindsRejectsAManifestWithoutKinds(t *testing.T) {
	if _, err := readKinds([]byte(`{"commands":[{"id":"proofread","title":"Proofread"}]}`)); !errors.Is(err, errNoKinds) {
		t.Errorf("got %v, want %v", err, errNoKinds)
	}
}

func TestProofreadReadsTheKindsItIsGiven(t *testing.T) {
	s := newScene(t)
	s.run.kinds = map[domain.TextKind]bool{"subtitle": true}
	cue := host.Selector{T: []float64{0, 1}}.Encode()
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
	if live.Kind != "subtitle" || len(parts) != 1 || parts[0].Text != "a flywheel spinning\n" {
		t.Errorf("the live version %+v reads %+v", live, parts)
	}
	if instruction := s.model.questions[0].GetMessages()[0].GetText(); instruction != speechInstruction {
		t.Errorf("timed text was not asked about as speech: %q", instruction)
	}
}
