package claudecode_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/claudecode"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// drawn is a window that keeps what it was told about a change being made, and
// a vault that says one stretch stands in one place.
type drawn struct {
	said  []domain.Edit
	at    time.Time
	found bool
	asked []string
}

func (d *drawn) drafting() claudecode.Drafting {
	return claudecode.Drafting{
		Report: func(_ context.Context, said domain.Edit) { d.said = append(d.said, said) },
		Location: func(_ context.Context, path, stood string) (int, int, bool) {
			d.asked = append(d.asked, stood)
			return 3, 9, d.found
		},
		// Every frame is a moment later, so the pace never holds one back.
		Now: func() time.Time { d.at = d.at.Add(time.Second); return d.at },
	}
}

// drafting is the agent with a window behind it, fed a canned stream.
func drafting(t *testing.T, window *drawn, prints string) port.Run {
	t.Helper()

	dir := t.TempDir()
	script := filepath.Join(dir, "claude")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat <<'SAID'\n"+prints+"\nSAID\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	claude := claudecode.Agent{
		Command: []string{script},
		Root:    dir,
		Tools:   claudecode.Endpoint{URL: "http://127.0.0.1:7717/mcp", Token: "let-me-in"},
		Words: map[string]claudecode.ToolDeclaration{
			claudecode.Tool("note_edit"): {
				Title: "Edit a note", Kind: port.StepEdit,
				Arguments: claudecode.Arguments{
					About: "path", Match: "stood", Text: "becomes",
				},
			},
		},
		Drafting: window.drafting(),
	}
	work, err := claude.Take(t.Context(), port.Task{Question: "change it"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { work.Stop() })
	return work
}

// piece is one delta of a call being written out.
func piece(partial string) string {
	return `{"type":"stream_event","event":{"type":"content_block_delta",` +
		`"delta":{"type":"input_json_delta","partial_json":` + quoted(partial) + `}}}`
}

func quoted(text string) string {
	out := `"`
	for _, r := range text {
		switch r {
		case '"':
			out += `\"`
		case '\\':
			out += `\\`
		default:
			out += string(r)
		}
	}
	return out + `"`
}

const opens = `{"type":"stream_event","event":{"type":"content_block_start",` +
	`"content_block":{"type":"tool_use","id":"toolu_01","name":"mcp__numen__note_edit"}}}`

// The text going in arrives after the text it replaces. A stretch shown with
// nothing in its place reads as having been deleted, so nothing is drawn until
// the replacement has begun.
func TestNothingIsDrawnBeforeTheReplacementHasBegun(t *testing.T) {
	window := &drawn{found: true}
	work := drafting(t, window, opens+"\n"+
		piece(`{"path":"Note.md",`)+"\n"+
		piece(`"stood":"A hedgehog`)+"\n"+
		piece(` is named"`))

	heard(t, work)

	if len(window.said) != 0 {
		t.Fatalf("a change was drawn before its replacement: %+v", window.said)
	}
	if len(window.asked) != 0 {
		t.Errorf("a stretch still arriving was looked for: %v", window.asked)
	}
}

// Once the replacement has begun the stretch it replaces is whole, and every
// piece of the replacement is drawn where that stretch stands.
func TestAChangeIsDrawnAsItsReplacementArrives(t *testing.T) {
	window := &drawn{found: true}
	work := drafting(t, window, opens+"\n"+
		piece(`{"path":"Note.md","stood":"A hedgehog",`)+"\n"+
		piece(`"becomes":"An axe`)+"\n"+
		piece(`, two-bladed"}`))

	heard(t, work)

	if len(window.said) < 2 {
		t.Fatalf("the change was drawn %d times: %+v", len(window.said), window.said)
	}
	first := window.said[0]
	if first.Path != "Note.md" || first.From != 3 || first.To != 9 {
		t.Errorf("the change was drawn as %+v", first)
	}
	if first.Change != "toolu_01" {
		t.Errorf("the change is named %q", first.Change)
	}
	last := window.said[len(window.said)-1]
	if last.Text != "An axe, two-bladed" {
		t.Errorf("what goes in was drawn as %q", last.Text)
	}
	if last.Done {
		t.Error("the adapter ended a change the vault ends")
	}
	if want := []string{"A hedgehog"}; len(window.asked) != 1 || window.asked[0] != want[0] {
		t.Errorf("the stretch was looked for as %v", window.asked)
	}
}

// A stretch that stands nowhere or twice is not a place, and drawing over a
// guess is worse than drawing nothing.
func TestAStretchThatIsNotOnePlaceIsNotDrawn(t *testing.T) {
	window := &drawn{found: true}
	window.found = false
	work := drafting(t, window, opens+"\n"+
		piece(`{"path":"Note.md","stood":"foe",`)+"\n"+
		piece(`"becomes":"friend"}`))

	heard(t, work)

	if len(window.said) != 0 {
		t.Fatalf("a change was drawn over a stretch that stands nowhere: %+v", window.said)
	}
}
