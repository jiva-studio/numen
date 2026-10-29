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

func TestGetFileMenuReturnsEntries(t *testing.T) {
	m := &menu{entries: []textversion.MenuEntry{
		{
			PluginID: "md.numen.tools.proofreading", PluginName: "LLM Proofreading", Command: "proofread", Title: "Proofread",
			Icon: "spell-check", Version: "md.numen.engines.ocr/recognition", IsLive: true,
		},
		{
			PluginID: "md.numen.engines.ocr", PluginName: "Document OCR", Command: "delete-recognition",
			Title: "Delete recognition", Version: "md.numen.engines.ocr/recognition",
		},
	}}
	h := NewHandler(vault, m)

	res, err := h.GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "scan.pdf"}))
	if err != nil {
		t.Fatal(err)
	}
	got := res.Msg.GetEntries()
	if len(got) != 2 ||
		got[0].GetPluginName() != "LLM Proofreading" || got[0].GetCommand() != "proofread" || got[0].GetTitle() != "Proofread" ||
		got[0].GetIcon() != "spell-check" || got[0].GetVersionId() != "md.numen.engines.ocr/recognition" || !got[0].GetIsLive() ||
		got[1].GetCommand() != "delete-recognition" || got[1].GetTitle() != "Delete recognition" || got[1].GetIsLive() {
		t.Fatalf("answered %v", got)
	}
	if _, err := h.GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "gone.pdf"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Errorf("a file that is not there was answered %v", err)
	}
}

func TestFileMenuWithoutVaultIsEmpty(t *testing.T) {
	m := &menu{entries: []textversion.MenuEntry{{PluginID: "p", Command: "proofread"}}}
	h := NewHandler(shown{}, m)

	res, err := h.GetFileMenu(t.Context(), connect.NewRequest(&v1.GetFileMenuRequest{Path: "scan.pdf"}))
	if err != nil || len(res.Msg.GetEntries()) != 0 {
		t.Errorf("answered %v (%v), want nothing", res.Msg.GetEntries(), err)
	}
	_, err = h.RunFileMenuEntry(t.Context(), connect.NewRequest(&v1.RunFileMenuEntryRequest{Path: "scan.pdf", PluginId: "p", Command: "proofread"}))
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
		Path: "scan.pdf", PluginId: "p", Command: "proofread", VersionId: "q/k",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := textversion.MenuEntry{PluginID: "p", Command: "proofread", Version: "q/k"}
	if len(m.ran) != 1 || m.ran[0] != want {
		t.Fatalf("ran %+v, want %+v", m.ran, want)
	}
}

func TestRunFileMenuEntryMapsErrors(t *testing.T) {
	for _, why := range []error{textversion.ErrInvalidVersion, textversion.ErrSourceNotAccepted, textversion.ErrNotTextPlugin, textversion.ErrNoText} {
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
