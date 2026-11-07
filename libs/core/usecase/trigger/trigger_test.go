package trigger_test

import (
	"context"
	"errors"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const (
	scanner = "org.example.scans"
	linker  = "org.example.links"
)

// memoryVault is a vault in memory: its files by path, each with a modification time.
type memoryVault struct {
	mu    sync.Mutex
	files map[string]string
	times map[string]time.Time
	reads int
	opens int
}

func newMemoryVault(files map[string]string) *memoryVault {
	s := &memoryVault{files: map[string]string{}, times: map[string]time.Time{}}
	for _, path := range slices.Sorted(maps.Keys(files)) {
		s.put(path, files[path])
	}
	return s
}

func (s *memoryVault) put(path, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.files[path] = content
	s.times[path] = time.Unix(int64(1_700_000_000+len(s.times)*100+len(content)), 0)
}

func (s *memoryVault) remove(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.files, path)
	delete(s.times, path)
}

func (s *memoryVault) WalkFiles(ctx context.Context, fn func(domain.FileStat) error) error {
	s.mu.Lock()
	paths := slices.Sorted(maps.Keys(s.files))
	s.mu.Unlock()
	for _, path := range paths {
		stat, err := s.StatFile(ctx, path)
		if err != nil {
			return err
		}
		if err := fn(stat); err != nil {
			return err
		}
	}
	return nil
}

func (s *memoryVault) List(context.Context, string) ([]domain.Entry, error) { return nil, nil }

func (s *memoryVault) Read(_ context.Context, path string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.files[path]
	if !ok {
		return nil, vaultport.ErrNotExist
	}
	s.reads++
	return []byte(content), nil
}

func (s *memoryVault) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.files[path]
	if !ok {
		return nil, vaultport.ErrNotExist
	}
	s.opens++
	return nopCloser{strings.NewReader(content)}, nil
}

// nopCloser is a reader of a string that has nothing to close.
type nopCloser struct{ *strings.Reader }

func (nopCloser) Close() error { return nil }

func (s *memoryVault) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.files[path]
	if !ok {
		return domain.FileStat{}, vaultport.ErrNotExist
	}
	kind := domain.KindFile
	if strings.HasSuffix(path, ".md") {
		kind = domain.KindNote
	}
	return domain.FileStat{Path: path, Kind: kind, Size: int64(len(content)), ModTime: s.times[path]}, nil
}

// readers opens the one memoryVault for every vault.
type readers struct{ memoryVault *memoryVault }

func (r readers) Open(domain.Vault) (vaultport.VaultReader, error) { return r.memoryVault, nil }

// types gives a PDF and a PNG their type by extension.
type types struct{}

func (types) GetContentType(path string) domain.ContentType {
	switch {
	case strings.HasSuffix(path, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	case strings.HasSuffix(path, ".md"):
		return "text/markdown"
	}
	return domain.UnknownContentType
}

func (types) ListExtensions(domain.ContentType) []string { return nil }

// plugins declares the triggers it is given and records each run.
type plugins struct {
	mu       sync.Mutex
	triggers []pluginport.PluginTriggerEntry
	ran      []string
	failing  map[string]bool
	// handlerWhen is the condition of the handler of each command, by its full id.
	handlerWhen map[string]string
}

func (p *plugins) ListTriggers() []pluginport.PluginTriggerEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.triggers)
}

// ListHandlers lists a handler of each command a trigger runs, under the condition handlerWhen gives
// its command, or none.
func (p *plugins) ListHandlers() []pluginport.PluginCommandEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []pluginport.PluginCommandEntry
	for _, one := range p.triggers {
		out = append(out, pluginport.PluginCommandEntry{PluginID: one.PluginID, Command: pluginport.PluginCommand{
			ID: one.Trigger.Command, When: p.handlerWhen[one.Trigger.Command],
		}})
	}
	return out
}

func (p *plugins) RunCommand(_ context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failing[call.Path] {
		return pluginport.PluginCommandResult{}, errors.New("the scanner is out of paper")
	}
	p.ran = append(p.ran, call.PluginID+" "+call.Command+" "+call.Path)
	return pluginport.PluginCommandResult{Message: "done"}, nil
}

