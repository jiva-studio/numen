package textmenu

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

type shown domain.Vault

func (s shown) GetCurrentVault() domain.Vault { return domain.Vault(s) }

// menu offers the entries it holds for every file but a missing one, and records each run.
type menu struct {
	entries []textversion.MenuEntry
	ran     []textversion.MenuEntry
	runErr  error
}

func (m *menu) GetMenu(_ context.Context, _ domain.Vault, path string) ([]textversion.MenuEntry, error) {
	if path == "gone.pdf" {
		return nil, fmt.Errorf("read %s: %w", path, fs.ErrNotExist)
	}
	return m.entries, nil
}

func (m *menu) RunMenuEntry(_ context.Context, _ domain.Vault, _ string, entry textversion.MenuEntry) error {
	m.ran = append(m.ran, entry)
	return m.runErr
}

var vault = shown{ID: "v1", Name: "Physics"}

func TestActionsCoverEveryFileMenuAction(t *testing.T) {
	mapped := map[v1.FileMenuAction]bool{}
	for _, wire := range actions {
		mapped[wire] = true
	}
	for value := range v1.FileMenuAction_name {
		action := v1.FileMenuAction(value)
		if action != v1.FileMenuAction_FILE_MENU_ACTION_UNSPECIFIED && !mapped[action] {
			t.Errorf("%s is mapped for no entry of the menu", action)
		}
	}
}

func TestGetFileMenuReturnsEntries(t *testing.T) {
	m := &menu{entries: []textversion.MenuEntry{
		{Action: textversion.ActionRecognise, PluginID: "md.numen.engines.ocr", PluginName: "Document OCR"},
		{Action: textversion.ActionDeleteVersion, PluginID: "md.numen.engines.ocr", PluginName: "Document OCR", Version: "md.numen.engines.ocr/recognition"},
	}}
	h := NewHandler(vault, m)

	res, err := h.GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "scan.pdf"}))
	if err != nil {
		t.Fatal(err)
	}
	got := res.Msg.GetEntries()
	if len(got) != 2 ||
		got[0].GetAction() != v1.FileMenuAction_FILE_MENU_ACTION_RECOGNISE || got[0].GetPluginName() != "Document OCR" ||
		got[1].GetAction() != v1.FileMenuAction_FILE_MENU_ACTION_DELETE_VERSION || got[1].GetVersionId() != "md.numen.engines.ocr/recognition" {
		t.Fatalf("answered %v", got)
	}
	if _, err := h.GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "gone.pdf"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a file that is not there was answered %v", err)
	}
}

func TestFileMenuWithoutVaultIsEmpty(t *testing.T) {
	m := &menu{entries: []textversion.MenuEntry{{Action: textversion.ActionRecognise, PluginID: "p"}}}
	h := NewHandler(shown{}, m)

	res, err := h.GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "scan.pdf"}))
	if err != nil || len(res.Msg.GetEntries()) != 0 {
		t.Errorf("answered %v (%v), want nothing", res.Msg.GetEntries(), err)
	}
	_, err = h.RunFileMenuEntry(t.Context(), connect.NewRequest(&v1.RunFileMenuEntryRequest{Path: "scan.pdf", Action: v1.FileMenuAction_FILE_MENU_ACTION_RECOGNISE, PluginId: "p"}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || len(m.ran) != 0 {
		t.Errorf("a run was answered %v and ran %v", err, m.ran)
	}
	if res, err := NewHandler(vault, nil).GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "scan.pdf"})); err != nil || len(res.Msg.GetEntries()) != 0 {
		t.Errorf("a window with no menu answered %v (%v)", res.Msg.GetEntries(), err)
	}
}

func TestRunFileMenuEntryPassesEntry(t *testing.T) {
	m := &menu{}
	h := NewHandler(vault, m)

	_, err := h.RunFileMenuEntry(t.Context(), connect.NewRequest(&v1.RunFileMenuEntryRequest{
		Path: "scan.pdf", Action: v1.FileMenuAction_FILE_MENU_ACTION_PROOFREAD, PluginId: "p", VersionId: "q/k",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := textversion.MenuEntry{Action: textversion.ActionProofread, PluginID: "p", Version: "q/k"}
	if len(m.ran) != 1 || m.ran[0] != want {
		t.Fatalf("ran %+v, want %+v", m.ran, want)
	}
}

func TestRunFileMenuEntryMapsErrors(t *testing.T) {
	for _, why := range []error{textversion.ErrInvalidVersion, textversion.ErrInputNotAccepted, textversion.ErrNotTextPlugin, textversion.ErrNoText} {
		h := NewHandler(vault, &menu{runErr: fmt.Errorf("run: %w", why)})
		_, err := h.RunFileMenuEntry(t.Context(), connect.NewRequest(&v1.RunFileMenuEntryRequest{Path: "scan.pdf"}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Errorf("%v was answered %v", why, err)
		}
	}
	h := NewHandler(vault, &menu{runErr: errors.New("the plugin did not answer")})
	if _, err := h.RunFileMenuEntry(t.Context(), connect.NewRequest(&v1.RunFileMenuEntryRequest{Path: "scan.pdf"})); connect.CodeOf(err) != connect.CodeInternal {
		t.Errorf("a failed run was answered %v", err)
	}
}
