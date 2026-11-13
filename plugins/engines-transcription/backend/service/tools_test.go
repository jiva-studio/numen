package service

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func TestRegistryListsTranscriptCreateAsAWritingTool(t *testing.T) {
	w := newWire(t)

	listing, err := w.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	infos := listing.Tools
	if len(infos) != 1 || infos[0].GetName() != "transcript_create" {
		t.Fatalf("the registry lists %+v, want transcript_create alone", infos)
	}
	if infos[0].GetEffect() != tools.ReadWrite {
		t.Errorf("transcript_create has effect %v, want ReadWrite", infos[0].GetEffect())
	}
}

func TestTranscriptCreateRejectsACallWithoutAVault(t *testing.T) {
	w := newWire(t)

	res, err := w.registry.InvokeTool(t.Context(), tools.Call{
		VaultID: "", Name: "transcript_create", Arguments: []byte(`{"path":"` + lecture + `"}`),
	})

	if err != nil {
		t.Fatal(err)
	}
	if !res.GetResult().GetIsError() {
		t.Errorf("a call with no vault returned %q %s, want an error", res.GetResult().GetText(), res.GetResult().GetStructured())
	}
}

func TestTheTranscriptionToolHasItsInstructions(t *testing.T) {
	w := newWire(t)

	listing, err := w.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	sentences := getToolSentences(listing, "transcript_create")
	if len(sentences) == 0 {
		t.Fatal("transcript_create lists no instructions")
	}
	if !strings.Contains(strings.Join(sentences, " "), "transcript_create") {
		t.Errorf("no instruction of transcript_create names it: %q", sentences)
	}
	for _, sentence := range sentences {
		if !isOneSentence(sentence) {
			t.Errorf("an instruction of transcript_create is not one sentence: %q", sentence)
		}
	}
}