func (p *plugins) take() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.ran
	p.ran = nil
	return out
}

// memoryRecord keeps the state of the triggers in memory.
type memoryRecord struct {
	mu    sync.Mutex
	state pluginport.TriggerState
}

func (m *memoryRecord) Open(domain.Vault) (pluginport.TriggerRecord, error) { return m, nil }

func (m *memoryRecord) Read(context.Context) (pluginport.TriggerState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return pluginport.TriggerState{Files: maps.Clone(m.state.Files), Runs: maps.Clone(m.state.Runs)}, nil
}

func (m *memoryRecord) Write(_ context.Context, state pluginport.TriggerState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
	return nil
}

type scene struct {
	memoryVault *memoryVault
	plugins     *plugins
	config      map[string]any
	triggers    *trigger.Triggers
}

func newScene(t *testing.T, files map[string]string, entries ...pluginport.PluginTriggerEntry) *scene {
	t.Helper()
	s := &scene{memoryVault: newMemoryVault(files), plugins: &plugins{triggers: entries, failing: map[string]bool{}}, config: map[string]any{}}
	s.triggers = trigger.NewTriggers(s.plugins, readers{s.memoryVault}, types{}, &memoryRecord{},
		func() (map[string]any, error) { return s.config, nil })
	return s
}

func (s *scene) scan(t *testing.T) []string {
	t.Helper()
	if err := s.triggers.Execute(t.Context(), domain.Vault{ID: "v1"}); err != nil {
		t.Fatal(err)
	}
	return s.plugins.take()
}

// getExtractOnAdd extracts the text of a PDF added.
func getExtractOnAdd() pluginport.PluginTriggerEntry {
	return pluginport.PluginTriggerEntry{PluginID: scanner, Trigger: manifest.Trigger{
		On: manifest.EventFileAdded, Command: "app:file.createText", When: "target.contentType == 'application/pdf'",
	}}
}

// getRecognizeWhenOn runs over every file added while a setting is on.
func getRecognizeWhenOn() pluginport.PluginTriggerEntry {
	return pluginport.PluginTriggerEntry{PluginID: scanner, Trigger: manifest.Trigger{
		On: manifest.EventFileAdded, Command: "app:version.delete", When: "config.org.example.scans.isAutomatic == true",
	}}
}

// getRecheckOnChange reads a picture again when it changes.
func getRecheckOnChange() pluginport.PluginTriggerEntry {
	return pluginport.PluginTriggerEntry{PluginID: scanner, Trigger: manifest.Trigger{
		On: manifest.EventFileChanged, Command: "app:file.createText", When: "target.contentType == 'image/png'",
	}}
}

// getCaptionsOnLink downloads the captions of a link note that holds an address.
func getCaptionsOnLink() pluginport.PluginTriggerEntry {
	return pluginport.PluginTriggerEntry{PluginID: linker, Trigger: manifest.Trigger{
		On: manifest.EventFileAdded, Command: "app:file.createText", When: "target.noteType == 'link' && has(target.properties.url)",
	}}
}

func TestATriggerOnFileAddedRunsOnceOverEachMatchingFile(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "notes/n.md": "# A note\n"}, getExtractOnAdd())

	if got, want := s.scan(t), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Fatalf("the first scan ran %v, want %v", got, want)
	}
	if got := s.scan(t); len(got) != 0 {
		t.Errorf("a second scan over the same content ran %v", got)
	}
	s.memoryVault.put("scans/b.pdf", "%PDF b")
	if got, want := s.scan(t), []string{scanner + " app:file.createText scans/b.pdf"}; !slices.Equal(got, want) {
		t.Errorf("a file added ran %v, want %v", got, want)
	}
}

func TestATriggerOnFileAddedDoesNotRunWhenAFileChanges(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a"}, getExtractOnAdd())
	s.scan(t)

	s.memoryVault.put("scans/a.pdf", "%PDF a, second printing")
	if got := s.scan(t); len(got) != 0 {
		t.Errorf("a changed file ran %v", got)
	}
}

