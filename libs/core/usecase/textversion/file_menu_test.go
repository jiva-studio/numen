package textversion_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"path"
	"reflect"
	"strings"
	"testing"
	"time"

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

func (r fileReader) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	raw, ok := r.files[path]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return memoryFile{bytes.NewReader(raw)}, nil
}

// StatFile gives every file of the map the same modification time.
func (r fileReader) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	raw, ok := r.files[path]
	if !ok {
		return domain.FileStat{}, fs.ErrNotExist
	}
	return domain.FileStat{Path: path, Kind: domain.KindFile, Size: int64(len(raw)), ModTime: getFilesTime()}, nil
}

func getFilesTime() time.Time { return time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC) }

// memoryFile is a file stored in memory, with nothing to release.
type memoryFile struct{ *bytes.Reader }

func (memoryFile) Close() error { return nil }

// readCounter counts the files read or opened through the readers it wraps.
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

func (r countingReader) Open(ctx context.Context, path string) (io.ReadSeekCloser, error) {
	*r.reads++
	return r.VaultReader.Open(ctx, path)
}

// commands offers the commands it holds, and records each run.
type commands struct {
	all []pluginport.PluginCommandEntry
	ran []pluginport.PluginCommandCall
}

func (c *commands) ListSlots() []pluginport.PluginSlotEntry { return nil }

func (c *commands) ListHandlers() []pluginport.PluginCommandEntry { return nil }

func (c *commands) ListCommands() []pluginport.PluginCommandEntry { return c.all }

func (c *commands) GetParts(string) pluginport.PluginParts {
	return pluginport.PluginParts{HasBackend: true}
}

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
		ID: "recognition.create", Title: "Recognise text", Icon: "scan-text", When: "target.contentType == 'application/pdf'",
	}}
}

func getProofreadCommand() pluginport.PluginCommandEntry {
	return pluginport.PluginCommandEntry{PluginID: proofreader, PluginName: "Proofreading", Command: pluginport.PluginCommand{
		ID: "proofreading.create", Title: "Proofread", Icon: "spell-check",
		When: liveOfKinds(string(kindPage), string(kindRecognition)),
	}}
}

// liveOfKinds holds over the live version when it is of one of the kinds.
func liveOfKinds(kinds ...string) string {
	return "version.isLive && version.kind in ['" + strings.Join(kinds, "', '") + "']"
}

func newMenu(t *testing.T, s *scene, entries ...pluginport.PluginCommandEntry) (*textversion.FileMenu, *commands) {
	t.Helper()
	recorder := &commands{all: entries}
	menu, err := textversion.NewFileMenu(s.versions, recorder, getPdfTypes())
	if err != nil {
		t.Fatal(err)
	}
	return menu, recorder
}

func TestFileMenuRunsACommandOverAVersionItemWithTheVersionArgument(t *testing.T) {
	s := newScene(t)
	proofread := getProofreadCommand()
	menu, recorder := newMenu(t, s, proofread)
	s.requestText(t)

	if _, err := menu.RunVersionItem(t.Context(), s.vault, scanPath, proofread, s.getReaderVersion()); err != nil {
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

func TestFileMenuRejectsAVersionTheCommandDoesNotList(t *testing.T) {
	s := newScene(t)
	proofread := getProofreadCommand()
	menu, recorder := newMenu(t, s, proofread)
	s.requestText(t)

	for name, version := range map[string]domain.TextVersionID{
		"another version": "org.numen.tools-proofreading/other",
		"no version":      "",
	} {
		if _, err := menu.RunVersionItem(t.Context(), s.vault, scanPath, proofread, version); !errors.Is(err, textversion.ErrInvalidVersion) {
			t.Errorf("%s: got %v, want ErrInvalidVersion", name, err)
		}
	}
	if _, err := menu.RunVersionItem(t.Context(), s.vault, scanPath, getRecogniseCommand(), s.getReaderVersion()); !errors.Is(err, textversion.ErrInvalidVersion) {
		t.Errorf("a command without items: got %v, want ErrInvalidVersion", err)
	}
	if len(recorder.ran) != 0 {
		t.Fatalf("ran %+v", recorder.ran)
	}
}

func TestNewFileMenuRequiresVersionsCommandsAndTypes(t *testing.T) {
	s := newScene(t)
	if _, err := textversion.NewFileMenu(nil, &commands{}, getPdfTypes()); err == nil {
		t.Error("a file menu was made without versions")
	}
	if _, err := textversion.NewFileMenu(s.versions, nil, getPdfTypes()); err == nil {
		t.Error("a file menu was made without commands")
	}
	if _, err := textversion.NewFileMenu(s.versions, &commands{}, nil); err == nil {
		t.Error("a file menu was made without a content type table")
	}
}

func TestAHandlerOfASharedCommandListsEveryVersionItsConditionMatches(t *testing.T) {
	s := newScene(t)
	deleteText := pluginport.PluginCommandEntry{PluginID: ocr, Command: pluginport.PluginCommand{
		ID: "app:version.delete", Title: "Delete recognition", When: "version.kind != ''",
	}}
	menu, recorder := newMenu(t, s, deleteText)
	s.requestText(t)
	s.runEntry(t, ocr)

	items, err := menu.ListVersionItems(t.Context(), s.vault, scanPath, deleteText)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("the handler offers %+v, want the versions of both plugins", items)
	}
	if _, err := menu.RunVersionItem(t.Context(), s.vault, scanPath, deleteText, s.getReaderVersion()); err != nil || len(recorder.ran) != 1 {
		t.Errorf("a version of another plugin: got %v, ran %+v", err, recorder.ran)
	}
}
