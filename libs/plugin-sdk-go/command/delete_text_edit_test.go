package command

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/hosttest"
)

// An edit saved in a viewer is a version of the viewer: the handler of the plugin that made its
// source refuses it, and the viewer's own handler deletes it.
func TestNewDeleteTextRunDeletesAnEditOnlyInThePluginThatMadeIt(t *testing.T) {
	ctx := t.Context()
	const transcriber, viewer = "org.example.transcriber", "org.example.viewer"
	file := []byte("RIFF")
	texts := hosttest.NewTexts("v1", transcriber)
	texts.File("a.wav", file)
	store := texts.GetStore(transcriber)
	if err := host.WriteText(ctx, store, "k.jsonl", []domain.TextSegment{{Text: "the tide"}}); err != nil {
		t.Fatal(err)
	}
	made, err := texts.PutVersion(ctx, host.TextPut{Path: "a.wav", Key: "k", Kind: "transcript", InputHash: domain.HashContent(file), File: "k.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	edits, edited := texts.As(viewer), texts.GetStore(viewer)
	if err := host.WriteText(ctx, edited, "edit-1.jsonl", []domain.TextSegment{{Text: "the tide turns"}}); err != nil {
		t.Fatal(err)
	}
	edit, err := edits.PutVersion(ctx, host.TextPut{Path: "a.wav", Key: "edit-1", Source: made.GetID(), Kind: "transcript", InputHash: made.TextHash, File: "edit-1.jsonl"})
	if err != nil {
		t.Fatal(err)
	}
	request := Request{Command: "app:version.delete", Path: "a.wav", Arguments: Arguments{ArgumentVersion: string(edit.GetID())}}

	if _, err := NewDeleteTextRun(texts, store, transcriber)(ctx, request); !errors.Is(err, host.ErrNotOwner) {
		t.Fatalf("the transcriber's handler deleting the edit %s: got %v, want ErrNotOwner", edit.GetID(), err)
	}
	if _, err := NewDeleteTextRun(edits, edited, viewer)(ctx, request); err != nil {
		t.Fatalf("the viewer's handler deleting its edit %s: %v", edit.GetID(), err)
	}
	listing, err := texts.ListVersions(ctx, "a.wav")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range listing.Versions {
		if one.GetID() == edit.GetID() {
			t.Fatalf("the edit is still listed: %+v", listing.Versions)
		}
	}
	if _, err := edited.Read(ctx, "edit-1.jsonl"); !errors.Is(err, host.ErrNotExist) {
		t.Errorf("the file of the edit is still kept: %v", err)
	}
}
