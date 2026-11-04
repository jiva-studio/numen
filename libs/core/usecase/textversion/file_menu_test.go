package textversion_test

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"reflect"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
)

// fileReaders opens a vault whose files are the map, as they are when read.
type fileReaders map[string][]byte

func (r fileReaders) Open(domain.Vault) (vaultport.VaultReader, error) {
	return fileReader{files: r}, nil
}

type fileReader struct {
	vaultport.VaultReader
	files fileReaders
}

func (r fileReader) Read(_ context.Context, path string) ([]byte, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return raw, nil
}

// readCounter counts the files read through the readers it wraps.
type readCounter struct {
	fileReaders
	reads *int
}

func (r readCounter) Open(v domain.Vault) (vaultport.VaultReader, error) {
	inner, err := r.fileReaders.Open(v)
	return countingReader{VaultReader: inner, reads: r.reads}, err
}

type countingReader struct {
	vaultport.VaultReader
	reads *int
}

func (r countingReader) Read(ctx context.Context, path string) ([]byte, error) {
	*r.reads++
	return r.VaultReader.Read(ctx, path)
}

// commands offers the commands it holds, and records each run.
type commands struct {
	all []pluginport.PluginCommandEntry
	ran []pluginport.PluginCommandCall
}

func (c *commands) ListCommands() []pluginport.PluginCommandEntry { return c.all }

func (c *commands) RunCommand(_ context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error) {
	c.ran = append(c.ran, call)
	return pluginport.PluginCommandResult{}, nil
}

// types gives a file its type by the extension.
type types map[string]domain.ContentType

func (t types) GetContentType(name string) domain.ContentType {
	if one, ok := t[path.Ext(name)]; ok {
		return one
	}
	return domain.UnknownContentType
}

func (t types) ListExtensions(domain.ContentType) []string { return nil }

func getPdfTypes() types { return types{".pdf": "application/pdf"} }

func getRecogniseCommand() pluginport.PluginCommandEntry {
	return pluginport.PluginCommandEntry{PluginID: ocr, PluginName: "OCR", Command: pluginport.PluginCommand{
		ID: "recognition.create", Title: "Recognise text", Icon: "scan-text", ContentTypes: []domain.ContentType{"application/pdf"},
	}}
}

func getProofreadCommand() pluginport.PluginCommandEntry {
	return pluginport.PluginCommandEntry{PluginID: proofreader, PluginName: "Proofreading", Command: pluginport.PluginCommand{
		ID: "proofreading.create", Title: "Proofread", Icon: "spell-check",
		When: liveOfKinds(string(kindPage), string(kindRecognition)),
	}}
}

func getDeleteProofreadingCommand() pluginport.PluginCommandEntry {
	return pluginport.PluginCommandEntry{PluginID: proofreader, PluginName: "Proofreading", Command: pluginport.PluginCommand{
		ID: "proofreading.delete", Title: "Delete proofreading", When: ownedBy(proofreader),
	}}
}

// liveOfKinds holds in the context of the live version when it is of one of the kinds.
func liveOfKinds(kinds ...string) []pluginport.PluginContextCondition {
	return []pluginport.PluginContextCondition{
		{Key: textversion.ContextKeyLive, Is: []string{"true"}},
		{Key: textversion.ContextKeyKind, Is: kinds},
	}
}

// ownedBy holds in the context of a version the plugin made.
func ownedBy(pluginID string) []pluginport.PluginContextCondition {
	return []pluginport.PluginContextCondition{{Key: textversion.ContextKeyPlugin, Is: []string{pluginID}}}
}

func newMenu(t *testing.T, s *scene, entries ...pluginport.PluginCommandEntry) (*textversion.FileMenu, *commands) {
	t.Helper()
	recorder := &commands{all: entries}
	menu, err := textversion.NewFileMenu(s.versions, recorder, getPdfTypes(), fileReaders(s.files))
	if err != nil {
		t.Fatal(err)
	}
	return menu, recorder
}

func TestFileMenuListsCommandForDeclaredKind(t *testing.T) {
	s := newScene(t)
	menu, _ := newMenu(t, s, getProofreadCommand())
	s.requestText(t)

	entries, err := menu.GetMenu(t.Context(), s.vault, scanPath)
	want := []textversion.MenuEntry{{
		PluginID: proofreader, PluginName: "Proofreading", Command: "proofreading.create", Title: "Proofread", Icon: "spell-check",
		Version: s.getReaderVersion(), IsLive: true,
	}}
	if err != nil || !slices.Equal(entries, want) {
		t.Fatalf("a file with a live %s version gets %+v (%v), want %+v", kindPage, entries, err, want)
	}

	s.files[scanPath] = []byte("%PDF scan, second printing")
	entries, err = menu.GetMenu(t.Context(), s.vault, scanPath)
	if err != nil || len(entries) != 0 {
		t.Fatalf("a file replaced since its version was made gets %+v (%v), want nothing", entries, err)
	}
}

