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

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/trigger"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
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
		return nil, edit.ErrNotExist
	}
	s.reads++
	return []byte(content), nil
}

func (s *memoryVault) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	content, ok := s.files[path]
	if !ok {
		return nil, edit.ErrNotExist
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
		return domain.FileStat{}, edit.ErrNotExist
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

// plugins declares the triggers it is given and records each run. A path in failing fails, and a
// plugin in unavailable cannot be reached.
type plugins struct {
	mu          sync.Mutex
	triggers    []plugin.TriggerEntry
	ran         []string
	failing     map[string]bool
	unavailable map[string]bool
	handlerWhen map[string]string
	hold        chan struct{}
	started     chan struct{}
}

func (p *plugins) ListTriggers() []plugin.TriggerEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.triggers)
}

// ListHandlers lists a handler of each command a trigger runs, under the condition handlerWhen gives
// its command, or none.
func (p *plugins) ListHandlers() []plugin.CommandEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []plugin.CommandEntry
	for _, one := range p.triggers {
		out = append(out, plugin.CommandEntry{PluginID: one.PluginID, Command: manifest.Command{
			ID: one.Trigger.Command, When: p.handlerWhen[one.Trigger.Command],
		}})
	}
	return out
}

func (p *plugins) Execute(_ context.Context, _ domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	if p.hold != nil {
		p.started <- struct{}{}
		<-p.hold
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ran = append(p.ran, call.PluginID+" "+call.Command+" "+call.Path)
	if p.unavailable[call.PluginID] {
		return plugin.CommandResult{}, &plugin.UnavailableError{PluginID: call.PluginID, Reason: plugin.Failed}
	}
	if p.failing[call.Path] {
		return plugin.CommandResult{}, errors.New("the scanner is out of paper")
	}
	return plugin.CommandResult{Message: "done"}, nil
}

func (p *plugins) take() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := p.ran
	p.ran = nil
	return out
}

// versions is the text versions of each file by its path, counting the reads.
type versions struct {
	mu     sync.Mutex
	byPath map[string][]plugin.TargetVersion
	reads  int
}

func (v *versions) ListTargetVersions(_ context.Context, _ domain.Vault, path string) ([]plugin.TargetVersion, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.reads++
	return v.byPath[path], nil
}

// settingsOf is the config of when conditions it holds.
type settingsOf struct{ config *map[string]any }

func (s settingsOf) GetWhenConfig() map[string]any { return *s.config }

type scene struct {
	memoryVault *memoryVault
	plugins     *plugins
	versions    *versions
	config      map[string]any
	triggers    *trigger.Triggers
}

func newScene(t *testing.T, files map[string]string, entries ...plugin.TriggerEntry) *scene {
	t.Helper()
	s := &scene{
		memoryVault: newMemoryVault(files),
		plugins:     &plugins{triggers: entries, failing: map[string]bool{}, unavailable: map[string]bool{}},
		versions:    &versions{byPath: map[string][]plugin.TargetVersion{}},
		config:      map[string]any{},
	}
	vaultReaders := readers{s.memoryVault}
	s.triggers = trigger.NewTriggers(trigger.TriggersDeps{
		Plugins: s.plugins, Run: s.plugins, Readers: vaultReaders, Notes: note.NewReadFrontmatter(vaultReaders), Types: types{},
		Versions: s.versions, Config: settingsOf{config: &s.config},
	})
	return s
}

func (s *scene) runAll(t *testing.T) []string {
	t.Helper()
	if err := s.triggers.ExecuteAll(t.Context(), domain.Vault{ID: "v1"}); err != nil {
		t.Fatal(err)
	}
	return s.plugins.take()
}

func getExtractText() plugin.TriggerEntry {
	return plugin.TriggerEntry{PluginID: scanner, Trigger: manifest.Trigger{
		Event: manifest.EventFileAdded, Command: "app:file.createText",
		When: "target.contentType == 'application/pdf' && !target.versions.exists(v, v.kind == 'text-layer' && v.isValid)",
	}}
}

func getRecognizeWhenOn() plugin.TriggerEntry {
	return plugin.TriggerEntry{PluginID: scanner, Trigger: manifest.Trigger{
		Event: manifest.EventFileAdded, Command: "app:file.createText",
		When: "target.contentType == 'image/png' && config.org.example.scans.isAutomatic == true",
	}}
}

func getCaptionsOnLink() plugin.TriggerEntry {
	return plugin.TriggerEntry{PluginID: linker, Trigger: manifest.Trigger{
		Event: manifest.EventFileAdded, Command: "app:file.createText", When: "target.noteType == 'link' && has(target.properties.url)",
	}}
}