func TestATriggerOnFileChangedRunsWhenTheContentChanges(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.png": "pixels"}, getRecheckOnChange())
	if got := s.scan(t); len(got) != 0 {
		t.Fatalf("a file found for the first time ran %v on file.changed", got)
	}

	s.memoryVault.put("scans/a.png", "other pixels")
	if got, want := s.scan(t), []string{scanner + " app:file.createText scans/a.png"}; !slices.Equal(got, want) {
		t.Errorf("a changed file ran %v, want %v", got, want)
	}
	if got := s.scan(t); len(got) != 0 {
		t.Errorf("a second scan ran %v", got)
	}
}

func TestATriggerRunsOverTheFilesAlreadyThereOnceASettingMakesItsConditionTrue(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.png": "pixels", "scans/b.pdf": "%PDF b"}, getRecognizeWhenOn())
	if got := s.scan(t); len(got) != 0 {
		t.Fatalf("a trigger whose setting is off ran %v", got)
	}

	s.config = map[string]any{"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": true}}}}
	got := s.scan(t)
	want := []string{scanner + " app:version.delete scans/a.png", scanner + " app:version.delete scans/b.pdf"}
	if !slices.Equal(got, want) {
		t.Errorf("once the setting is on the scan ran %v, want %v", got, want)
	}
}

func TestATriggerOfAPluginEnabledLaterRunsOverTheFilesAlreadyThere(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a"})
	s.scan(t)

	s.plugins.triggers = []pluginport.PluginTriggerEntry{getExtractOnAdd()}
	if got, want := s.scan(t), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Errorf("the trigger of the plugin enabled ran %v, want %v", got, want)
	}
}

func TestATriggerReadsTheTypeAndPropertiesOfANote(t *testing.T) {
	s := newScene(t, map[string]string{
		"links/tide.md":  "---\ntype: link\nurl: https://tides.example/mill\n---\n",
		"links/plain.md": "---\ntype: link\n---\n",
		"notes/n.md":     "# A note\n",
	}, getCaptionsOnLink())

	if got, want := s.scan(t), []string{linker + " app:file.createText links/tide.md"}; !slices.Equal(got, want) {
		t.Errorf("the scan ran %v, want %v", got, want)
	}
}

func TestAFailedRunIsTriedAgainOnTheNextScan(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a"}, getExtractOnAdd())
	s.plugins.failing["scans/a.pdf"] = true
	if err := s.triggers.Execute(t.Context(), domain.Vault{ID: "v1"}); err == nil {
		t.Fatal("a failing run reported no error")
	}

	s.plugins.failing["scans/a.pdf"] = false
	if got, want := s.scan(t), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Errorf("the next scan ran %v, want %v", got, want)
	}
}

func TestAScanReadsNoFileWhenNothingChanged(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.png": "pixels"}, getExtractOnAdd())
	s.scan(t)
	before := s.memoryVault.reads + s.memoryVault.opens

	s.scan(t)
	if s.memoryVault.reads+s.memoryVault.opens != before {
		t.Errorf("a scan over unchanged files read %d files", s.memoryVault.reads+s.memoryVault.opens-before)
	}
}

func TestATriggerOnFileAddedRunsOverAFileDeletedAndAddedAgainWithTheSameContent(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a"}, getExtractOnAdd())
	s.scan(t)

	s.memoryVault.remove("scans/a.pdf")
	if got := s.scan(t); len(got) != 0 {
		t.Fatalf("a scan without the file ran %v", got)
	}
	s.memoryVault.put("scans/a.pdf", "%PDF a")
	if got, want := s.scan(t), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Errorf("the file added again ran %v, want %v", got, want)
	}
}

func TestAScanHashesAFileWithoutReadingItWhole(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a"}, getExtractOnAdd())
	s.scan(t)

	if s.memoryVault.reads != 0 {
		t.Errorf("the scan read %d files whole to hash them", s.memoryVault.reads)
	}
}

func TestATriggerRunsOnlyWhereTheConditionOfItsHandlerIsTrue(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.png": "pixels"}, getRecognizeWhenOn())
	s.plugins.handlerWhen = map[string]string{"app:version.delete": "target.contentType == 'image/png'"}
	s.config = map[string]any{"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": true}}}}

	if got, want := s.scan(t), []string{scanner + " app:version.delete scans/b.png"}; !slices.Equal(got, want) {
		t.Errorf("the scan ran %v, want %v", got, want)
	}
}
