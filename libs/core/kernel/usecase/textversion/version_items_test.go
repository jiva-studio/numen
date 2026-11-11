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

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
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
	all []plugin.CommandEntry
	ran []plugin.CommandCall
}

func (c *commands) ListKeybindings() []plugin.KeybindingEntry { return nil }

func (c *commands) ListSlots() []plugin.SlotEntry { return nil }

func (c *commands) ListHandlers() []plugin.CommandEntry { return nil }

func (c *commands) ListCommands() []plugin.CommandEntry { return c.all }

func (c *commands) GetParts(string) plugin.Parts {
	return plugin.Parts{HasBackend: true}
}

func (c *commands) Execute(_ context.Context, _ domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	c.ran = append(c.ran, call)
	return plugin.CommandResult{}, nil
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

func getRecognizeCommand() plugin.CommandEntry {
	return plugin.CommandEntry{PluginID: ocr, PluginName: "OCR", Command: manifest.Command{
		ID: "recognition.create", Title: "Recognize text", Icon: "scan-text", When: "target.contentType == 'application/pdf'",
	}}
}

func getProofreadCommand() plugin.CommandEntry {
	return plugin.CommandEntry{PluginID: proofreader, PluginName: "Proofreading", Command: manifest.Command{
		ID: "proofreading.create", Title: "Proofread", Icon: "spell-check",
		When: liveOfKinds(string(kindPage), string(kindRecognition)),
	}}
}

func liveOfKinds(kinds ...string) string {
	return "version.isLive && version.kind in ['" + strings.Join(kinds, "', '") + "']"
}

func newMenu(t *testing.T, s *scene, entries ...plugin.CommandEntry) (*textversion.VersionItems, *commands) {
	t.Helper()
	recorder := &commands{all: entries}
	menu := textversion.NewVersionItems(s.versions.Texts, s.versions.Versions, recorder, getPdfTypes())
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
	want := []plugin.CommandCall{{
		PluginID: proofreader, Command: "proofreading.create", Path: scanPath,
		Arguments: map[string]string{manifest.CommandArgumentVersion: string(s.getReaderVersion())},
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
		if _, err := menu.RunVersionItem(t.Context(), s.vault, scanPath, proofread, version); !errors.Is(err, text.ErrInvalidVersion) {
			t.Errorf("%s: got %v, want ErrInvalidVersion", name, err)
		}
	}
	if _, err := menu.RunVersionItem(t.Context(), s.vault, scanPath, getRecognizeCommand(), s.getReaderVersion()); !errors.Is(err, text.ErrInvalidVersion) {
		t.Errorf("a command without items: got %v, want ErrInvalidVersion", err)
	}
	if len(recorder.ran) != 0 {
		t.Fatalf("ran %+v", recorder.ran)
	}
}

func TestAHandlerOfASharedCommandListsEveryVersionItsConditionMatches(t *testing.T) {
	s := newScene(t)
	deleteText := plugin.CommandEntry{PluginID: ocr, Command: manifest.Command{
		ID: "app:file.createText", Title: "Proofread recognition", When: "version.kind != ''",
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

func TestACommandOverTheLiveVersionListsTheLiveVersionAlone(t *testing.T) {
	s := newScene(t)
	overLive := plugin.CommandEntry{PluginID: proofreader, Command: manifest.Command{
		ID: "proofreading.create", Title: "Proofread", When: "version.isLive",
	}}
	items, _ := newMenu(t, s, overLive)
	s.requestText(t)
	s.runEntry(t, ocr)
	live, _ := s.getLive(t)

	got, err := items.ListVersionItems(t.Context(), s.vault, scanPath, overLive)
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].Version != live || !got[0].IsLive {
		t.Errorf("items %+v, want the live version %s alone", got, live)
	}
}