func TestTriggerRunsOverEveryFileWhereItsConditionIsTrue(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.png": "pixels", "notes/n.md": "# A note\n"}, getExtractText())

	if got, want := s.runAll(t), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestTriggerSkipsAFileWithAValidVersionOfItsKind(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.pdf": "%PDF b", "scans/c.pdf": "%PDF c", "scans/d.pdf": "%PDF d"}, getExtractText())
	s.versions.byPath["scans/a.pdf"] = []plugin.TargetVersion{{Kind: "text-layer", PluginID: scanner, IsLive: true, IsValid: true}}
	s.versions.byPath["scans/c.pdf"] = []plugin.TargetVersion{{Kind: "text-layer", PluginID: scanner, IsLive: false, IsValid: false}}
	s.versions.byPath["scans/d.pdf"] = []plugin.TargetVersion{
		{Kind: "text-layer", PluginID: scanner, IsLive: false, IsValid: true},
		{Kind: "proofreading", PluginID: "org.example.proofs", IsLive: true, IsValid: true},
	}

	if got, want := s.runAll(t), []string{scanner + " app:file.createText scans/b.pdf", scanner + " app:file.createText scans/c.pdf"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want the files with no valid text layer %v", got, want)
	}
}

func TestATriggerReadsNoFileAndNoVersionsWhereThePathAndTheSettingsDecide(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.png": "pixels", "notes/n.md": "# A note\n", "scans/b.pdf": "%PDF"}, getRecognizeWhenOn())
	s.config = map[string]any{"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": false}}}}

	if got := s.runAll(t); len(got) != 0 {
		t.Errorf("a trigger whose setting is off ran %v", got)
	}
	if s.memoryVault.reads != 0 || s.memoryVault.opens != 0 || s.versions.reads != 0 {
		t.Errorf("read %d files, opened %d and read the versions %d times, want none", s.memoryVault.reads, s.memoryVault.opens, s.versions.reads)
	}
}

func TestATriggerRunsOverTheFilesAlreadyThereOnceASettingMakesItsConditionTrue(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.png": "pixels"}, getRecognizeWhenOn())
	s.runAll(t)

	s.config = map[string]any{"org": map[string]any{"example": map[string]any{"scans": map[string]any{"isAutomatic": true}}}}

	if got, want := s.runAll(t), []string{scanner + " app:file.createText scans/a.png"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestATriggerReadsANoteOnlyWhereItsTypeOrPropertiesDecide(t *testing.T) {
	s := newScene(t, map[string]string{
		"links/tide.md": "---\ntype: link\nurl: https://tides.example/mill\n---\n",
		"links/bare.md": "---\ntype: link\n---\n",
		"scans/a.pdf":   "%PDF",
	}, getCaptionsOnLink())

	if got, want := s.runAll(t), []string{linker + " app:file.createText links/tide.md"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
	if s.memoryVault.reads != 2 || s.versions.reads != 0 {
		t.Errorf("read %d files and the versions %d times, want the two notes and no versions", s.memoryVault.reads, s.versions.reads)
	}
}

func TestTriggerRunsOnlyWhereTheConditionOfItsHandlerIsTrue(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.pdf": "%PDF b"}, getExtractText())
	s.plugins.handlerWhen = map[string]string{"app:file.createText": "!target.contentType.startsWith('application/') || target.isRoot"}

	if got := s.runAll(t); len(got) != 0 {
		t.Errorf("ran %v where the handler does not handle the files", got)
	}
}

func TestAPassOverPathsRunsOnlyOverThoseFiles(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.pdf": "%PDF b", "scans/c.pdf": "%PDF c"}, getExtractText())

	if err := s.triggers.Execute(t.Context(), domain.Vault{ID: "v1"}, []string{"scans/b.pdf", "scans/gone.pdf", "scans"}); err != nil {
		t.Fatal(err)
	}

	if got, want := s.plugins.take(), []string{scanner + " app:file.createText scans/b.pdf"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestAFileWhoseRunFailsIsTriedOncePerPassAndReported(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.pdf": "%PDF b"}, getExtractText())
	s.plugins.failing["scans/a.pdf"] = true

	for pass := range 2 {
		err := s.triggers.ExecuteAll(t.Context(), domain.Vault{ID: "v1"})
		if err == nil || !strings.Contains(err.Error(), "out of paper") {
			t.Errorf("pass %d answered %v, want the failure", pass, err)
		}
		want := []string{scanner + " app:file.createText scans/a.pdf", scanner + " app:file.createText scans/b.pdf"}
		if got := s.plugins.take(); !slices.Equal(got, want) {
			t.Errorf("pass %d ran %v, want %v", pass, got, want)
		}
	}
}

func TestAPluginThatCannotBeReachedIsSkippedForTheRestOfThePass(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a", "scans/b.pdf": "%PDF b", "links/tide.md": "---\ntype: link\nurl: https://tides.example\n---\n"},
		getExtractText(), getCaptionsOnLink())
	s.plugins.unavailable[scanner] = true

	err := s.triggers.ExecuteAll(t.Context(), domain.Vault{ID: "v1"})

	if !errors.As(err, new(*plugin.UnavailableError)) {
		t.Errorf("the pass answered %v, want the plugin unavailable", err)
	}
	want := []string{linker + " app:file.createText links/tide.md", scanner + " app:file.createText scans/a.pdf"}
	if got := s.plugins.take(); !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestOnePassRunsAtATime(t *testing.T) {
	s := newScene(t, map[string]string{"scans/a.pdf": "%PDF a"}, getExtractText())
	s.plugins.hold, s.plugins.started = make(chan struct{}), make(chan struct{}, 2)
	vault := domain.Vault{ID: "v1"}
	results := make(chan error, 2)
	go func() { results <- s.triggers.ExecuteAll(t.Context(), vault) }()
	<-s.plugins.started
	go func() { results <- s.triggers.Execute(t.Context(), vault, []string{"scans/a.pdf"}) }()

	select {
	case <-s.plugins.started:
		t.Fatal("a second pass ran while the first one was running")
	case <-time.After(50 * time.Millisecond):
	}
	close(s.plugins.hold)
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	if got := s.plugins.take(); len(got) != 2 {
		t.Errorf("ran %v, want the file once in each pass", got)
	}
}
