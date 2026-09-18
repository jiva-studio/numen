package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// The span named is replaced and the rest of the note is the bytes it was,
// which is the whole reason for a tool that is not `note_rewrite`.
func TestEditingANoteChangesOnlyTheSpanNamed(t *testing.T) {
	v, core := newCoreWithNotes(t, map[string]string{
		"Aggressor.md": "# The aggressor\n\nA hedgehog is named.\n\nAnd nothing else.\n",
	})
	s := newSessionOver(t, core)

	answer := call[struct {
		Path        string `json:"path"`
		Fingerprint string `json:"fingerprint"`
		Match       string `json:"match"`
		IsLoose     bool   `json:"loose"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "A hedgehog", "text": "An axe",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})

	if answer.Match != "A hedgehog" {
		t.Errorf("what stood there is answered as %q", answer.Match)
	}
	if answer.Fingerprint == "" {
		t.Error("no fingerprint came back")
	}
	body := onDisk(t, v, "Aggressor.md")
	if !strings.Contains(body, "An axe is named.") {
		t.Errorf("the span was not replaced:\n%s", body)
	}
	if !strings.Contains(body, "And nothing else.\n") {
		t.Errorf("what was not named changed:\n%s", body)
	}
}

// A span standing twice is refused, and the refusal says how many places
// there are rather than picking one.
func TestEditingRefusesASpanThatStandsTwice(t *testing.T) {
	_, core := newCoreWithNotes(t, map[string]string{
		"Aggressor.md": "A foe advances.\n\nAnother foe advances.\n",
	})
	s := newSessionOver(t, core)

	said := getRefusal(t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "foe advances", "text": "foe retreats",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})
	if !strings.Contains(said, "2 places") {
		t.Errorf("the refusal reads: %s", said)
	}
}

// A span that is not there is refused with what the note holds in its place,
// so the next attempt is not the same guess again.
func TestEditingSaysWhatTheNoteHoldsInstead(t *testing.T) {
	_, core := newCoreWithNotes(t, map[string]string{
		"Aggressor.md": "the wrath of the advancing foe\n",
	})
	s := newSessionOver(t, core)

	said := getRefusal(t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "the wrath of the retreating foe", "text": "nothing",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})
	if !strings.Contains(said, "advancing") {
		t.Errorf("the refusal does not say what stands there: %s", said)
	}
}

// Quotes and dashes a person's editor wrote are not what a program reproduces.
// The span is found, and the answer says the reading was a loose one.
func TestEditingFindsASpanWhosePunctuationDiffers(t *testing.T) {
	v, core := newCoreWithNotes(t, map[string]string{
		"Aggressor.md": "Он сказал «да» — и ушёл.\n",
	})
	s := newSessionOver(t, core)

	answer := call[struct {
		Match   string `json:"match"`
		IsLoose bool   `json:"loose"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "сказал \"да\" - и ушёл", "text": "промолчал",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})

	if !answer.IsLoose {
		t.Error("the reading was not reported")
	}
	if answer.Match != "сказал «да» — и ушёл" {
		t.Errorf("what stood there is answered as %q", answer.Match)
	}
	if body := onDisk(t, v, "Aggressor.md"); !strings.Contains(body, "Он промолчал.\n") {
		t.Errorf("the note reads:\n%s", body)
	}
}

// An empty replacement takes the text out, which is one operation and not a
// second tool.
func TestEditingWithNothingTakesTheSpanOut(t *testing.T) {
	v, core := newCoreWithNotes(t, map[string]string{"Aggressor.md": "one two three\n"})
	s := newSessionOver(t, core)

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": " two", "text": "",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
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
	s, looking := newSessionWithWindow(t, map[string]string{
		"Aggressor.md": "A hedgehog is named.\n",
	})

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "A hedgehog", "text": "An axe",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
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
	if began.IsDone {
		t.Error("the first report ends the change")
	}
	if !ended.IsDone || ended.Change != began.Change {
		t.Errorf("the change ends as %+v", ended)
	}
}

