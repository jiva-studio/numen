package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// The stretch named is replaced and the rest of the note is the bytes it was,
// which is the whole reason for a tool that is not `note_write`.
func TestEditingANoteChangesOnlyTheStretchNamed(t *testing.T) {
	v, core := built(t, map[string]string{
		"Aggressor.md": "# The aggressor\n\nA hedgehog is named.\n\nAnd nothing else.\n",
	})
	s := connectedTo(t, core)

	answer := call[struct {
		Path        string `json:"path"`
		Fingerprint string `json:"fingerprint"`
		Stood       string `json:"stood"`
		Plainly     bool   `json:"plainly"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": "A hedgehog", "becomes": "An axe",
	})

	if answer.Stood != "A hedgehog" {
		t.Errorf("what stood there is answered as %q", answer.Stood)
	}
	if answer.Fingerprint == "" {
		t.Error("no fingerprint came back")
	}
	body := onDisk(t, v, "Aggressor.md")
	if !strings.Contains(body, "An axe is named.") {
		t.Errorf("the stretch was not replaced:\n%s", body)
	}
	if !strings.Contains(body, "And nothing else.\n") {
		t.Errorf("what was not named changed:\n%s", body)
	}
}

// A stretch standing twice is refused, and the refusal says how many places
// there are rather than picking one.
func TestEditingRefusesAStretchThatStandsTwice(t *testing.T) {
	_, core := built(t, map[string]string{
		"Aggressor.md": "A foe advances.\n\nAnother foe advances.\n",
	})
	s := connectedTo(t, core)

	said := failing(t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": "foe advances", "becomes": "foe retreats",
	})
	if !strings.Contains(said, "2 places") {
		t.Errorf("the refusal reads: %s", said)
	}
}

// A stretch that is not there is refused with what the note holds in its place,
// so the next attempt is not the same guess again.
func TestEditingSaysWhatTheNoteHoldsInstead(t *testing.T) {
	_, core := built(t, map[string]string{
		"Aggressor.md": "the wrath of the advancing foe\n",
	})
	s := connectedTo(t, core)

	said := failing(t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": "the wrath of the retreating foe", "becomes": "nothing",
	})
	if !strings.Contains(said, "advancing") {
		t.Errorf("the refusal does not say what stands there: %s", said)
	}
}

// Quotes and dashes a person's editor wrote are not what a program reproduces.
// The stretch is found, and the answer says the reading was a loose one.
func TestEditingFindsAStretchWhosePunctuationDiffers(t *testing.T) {
	v, core := built(t, map[string]string{
		"Aggressor.md": "Он сказал «да» — и ушёл.\n",
	})
	s := connectedTo(t, core)

	answer := call[struct {
		Stood   string `json:"stood"`
		Plainly bool   `json:"plainly"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": "сказал \"да\" - и ушёл", "becomes": "промолчал",
	})

	if !answer.Plainly {
		t.Error("the reading was not reported")
	}
	if answer.Stood != "сказал «да» — и ушёл" {
		t.Errorf("what stood there is answered as %q", answer.Stood)
	}
	if body := onDisk(t, v, "Aggressor.md"); !strings.Contains(body, "Он промолчал.\n") {
		t.Errorf("the note reads:\n%s", body)
	}
}

// An empty replacement takes the text out, which is one operation and not a
// second tool.
func TestEditingWithNothingTakesTheStretchOut(t *testing.T) {
	v, core := built(t, map[string]string{"Aggressor.md": "one two three\n"})
	s := connectedTo(t, core)

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": " two", "becomes": "",
	})

	if body := onDisk(t, v, "Aggressor.md"); !strings.Contains(body, "one three") {
		t.Errorf("the note reads:\n%s", body)
	}
}

// onDisk is a note's file as it now stands, for what an answer does not carry.
func onDisk(t *testing.T, v domain.Vault, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(v.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// A person reading the note sees the change arrive where it belongs, so an
// edit says what it is doing before it does it, and says when it is over.
func TestEditingTellsTheWindowWhereItIsChangingTheNote(t *testing.T) {
	s, looking := watched(t, map[string]string{
		"Aggressor.md": "A hedgehog is named.\n",
	})

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": "A hedgehog", "becomes": "An axe",
	})

	if len(looking.drawn) != 2 {
		t.Fatalf("the window was told %d times, wanted two", len(looking.drawn))
	}
	began, ended := looking.drawn[0], looking.drawn[1]
	if began.Path != "Aggressor.md" || began.From != 0 || began.To != 10 {
		t.Errorf("the change begins as %+v", began)
	}
	if began.Text != "An axe" {
		t.Errorf("what goes in is %q", began.Text)
	}
	if began.Done {
		t.Error("the first report ends the change")
	}
	if !ended.Done || ended.Change != began.Change {
		t.Errorf("the change ends as %+v", ended)
	}
}

// A change that was refused is over, and a drawing that is never ended stays
// on the screen.
func TestARefusedEditIsStillEnded(t *testing.T) {
	s, looking := watched(t, map[string]string{
		"Aggressor.md": "A foe.\n\nAnother foe.\n",
	})

	failing(t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "stood": "foe", "becomes": "friend",
	})

	for _, said := range looking.drawn {
		if !said.Done {
			t.Fatalf("a change was begun and not ended: %+v", said)
		}
	}
}

// A note written whole is drawn as the stretch that changed, so a person sees
// the change and not the note.
func TestWritingANoteWholeTellsTheWindowOnlyWhatChanged(t *testing.T) {
	s, looking := watched(t, map[string]string{
		"Aggressor.md": "# Title\n\nA hedgehog is named.\n\nAnd nothing else.\n",
	})

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_write", map[string]any{
		"path": "Aggressor.md",
		"body": "# Title\n\nAn axe is named.\n\nAnd nothing else.\n",
	})

	if len(looking.drawn) == 0 {
		t.Fatal("the window was told nothing")
	}
	began := looking.drawn[0]
	if began.Text != "An axe" {
		t.Errorf("what goes in is %q, wanted only what changed", began.Text)
	}
	if began.To-began.From != len("A hedgehog") {
		t.Errorf("the stretch is %d bytes, wanted the ten that changed", began.To-began.From)
	}
}
