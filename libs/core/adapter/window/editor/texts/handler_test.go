package texts

import (
	"context"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

type shownVault struct{ v domain.Vault }

func (s shownVault) GetCurrentVault() domain.Vault { return s.v }

// fileText is one file's live text, which an edit replaces.
type fileText struct {
	live  domain.VersionText
	isSet bool
	edits []textversion.TextEdit
}

func (f *fileText) ReadLiveText(context.Context, domain.Vault, string) (domain.VersionText, bool, error) {
	return f.live, f.isSet, nil
}

func (f *fileText) CreateEdit(_ context.Context, _ domain.Vault, edit textversion.TextEdit) (domain.TextVersion, error) {
	if edit.Source != f.live.Version.GetID() {
		return domain.TextVersion{}, textversion.ErrNoText
	}
	f.edits = append(f.edits, edit)
	version := domain.TextVersion{Path: edit.Path, PluginID: edit.PluginID, Key: "edit-1", Source: edit.Source, Kind: f.live.Version.Kind}
	f.live = domain.NewVersionText(version, edit.Segments)
	return version, nil
}

var vault = shownVault{v: domain.Vault{ID: "v1"}}

func TestReadLiveTextReturnsTheVersionAndItsSegments(t *testing.T) {
	text := &fileText{isSet: true, live: domain.NewVersionText(
		domain.TextVersion{Path: "a.mp3", PluginID: "p.t", Key: "k", Kind: "transcript"},
		[]domain.TextSegment{{Text: "Hello.\n", Selector: `{"t":[0,1]}`}},
	)}
	res, err := NewHandler(vault, text).ReadLiveText(t.Context(), connect.NewRequest(&v1.ReadLiveTextRequest{Path: "a.mp3"}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetVersion().GetId() != "p.t/k" || len(res.Msg.GetSegments()) != 1 || res.Msg.GetSegments()[0].GetSelector() != `{"t":[0,1]}` {
		t.Fatalf("got %v", res.Msg)
	}
}

func TestReadLiveTextOfAFileWithoutOneIsEmpty(t *testing.T) {
	res, err := NewHandler(vault, &fileText{}).ReadLiveText(t.Context(), connect.NewRequest(&v1.ReadLiveTextRequest{Path: "a.mp3"}))
	if err != nil || res.Msg.GetVersion() != nil {
		t.Fatalf("got %v (%v), want no version", res.Msg, err)
	}
}

func TestCreateTextEditRecordsTheEditForThePlugin(t *testing.T) {
	text := &fileText{isSet: true, live: domain.NewVersionText(domain.TextVersion{PluginID: "p.t", Key: "k"}, nil)}
	h := NewHandler(vault, text)
	res, err := h.CreateTextEdit(t.Context(), connect.NewRequest(&v1.CreateTextEditRequest{
		PluginId: "p.view", Path: "a.mp3", SourceVersionId: "p.t/k",
		Segments: []*v1.TextSegment{{Text: "Hello there.\n", Selector: `{"t":[0,1]}`}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetVersion().GetPluginId() != "p.view" || len(text.edits) != 1 || text.edits[0].Segments[0].Selector != `{"t":[0,1]}` {
		t.Fatalf("got %v and edits %+v", res.Msg, text.edits)
	}
	missing, err := h.CreateTextEdit(t.Context(), connect.NewRequest(&v1.CreateTextEditRequest{PluginId: "p.view", Path: "a.mp3", SourceVersionId: "x/y"}))
	if err != nil || missing.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_MISSING {
		t.Fatalf("an edit of a version that is not there: %v (%v)", missing.Msg, err)
	}
}

func TestCreateTextEditWithNoVaultIsAFailedPrecondition(t *testing.T) {
	_, err := NewHandler(shownVault{}, &fileText{}).CreateTextEdit(t.Context(), connect.NewRequest(&v1.CreateTextEditRequest{}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("got %v", err)
	}
}