// A change that was refused is over, and a drawing that is never ended stays
// on the screen.
func TestARefusedEditIsStillEnded(t *testing.T) {
	s, looking := newSessionWithWindow(t, map[string]string{
		"Aggressor.md": "A foe.\n\nAnother foe.\n",
	})

	getRefusal(t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "foe", "text": "friend",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})

	for _, said := range looking.drawn {
		if !said.IsDone {
			t.Fatalf("a change was begun and not ended: %+v", said)
		}
	}
}

// A note written whole is drawn as the span that changed, so a person sees
// the change and not the note.
func TestWritingANoteWholeTellsTheWindowOnlyWhatChanged(t *testing.T) {
	s, looking := newSessionWithWindow(t, map[string]string{
		"Aggressor.md": "# Title\n\nA hedgehog is named.\n\nAnd nothing else.\n",
	})

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_rewrite", map[string]any{
		"path":        "Aggressor.md",
		"body":        "# Title\n\nAn axe is named.\n\nAnd nothing else.\n",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})

	if len(looking.drawn) == 0 {
		t.Fatal("the window was told nothing")
	}
	began := looking.drawn[0]
	if began.Text != "An axe" {
		t.Errorf("what goes in is %q, wanted only what changed", began.Text)
	}
	if began.To-began.From != len("A hedgehog") {
		t.Errorf("the span is %d bytes, wanted the ten that changed", began.To-began.From)
	}
}

// A note that is not ASCII is where counting in bytes and counting the way a
// client counts part company, and a span named in bytes is drawn over the
// wrong words.
func TestTheSpanIsCountedTheWayAClientCountsText(t *testing.T) {
	s, looking := newSessionWithWindow(t, map[string]string{
		"Aggressor.md": "Он сказал да.\n",
	})

	call[struct {
		Path string `json:"path"`
	}](t, s, "note_edit", map[string]any{
		"path": "Aggressor.md", "match": "сказал", "text": "промолчал",
		"fingerprint": fingerprint(t, s, "Aggressor.md"),
	})

	began := looking.drawn[0]
	// "Он " is three characters and four bytes.
	if began.From != 3 || began.To != 9 {
		t.Errorf("the span is %d..%d, wanted 3..9", began.From, began.To)
	}
}

// Every tool that writes takes the fingerprint of what it read, and one
// presenting none is refused with the read that gives it named.
func TestAWriteWithNoFingerprintIsRefused(t *testing.T) {
	session, _ := newSession(t, map[string]string{
		"Animal.md":    stencil,
		"Animals.md":   deck,
		"Aggressor.md": "A hedgehog is named.\n",
	})

	writes := []struct {
		tool string
		args map[string]any
	}{
		{"note_rewrite", map[string]any{"path": "Aggressor.md", "body": "An axe.\n"}},
		{"note_edit", map[string]any{
			"path": "Aggressor.md", "match": "A hedgehog", "text": "An axe",
		}},
		{"card_add", map[string]any{
			"path": "Animals.md", "stencil": "Animal",
			"values": []map[string]string{{"field": "Name", "text": "Vicuña"}},
		}},
		{"card_edit", map[string]any{
			"path": "Animals.md", "mark": llama,
			"values": []map[string]string{{"field": "Height", "text": "about 46\""}},
		}},
		{"card_remove", map[string]any{"path": "Animals.md", "mark": llama}},
		{"card_section_add", map[string]any{"path": "Animals.md", "name": "Others"}},
	}
	for _, w := range writes {
		t.Run(w.tool, func(t *testing.T) {
			w.args["fingerprint"] = ""
			said := getRefusal(t, session, w.tool, w.args)
			if !strings.Contains(said, "note_read") || !strings.Contains(said, "card_read") {
				t.Errorf("the refusal does not say where a fingerprint comes from: %s", said)
			}
		})
	}
}