func TestFileMenuOmitsCommandForUndeclaredKind(t *testing.T) {
	s := newScene(t)
	command := getProofreadCommand()
	command.Command.When = liveOfKinds(string(kindRecognition))
	menu, _ := newMenu(t, s, command)
	s.requestText(t)

	if entries, err := menu.GetMenu(t.Context(), s.vault, scanPath); err != nil || len(entries) != 0 {
		t.Fatalf("a %s version gets %+v (%v) to a command of %s", kindPage, entries, err, kindRecognition)
	}
}

func TestFileMenuAcceptsAnyKindWhereTheCommandNamesAll(t *testing.T) {
	s := newScene(t)
	command := getProofreadCommand()
	command.Command.When = liveOfKinds(pluginport.ContextAnyValue)
	menu, _ := newMenu(t, s, command)
	s.requestText(t)

	if entries, err := menu.GetMenu(t.Context(), s.vault, scanPath); err != nil || len(entries) != 1 {
		t.Fatalf("a command of any kind gets %+v (%v)", entries, err)
	}
}

func TestFileMenuListsDeletionForEachPluginVersion(t *testing.T) {
	s := newScene(t)
	menu, _ := newMenu(t, s, getDeleteProofreadingCommand())
	s.requestText(t)
	s.runEntry(t, proofreader)
	b := getProofreadingOf(s.getReaderVersion())

	entries, err := menu.GetMenu(t.Context(), s.vault, scanPath)
	want := []textversion.MenuEntry{{
		PluginID: proofreader, PluginName: "Proofreading", Command: "proofreading.delete", Title: "Delete proofreading",
		Version: b, IsLive: true,
	}}
	if err != nil || !slices.Equal(entries, want) {
		t.Fatalf("the deletions are %+v (%v), want %+v", entries, err, want)
	}
}

func TestFileMenuMatchesLiveVersionToCommandOwner(t *testing.T) {
	s := newScene(t)
	command := getDeleteProofreadingCommand()
	command.Command.When = append(liveOfKinds(string(kindPage), string(kindRecognition)), ownedBy(proofreader)...)
	menu, _ := newMenu(t, s, command)
	s.requestText(t)

	if entries, err := menu.GetMenu(t.Context(), s.vault, scanPath); err != nil || len(entries) != 0 {
		t.Fatalf("a live version of the reader gets %+v (%v) to the proofreader's deletion", entries, err)
	}
	s.runEntry(t, proofreader)
	if entries, err := menu.GetMenu(t.Context(), s.vault, scanPath); err != nil || len(entries) != 1 {
		t.Fatalf("a live version of the proofreader gets %+v (%v)", entries, err)
	}
}

func TestFileMenuWithoutVersionListsTypeCommandsUnread(t *testing.T) {
	s := newScene(t)
	reads := 0
	recorder := &commands{all: []pluginport.PluginCommandEntry{getRecogniseCommand(), getProofreadCommand(), getDeleteProofreadingCommand()}}
	menu, err := textversion.NewFileMenu(s.versions, recorder, getPdfTypes(), readCounter{fileReaders: s.files, reads: &reads})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := menu.GetMenu(t.Context(), s.vault, scanPath)
	want := []textversion.MenuEntry{{PluginID: ocr, PluginName: "OCR", Command: "recognition.create", Title: "Recognise text", Icon: "scan-text"}}
	if err != nil || !slices.Equal(entries, want) || reads != 0 {
		t.Fatalf("a file with no version gets %+v (%v) after %d reads, want %+v and no read", entries, err, reads, want)
	}
	s.requestText(t)
	if _, err := menu.GetMenu(t.Context(), s.vault, scanPath); err != nil || reads != 1 {
		t.Fatalf("a file with a version was read %d times (%v), want once", reads, err)
	}
}

