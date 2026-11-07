package service_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/service"
)

// cancelledBatchVault is the geology vault whose host returns a move or a removal as a call that was
// cancelled after the first note: the first note is changed, every other is not attempted.
type cancelledBatchVault struct {
	*hosttest.Vault
}

func (c cancelledBatchVault) With(domain.VaultID) host.Vault { return c }

func (c cancelledBatchVault) MoveNotes(ctx context.Context, paths []string, folder string) ([]host.NoteMove, error) {
	out := make([]host.NoteMove, 0, len(paths))
	for i, from := range paths {
		if i == 0 {
			to := folder + "/" + from[strings.LastIndex(from, "/")+1:]
			out = append(out, host.NoteMove{Move: domain.MoveResult{From: from, To: to, IsMoved: true, Repaired: []string{"Minerals.md"}}})
			continue
		}
		out = append(out, host.NoteMove{
			Move: domain.MoveResult{From: from},
			Err:  fmt.Errorf("the call was cancelled before %s was moved: %w", from, host.ErrNotAttempted),
		})
	}
	return out, nil
}

func (c cancelledBatchVault) RemoveNotes(ctx context.Context, paths []string) ([]host.NoteRemoval, error) {
	out := make([]host.NoteRemoval, 0, len(paths))
	for i, target := range paths {
		if i == 0 {
			out = append(out, host.NoteRemoval{Path: target, TrashPath: ".trash/" + target, BrokenLinks: []string{"Minerals.md"}})
			continue
		}
		out = append(out, host.NoteRemoval{
			Path: target,
			Err:  fmt.Errorf("the call was cancelled before %s was removed: %w", target, host.ErrNotAttempted),
		})
	}
	return out, nil
}

func newCancelledBatchRegistry(t *testing.T) *tools.Registry {
	t.Helper()
	v, _ := newGeologyVault(t)
	registry, err := service.New(cancelledBatchVault{v}).Registry()
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	return registry
}

func TestCancelledNoteMoveReportsCompletedAndNotAttemptedNotes(t *testing.T) {
	registry := newCancelledBatchRegistry(t)

	var out noteMoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_move",
		`{"paths":["Rocks/Basalt.md","Rocks/Granite.md","Fossils.md"],"folder":"Archive"}`), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d moves, want 3: %+v", len(out.Results), out.Results)
	}
	if completed := out.Results[0]; completed.From != "Rocks/Basalt.md" || completed.Path != "Archive/Basalt.md" || completed.Error != "" {
		t.Errorf("the first entry is %+v, want Rocks/Basalt.md moved to Archive/Basalt.md with no error", completed)
	}
	for i, from := range []string{"Rocks/Granite.md", "Fossils.md"} {
		got := out.Results[i+1]
		want := "the call was cancelled before " + from + " was moved"
		if got.From != from || got.Path != "" || !strings.Contains(got.Error, want) {
			t.Errorf("entry %d is %+v, want %s not moved with an error containing %q", i+1, got, from, want)
		}
	}
}

func TestCancelledNoteRemoveReportsCompletedAndNotAttemptedNotes(t *testing.T) {
	registry := newCancelledBatchRegistry(t)

	var out noteRemoveOutput
	decodeInto(t, callTool(t, registry, geologyVault, "note_remove",
		`{"paths":["Fossils.md","Rocks/Granite.md","Rocks/Basalt.md"]}`), &out)
	if len(out.Results) != 3 {
		t.Fatalf("got %d removals, want 3: %+v", len(out.Results), out.Results)
	}
	if completed := out.Results[0]; completed.Path != "Fossils.md" || completed.TrashPath != ".trash/Fossils.md" || completed.Error != "" {
		t.Errorf("the first entry is %+v, want Fossils.md in .trash with no error", completed)
	}
	for i, target := range []string{"Rocks/Granite.md", "Rocks/Basalt.md"} {
		got := out.Results[i+1]
		want := "the call was cancelled before " + target + " was removed"
		if got.Path != target || got.TrashPath != "" || !strings.Contains(got.Error, want) {
			t.Errorf("entry %d is %+v, want %s not removed with an error containing %q", i+1, got, target, want)
		}
	}
}
