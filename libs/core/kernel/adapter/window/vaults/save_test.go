package vaults

import (
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"
)

func TestAPersonWhoChoseAFileHasTheTextSavedThere(t *testing.T) {
	f := onAList(t)
	path := filepath.Join(t.TempDir(), "account.json")
	f.dialog.pick, f.dialog.isChosen = path, true

	out, err := f.dialogs.SaveTextFile(t.Context(), connect.NewRequest(&v1.SaveTextFileRequest{
		FileName: "account.json", Extension: ".json", Content: `{"vaults":[]}`,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if !out.Msg.GetIsSaved() || out.Msg.GetPath() != path {
		t.Errorf("the answer is %v", out.Msg)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != `{"vaults":[]}` {
		t.Errorf("the file holds %q, %v", got, err)
	}
	if f.dialog.proposed != "account.json" {
		t.Errorf("the dialog proposed %q", f.dialog.proposed)
	}
}

func TestAPersonWhoClosedTheSaveDialogSavedNothing(t *testing.T) {
	f := onAList(t)

	out, err := f.dialogs.SaveTextFile(t.Context(), connect.NewRequest(&v1.SaveTextFileRequest{Content: "x"}))
	if err != nil {
		t.Fatalf("a dialog that was closed answered %v", err)
	}
	if out.Msg.GetIsSaved() || out.Msg.GetPath() != "" {
		t.Errorf("the answer is %v", out.Msg)
	}
}
