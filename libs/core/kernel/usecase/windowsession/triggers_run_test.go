package windowsession_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/windowsession"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	scanner = "org.example.scans"
	linker  = "org.example.links"
)

// memoryVault is a vault in memory: its files by path, each with a modification time.
type memoryVault struct {
	mu     sync.Mutex
	files  map[string]string
	times  map[string]time.Time
	reads  int
	opens  int
	broken map[string]bool
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
	if s.broken[path] {
		return domain.FileStat{}, errors.New("the disk cannot be read")
	}
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

// plugins declares the triggers it is given and records each run with its paths joined by commas. A
// path in failing fails, and a plugin in unavailable cannot be reached.
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

// ListHandlers lists a handler of each shared command a trigger runs, under the condition handlerWhen
// gives its command, or none.
func (p *plugins) ListHandlers() []plugin.CommandEntry {
	return p.listCommands(true)
}

// ListCommands lists each command of a plugin's own a trigger runs, as ListHandlers does.
func (p *plugins) ListCommands() []plugin.CommandEntry {
	return p.listCommands(false)
}

func (p *plugins) listCommands(isShared bool) []plugin.CommandEntry {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []plugin.CommandEntry
	for _, one := range p.triggers {
		if manifest.IsSharedCommand(one.Trigger.Command) != isShared {
			continue
		}
		out = append(out, plugin.CommandEntry{PluginID: one.PluginID, Command: manifest.Command{
			ID: one.Trigger.Command, When: p.handlerWhen[one.Trigger.Command],
		}})
	}
	return out
}

func (p *plugins) Run(_ context.Context, _ domain.Vault, call plugin.CommandCall) (plugin.CommandResult, error) {
	if p.hold != nil {
		p.started <- struct{}{}
		<-p.hold
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	target := call.Path
	if len(call.Paths) > 0 {
		target = strings.Join(call.Paths, ",")
	}
	p.ran = append(p.ran, call.PluginID+" "+call.Command+" "+target)
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
	triggers    *windowsession.Triggers
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
	s.triggers = windowsession.NewTriggers(windowsession.TriggersDeps{
		Plugins: s.plugins, Run: s.plugins, Readers: vaultReaders, Types: types{},
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

const tagger = "org.example.tags"

func getReadTagsOnFrontmatter() plugin.TriggerEntry {
	return plugin.TriggerEntry{PluginID: tagger, Trigger: manifest.Trigger{
		Event: manifest.EventNoteFrontmatterChanged, Command: "tags.read", When: "target.isNote",
	}}
}

func TestAFrontmatterPassRunsTheFrontmatterTriggersOverTheNotesNamedAndOverRemovedOnes(t *testing.T) {
	s := newScene(t, map[string]string{"notes/a.md": "---\ntags: [river]\n---\n", "notes/b.md": "# B\n", "scans/a.pdf": "%PDF a"},
		getExtractText(), getReadTagsOnFrontmatter())

	if _, err := s.triggers.ExecuteFrontmatter(t.Context(), domain.Vault{ID: "v1"}, []string{"notes/a.md", "notes/gone.md", "scans/a.pdf"}); err != nil {
		t.Fatal(err)
	}

	want := []string{tagger + " tags.read notes/a.md,notes/gone.md"}
	if got := s.plugins.take(); !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestAPassOverChangedFilesRunsNoFrontmatterTrigger(t *testing.T) {
	s := newScene(t, map[string]string{"notes/a.md": "# A\n", "scans/a.pdf": "%PDF a"}, getExtractText(), getReadTagsOnFrontmatter())

	if err := s.triggers.Execute(t.Context(), domain.Vault{ID: "v1"}, []string{"notes/a.md", "scans/a.pdf"}); err != nil {
		t.Fatal(err)
	}

	if got, want := s.plugins.take(), []string{scanner + " app:file.createText scans/a.pdf"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestAPassOverTheVaultRunsNoFrontmatterTrigger(t *testing.T) {
	s := newScene(t, map[string]string{"notes/a.md": "---\ntags: [river]\n---\n", "notes/b.md": "# B\n", "scans/a.pdf": "%PDF a"},
		getExtractText(), getReadTagsOnFrontmatter())

	want := []string{scanner + " app:file.createText scans/a.pdf"}
	if got := s.runAll(t); !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
}

func TestATriggerOfAPluginsOwnCommandRunsOnlyWhereTheConditionOfTheCommandIsTrue(t *testing.T) {
	s := newScene(t, map[string]string{"notes/a.md": "# A\n", "drafts/b.md": "# B\n"}, getReadTagsOnFrontmatter())
	s.plugins.handlerWhen = map[string]string{"tags.read": "!target.contentType.startsWith('text/') || target.isNote"}

	if _, err := s.triggers.ExecuteFrontmatter(t.Context(), domain.Vault{ID: "v1"}, []string{"drafts/b.md"}); err != nil {
		t.Fatal(err)
	}
	if got, want := s.plugins.take(), []string{tagger + " tags.read drafts/b.md"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}

	s.plugins.handlerWhen = map[string]string{"tags.read": "target.isFolder"}
	if _, err := s.triggers.ExecuteFrontmatter(t.Context(), domain.Vault{ID: "v1"}, []string{"drafts/b.md"}); err != nil {
		t.Fatal(err)
	}
	if got := s.plugins.take(); len(got) != 0 {
		t.Errorf("ran %v where the condition of the command does not hold", got)
	}
}

func TestAFrontmatterPassRunsATriggerOncePerBatchOverExactlyTheNotesNamed(t *testing.T) {
	files := map[string]string{"notes/untouched.md": "# U\n", "scans/a.pdf": "%PDF a"}
	var named []string
	for i := range 2*windowsession.FrontmatterBatchSize + 1 {
		path := fmt.Sprintf("notes/%04d.md", i)
		files[path] = "---\ntags: [river]\n---\n"
		named = append(named, path)
	}
	s := newScene(t, files, getReadTagsOnFrontmatter())

	if _, err := s.triggers.ExecuteFrontmatter(t.Context(), domain.Vault{ID: "v1"}, append(named, "scans/a.pdf")); err != nil {
		t.Fatal(err)
	}

	ran := s.plugins.take()
	if len(ran) != 3 {
		t.Fatalf("ran %d times, want 3", len(ran))
	}
	var covered []string
	for _, one := range ran {
		paths := strings.Split(strings.TrimPrefix(one, tagger+" tags.read "), ",")
		if len(paths) > windowsession.FrontmatterBatchSize {
			t.Errorf("one run was given %d notes, more than %d", len(paths), windowsession.FrontmatterBatchSize)
		}
		covered = append(covered, paths...)
	}
	if !slices.Equal(covered, named) {
		t.Errorf("the runs covered %d notes, want the %d named", len(covered), len(named))
	}
}

func TestAPluginThatCannotBeReachedGetsNoFurtherBatchesInTheFrontmatterPass(t *testing.T) {
	files := map[string]string{}
	var named []string
	for i := range windowsession.FrontmatterBatchSize + 1 {
		path := fmt.Sprintf("notes/%04d.md", i)
		files[path] = "# N\n"
		named = append(named, path)
	}
	s := newScene(t, files, getReadTagsOnFrontmatter())
	s.plugins.unavailable[tagger] = true

	retry, err := s.triggers.ExecuteFrontmatter(t.Context(), domain.Vault{ID: "v1"}, named)

	if err == nil {
		t.Error("a plugin that cannot be reached was not reported")
	}
	if got := len(s.plugins.take()); got != 1 {
		t.Errorf("ran %d times, want 1", got)
	}
	if !slices.Equal(retry, named) {
		t.Errorf("kept %d notes to run over again, want the %d of both batches", len(retry), len(named))
	}
}

func TestFrontmatterPassKeepsUnreadableNoteAndRunsTheRest(t *testing.T) {
	s := newScene(t, map[string]string{"notes/a.md": "# A\n", "notes/b.md": "# B\n", "notes/c.md": "# C\n"}, getReadTagsOnFrontmatter())
	s.memoryVault.broken = map[string]bool{"notes/a.md": true}

	retry, err := s.triggers.ExecuteFrontmatter(t.Context(), domain.Vault{ID: "v1"}, []string{"notes/a.md", "notes/b.md", "notes/c.md"})

	if err == nil {
		t.Error("a note that cannot be read was not reported")
	}
	if got, want := s.plugins.take(), []string{tagger + " tags.read notes/b.md,notes/c.md"}; !slices.Equal(got, want) {
		t.Errorf("ran %v, want %v", got, want)
	}
	if !slices.Equal(retry, []string{"notes/a.md"}) {
		t.Errorf("kept %v to run over again, want the note that cannot be read", retry)
	}
}
