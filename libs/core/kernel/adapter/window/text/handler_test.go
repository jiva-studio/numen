package text

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/textfake"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

type shownVault struct{ v domain.Vault }

func (s shownVault) GetCurrentVault() domain.Vault { return s.v }

// scene is a vault of files, the text versions the plugins recorded of them, and the handler that
// answers for the vault the window shows.
type scene struct {
	vault    domain.Vault
	files    textfake.Readers
	folders  *textfake.Folders
	versions *textfake.TextVersions
	handler  *Handler
}

func newScene(t *testing.T, files textfake.Readers) *scene {
	t.Helper()
	s := &scene{vault: domain.Vault{ID: "v1", Name: "v1"}, files: files, folders: &textfake.Folders{}}
	versions, err := textfake.NewTextVersions(
		&textfake.Versions{}, s.folders, textfake.Plugins{"p.t", "p.link"}, files, textfake.NewSteppingClock(),
	)
	if err != nil {
		t.Fatal(err)
	}
	fileText, err := textversion.NewFileText(versions.Texts, versions.Versions)
	if err != nil {
		t.Fatal(err)
	}
	s.versions = versions
	s.handler = NewHandler(shownVault{v: s.vault}, fileText)
	return s
}

func (s *scene) record(t *testing.T, pluginID, path string, report plugin.TextPut, segments []domain.TextSegment) {
	t.Helper()
	report.Path = path
	report.File = report.Key + ".jsonl"
	if _, err := s.folders.WriteTextFile(t.Context(), pluginID, report.File, segments); err != nil {
		t.Fatal(err)
	}
	if _, err := s.versions.PutVersion(t.Context(), s.vault, pluginID, report); err != nil {
		t.Fatal(err)
	}
}

func TestReadLiveTextReturnsTheVersionAndItsSegments(t *testing.T) {
	s := newScene(t, textfake.Readers{"a.mp3": []byte("audio")})
	s.record(t, "p.t", "a.mp3",
		plugin.TextPut{Key: "k", Kind: "transcript", InputHash: domain.HashContent([]byte("audio"))},
		[]domain.TextSegment{{Text: "Hello.\n", Selector: `{"t":[0,1]}`}})

	res, err := s.handler.ReadLiveText(t.Context(), connect.NewRequest(&v1.ReadLiveTextRequest{Path: "a.mp3"}))

	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetVersion().GetId() != "p.t/k" || len(res.Msg.GetSegments()) != 1 || res.Msg.GetSegments()[0].GetSelector() != `{"t":[0,1]}` {
		t.Fatalf("got %v", res.Msg)
	}
}

func TestReadLiveTextReturnsTheNotePropertyOfTheVersion(t *testing.T) {
	s := newScene(t, textfake.Readers{"talk.md": []byte("---\nurl: https://example.org/talk\n---\n# Talk\n")})
	s.record(t, "p.link", "talk.md",
		plugin.TextPut{
			Key: "link", Kind: "link",
			SourceProperty: domain.TextSourceProperty{Key: "url", Value: "https://example.org/talk"},
		},
		[]domain.TextSegment{{Text: "Hello.\n"}})

	res, err := s.handler.ReadLiveText(t.Context(), connect.NewRequest(&v1.ReadLiveTextRequest{Path: "talk.md"}))

	if err != nil {
		t.Fatal(err)
	}
	property := res.Msg.GetVersion().GetSourceProperty()
	if property.GetKey() != "url" || property.GetValue() != "https://example.org/talk" {
		t.Fatalf("got source property %v, want url = https://example.org/talk", property)
	}
}

func TestReadLiveTextOfAFileWithoutOneIsEmpty(t *testing.T) {
	s := newScene(t, textfake.Readers{"a.mp3": []byte("audio")})

	res, err := s.handler.ReadLiveText(t.Context(), connect.NewRequest(&v1.ReadLiveTextRequest{Path: "a.mp3"}))

	if err != nil || res.Msg.GetVersion() != nil {
		t.Fatalf("got %v (%v), want no version", res.Msg, err)
	}
}

func TestCreateTextEditRecordsTheEditForThePlugin(t *testing.T) {
	s := newScene(t, textfake.Readers{"a.mp3": []byte("audio")})
	s.record(t, "p.t", "a.mp3",
		plugin.TextPut{Key: "k", Kind: "transcript", InputHash: domain.HashContent([]byte("audio"))},
		[]domain.TextSegment{{Text: "Hello.\n"}})

	res, err := s.handler.CreateTextEdit(t.Context(), connect.NewRequest(&v1.CreateTextEditRequest{
		PluginId: "p.view", Path: "a.mp3", SourceVersionId: "p.t/k",
		Segments: []*v1.TextSegment{{Text: "Hello there.\n", Selector: `{"t":[0,1]}`}},
	}))

	if err != nil {
		t.Fatal(err)
	}
	if res.Msg.GetVersion().GetPluginId() != "p.view" {
		t.Fatalf("the edit was recorded for %q", res.Msg.GetVersion().GetPluginId())
	}
	live, err := s.handler.ReadLiveText(t.Context(), connect.NewRequest(&v1.ReadLiveTextRequest{Path: "a.mp3"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(live.Msg.GetSegments()) != 1 || live.Msg.GetSegments()[0].GetText() != "Hello there.\n" ||
		live.Msg.GetSegments()[0].GetSelector() != `{"t":[0,1]}` {
		t.Fatalf("the live text after the edit is %v", live.Msg.GetSegments())
	}
}

func TestCreateTextEditOfAVersionThatIsNotThereIsMissing(t *testing.T) {
	s := newScene(t, textfake.Readers{"a.mp3": []byte("audio")})

	missing, err := s.handler.CreateTextEdit(t.Context(), connect.NewRequest(
		&v1.CreateTextEditRequest{PluginId: "p.view", Path: "a.mp3", SourceVersionId: "x/y"}))

	if err != nil || missing.Msg.GetError() != v1.TextErrorCode_TEXT_ERROR_CODE_MISSING {
		t.Fatalf("an edit of a version that is not there: %v (%v)", missing.Msg, err)
	}
}

func TestCreateTextEditWithNoVaultIsAFailedPrecondition(t *testing.T) {
	s := newScene(t, textfake.Readers{})
	s.handler = NewHandler(shownVault{}, s.handler.text)

	_, err := s.handler.CreateTextEdit(t.Context(), connect.NewRequest(&v1.CreateTextEditRequest{}))

	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("got %v", err)
	}
}
