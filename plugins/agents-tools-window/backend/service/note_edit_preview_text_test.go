package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-window/backend/service"
)

type editPreviewAnswer struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

func TestNoteEditPreviewWithProposedTextDiffsAgainstTheNote(t *testing.T) {
	w := newWindowTools(t)
	proposed := "---\nid: 01H7X1234567890ABCDEFGHJKM\ntitle: Gray Heron\n---\nHerons fish at dawn."

	var answer editPreviewAnswer
	if failure := w.call(t, "note_edit_preview", `{"path":"`+notePath+`","proposedText":`+encodeJSONString(t, proposed)+`}`, nil, &answer); failure != "" {
		t.Fatalf("note_edit_preview with proposedText failed: %s", failure)
	}

	if !strings.Contains(answer.Diff, "-Herons fish at dusk.") || !strings.Contains(answer.Diff, "+Herons fish at dawn.") {
		t.Errorf("the diff is %q, want dusk removed and dawn added", answer.Diff)
	}
	if answer.Additions != 1 || answer.Deletions != 1 {
		t.Errorf("the diff has +%d -%d, want +1 -1", answer.Additions, answer.Deletions)
	}
}

func TestNoteEditPreviewWithOriginalTextDiffsAgainstIt(t *testing.T) {
	w := newWindowTools(t)

	var answer editPreviewAnswer
	arguments := `{"path":"` + notePath + `","originalText":"Egrets wade.\n","proposedText":"Egrets wade.\nEgrets fly.\n"}`
	if failure := w.call(t, "note_edit_preview", arguments, nil, &answer); failure != "" {
		t.Fatalf("note_edit_preview with originalText failed: %s", failure)
	}

	if !strings.Contains(answer.Diff, "+Egrets fly.") || strings.Contains(answer.Diff, "dusk") {
		t.Errorf("the diff is %q, want Egrets fly. added against the original text and nothing of the note", answer.Diff)
	}
	if answer.Additions != 1 || answer.Deletions != 0 {
		t.Errorf("the diff has +%d -%d, want +1 -0", answer.Additions, answer.Deletions)
	}
}

func TestNoteEditPreviewSchemaNamesTheTextArguments(t *testing.T) {
	w := newWindowTools(t)

	properties := getSchemaProperties(t, getToolSchema(t, w.registry, "note_edit_preview", false))

	requireKeys(t, "the input schema of note_edit_preview", properties,
		[]string{"path", "proposedText", "originalText", "diff"}, []string{"proposedContent", "originalContent"})
}

func TestNoteEditPreviewWithoutProposedTextOrDiffNamesProposedText(t *testing.T) {
	w := newWindowTools(t)

	failure := w.call(t, "note_edit_preview", `{"path":"`+notePath+`"}`, nil, nil)

	if want := "either proposedText or diff is required"; !strings.Contains(failure, want) {
		t.Errorf("note_edit_preview with neither proposedText nor diff gave %q, want %q", failure, want)
	}
}

func encodeJSONString(t *testing.T, text string) string {
	t.Helper()
	encoded, err := json.Marshal(text)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

func TestNoteEditPreviewInARunOnAnotherVaultDiffsAgainstTheNoteOfThatVault(t *testing.T) {
	shown, other := hosttest.New("marsh"), hosttest.New("dune")
	shown.Beside(other)
	shown.File("notes/Plover.md", "Plovers nest on the marsh.")
	other.File("notes/Plover.md", "Plovers nest on the dune.")
	registry := newRegistry(t, service.Host{Files: filesOf(shown)})

	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID: string(other.Vault().ID), Name: "note_edit_preview",
		Arguments: []byte(`{"path":"notes/Plover.md","proposedText":"Plovers nest on the dune and the beach."}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var answer editPreviewAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &answer); err != nil {
		t.Fatalf("%v in %s", err, res.GetResult().GetStructured())
	}

	if !strings.Contains(answer.Diff, "-Plovers nest on the dune.") || strings.Contains(answer.Diff, "marsh") {
		t.Errorf("the diff is %q, want the note of the dune vault replaced and nothing of the marsh vault", answer.Diff)
	}
	if answer.Additions != 1 || answer.Deletions != 1 {
		t.Errorf("the diff has +%d -%d, want +1 -1", answer.Additions, answer.Deletions)
	}
}

func TestNoteEditPreviewWithNoVaultOnTheCallDiffsAgainstTheNoteOfTheShownVault(t *testing.T) {
	shown, other := hosttest.New("marsh"), hosttest.New("dune")
	shown.Beside(other)
	shown.File("notes/Plover.md", "Plovers nest on the marsh.")
	other.File("notes/Plover.md", "Plovers nest on the dune.")
	registry := newRegistry(t, service.Host{Files: filesOf(shown)})

	res, err := registry.InvokeTool(t.Context(), tools.Call{
		Name:      "note_edit_preview",
		Arguments: []byte(`{"path":"notes/Plover.md","proposedText":"Plovers nest on the marsh and the beach."}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var answer editPreviewAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &answer); err != nil {
		t.Fatalf("%v in %s", err, res.GetResult().GetStructured())
	}

	if !strings.Contains(answer.Diff, "-Plovers nest on the marsh.") || strings.Contains(answer.Diff, "dune") {
		t.Errorf("the diff is %q, want the note of the marsh vault replaced and nothing of the dune vault", answer.Diff)
	}
}

func TestNoteEditPreviewOfANoteThatDoesNotExistYetShowsItAllAdded(t *testing.T) {
	shown := hosttest.New("marsh")
	registry := newRegistry(t, service.Host{Files: filesOf(shown)})

	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID: "marsh", Name: "note_edit_preview",
		Arguments: []byte(`{"path":"notes/Lapwing.md","proposedText":"Lapwings call.\n"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	var answer editPreviewAnswer
	if err := json.Unmarshal(res.GetResult().GetStructured(), &answer); err != nil {
		t.Fatalf("%v in %s", err, res.GetResult().GetStructured())
	}

	if answer.Additions != 1 || answer.Deletions != 0 {
		t.Errorf("the diff has +%d -%d, want +1 -0", answer.Additions, answer.Deletions)
	}
}

func TestNoteEditPreviewInAnUnknownVaultIsRefused(t *testing.T) {
	shown := hosttest.New("marsh")
	shown.File("notes/Plover.md", "Plovers nest on the marsh.")
	registry := newRegistry(t, service.Host{Files: filesOf(shown)})

	res, err := registry.InvokeTool(t.Context(), tools.Call{
		VaultID: "removed", Name: "note_edit_preview",
		Arguments: []byte(`{"path":"notes/Plover.md","proposedText":"Plovers nest on the beach."}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	if !res.GetResult().GetIsError() {
		t.Errorf("the answer is %s, want a refusal for a vault that is not known", res.GetResult().GetStructured())
	}
}