func TestFileMenuOmitsCommandForOtherTypeOrArgument(t *testing.T) {
	s := newScene(t)
	needsArgument := getRecogniseCommand()
	needsArgument.Command.Arguments = []pluginport.PluginCommandArgument{{Name: "address", Title: "Address", Type: pluginport.CommandArgumentString, IsRequired: true}}
	menu, _ := newMenu(t, s, getRecogniseCommand(), needsArgument)
	s.files["notes/a.txt"] = []byte("plain")

	if entries, err := menu.GetMenu(t.Context(), s.vault, "notes/a.txt"); err != nil || len(entries) != 0 {
		t.Fatalf("a text file gets %+v (%v)", entries, err)
	}
	if entries, err := menu.GetMenu(t.Context(), s.vault, scanPath); err != nil || len(entries) != 1 || entries[0].Command != "recognition.create" {
		t.Fatalf("a PDF gets %+v (%v), want the command that needs no argument", entries, err)
	}
}

func TestFileMenuOfAMissingFileIsNotFound(t *testing.T) {
	s := newScene(t)
	menu, _ := newMenu(t, s, getProofreadCommand())
	s.requestText(t)
	delete(s.files, scanPath)

	if _, err := menu.GetMenu(t.Context(), s.vault, scanPath); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a file that is gone was given a menu: %v", err)
	}
}

func TestFileMenuRunsListedEntryCommand(t *testing.T) {
	s := newScene(t)
	menu, recorder := newMenu(t, s, getProofreadCommand())
	s.requestText(t)
	entry := textversion.MenuEntry{PluginID: proofreader, Command: "proofreading.create", Version: s.getReaderVersion()}

	if err := menu.RunMenuEntry(t.Context(), s.vault, scanPath, entry); err != nil {
		t.Fatal(err)
	}
	want := []pluginport.PluginCommandCall{{
		PluginID: proofreader, Command: "proofreading.create", Path: scanPath,
		Arguments: map[string]string{pluginport.CommandArgumentVersion: string(s.getReaderVersion())},
	}}
	if !reflect.DeepEqual(recorder.ran, want) {
		t.Fatalf("ran %+v, want %+v", recorder.ran, want)
	}
}

func TestFileMenuRunsACommandOverTheFileWithNoVersionArgument(t *testing.T) {
	s := newScene(t)
	menu, recorder := newMenu(t, s, getRecogniseCommand())

	if err := menu.RunMenuEntry(t.Context(), s.vault, scanPath, textversion.MenuEntry{PluginID: ocr, Command: "recognition.create"}); err != nil {
		t.Fatal(err)
	}
	want := []pluginport.PluginCommandCall{{PluginID: ocr, Command: "recognition.create", Path: scanPath}}
	if !reflect.DeepEqual(recorder.ran, want) {
		t.Fatalf("ran %+v, want %+v", recorder.ran, want)
	}
}

func TestFileMenuRejectsRunningUnofferedEntry(t *testing.T) {
	s := newScene(t)
	menu, recorder := newMenu(t, s, getProofreadCommand())
	s.requestText(t)

	for name, entry := range map[string]textversion.MenuEntry{
		"another command": {PluginID: proofreader, Command: "proofreading.delete", Version: s.getReaderVersion()},
		"another version": {PluginID: proofreader, Command: "proofreading.create", Version: "org.numen.tools-proofreading/other"},
		"another plugin":  {PluginID: ocr, Command: "proofreading.create", Version: s.getReaderVersion()},
	} {
		if err := menu.RunMenuEntry(t.Context(), s.vault, scanPath, entry); !errors.Is(err, textversion.ErrInvalidVersion) {
			t.Errorf("%s: got %v, want ErrInvalidVersion", name, err)
		}
	}
	if len(recorder.ran) != 0 {
		t.Fatalf("ran %+v", recorder.ran)
	}
}

func TestNewFileMenuRequiresVersionsCommandsTypesAndReaders(t *testing.T) {
	s := newScene(t)
	if _, err := textversion.NewFileMenu(nil, &commands{}, getPdfTypes(), fileReaders{}); err == nil {
		t.Error("a file menu was made without versions")
	}
	if _, err := textversion.NewFileMenu(s.versions, nil, getPdfTypes(), fileReaders{}); err == nil {
		t.Error("a file menu was made without commands")
	}
	if _, err := textversion.NewFileMenu(s.versions, &commands{}, nil, fileReaders{}); err == nil {
		t.Error("a file menu was made without a content type table")
	}
	if _, err := textversion.NewFileMenu(s.versions, &commands{}, getPdfTypes(), nil); err == nil {
		t.Error("a file menu was made without readers")
	}
}
