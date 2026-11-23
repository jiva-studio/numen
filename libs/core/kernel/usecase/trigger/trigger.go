// Package trigger runs the triggers the enabled plugins declare: the plugin's own handler of the
// trigger's command runs over each file where the trigger's condition and the handler's own condition
// hold. Whether the work is done is read from the data, as target.versions, and nothing is recorded.
package trigger

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/when"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// Plugins is the triggers, the handlers and the commands of the enabled plugins.
type Plugins interface {
	PluginTriggers
	ListHandlers() []plugin.CommandEntry
	ListCommands() []plugin.CommandEntry
}

// Versions gives the text versions of a file as target.versions holds them.
type Versions interface {
	// ListTargetVersions lists the versions of the file, each with its kind, its plugin and whether it
	// is live.
	ListTargetVersions(ctx context.Context, v domain.Vault, path string) ([]plugin.TargetVersion, error)
}

// Triggers runs the triggers over the files of a vault. One pass over a vault runs at a time.
type Triggers struct {
	deps TriggersDeps

	conditions *when.Conditions
	mu         sync.Mutex
}

// TriggersDeps is what Triggers is made of: Run runs the triggers of the Plugins over the files the
// Readers open, Types gives a file its type, Versions
// gives target.versions and Config gives the settings.
type TriggersDeps struct {
	Plugins  Plugins
	Run      pluginport.CommandRun
	Readers  vaultport.VaultReaders
	Types    contenttypeport.ContentTypeTable
	Versions Versions
	Config   pluginport.WhenConfig
}

// NewTriggers makes the use case.
func NewTriggers(deps TriggersDeps) *Triggers {
	return &Triggers{deps: deps, conditions: when.NewConditions()}
}

// ExecuteAll runs the triggers of the file events over each file of the vault where the condition of
// the trigger and the condition of its handler hold. It returns the failures of the runs, joined; a
// failed file is tried again in the next pass. A trigger on note.frontmatterChanged runs only in
// ExecuteFrontmatter.
func (u *Triggers) ExecuteAll(ctx context.Context, v domain.Vault) error {
	_, err := u.execute(ctx, v, isFileEvent, false, func(reader vaultport.VaultReader, each func(path string, isGone bool) error) error {
		return reader.WalkFiles(ctx, func(stat domain.FileStat) error {
			if stat.IsAbsent || stat.Kind != domain.KindFile && stat.Kind != domain.KindNote {
				return nil
			}
			return each(stat.Path, false)
		})
	})
	return err
}

// Execute runs the triggers of the file events over the files at the paths, as ExecuteAll does over
// the vault. A path that holds no file is passed over.
func (u *Triggers) Execute(ctx context.Context, v domain.Vault, paths []string) error {
	_, err := u.execute(ctx, v, isFileEvent, false, func(reader vaultport.VaultReader, each func(path string, isGone bool) error) error {
		_, err := walkPaths(ctx, reader, paths, false, each)
		return err
	})
	return err
}

// FrontmatterBatchSize is the most notes one run of a trigger on note.frontmatterChanged is given.
const FrontmatterBatchSize = 500

// ExecuteFrontmatter runs the triggers of note.frontmatterChanged over the notes at the paths: each
// trigger runs once for every FrontmatterBatchSize notes where its condition and the condition of its
// command hold, with those notes as the paths of the call. A path that holds no file is a removed
// note, with no note type and no properties. It returns the notes to run over in a later pass: those
// that could not be read, and those of the batches of a plugin that could not be reached.
func (u *Triggers) ExecuteFrontmatter(ctx context.Context, v domain.Vault, paths []string) ([]string, error) {
	var failed []string
	retry, err := u.execute(ctx, v, isFrontmatterEvent, true, func(reader vaultport.VaultReader, each func(path string, isGone bool) error) error {
		var err error
		failed, err = walkPaths(ctx, reader, paths, true, each)
		return err
	})
	return slices.Concat(failed, retry), err
}

func isFileEvent(event string) bool { return event != manifest.EventNoteFrontmatterChanged }

func isFrontmatterEvent(event string) bool { return event == manifest.EventNoteFrontmatterChanged }

func walkPaths(
	ctx context.Context, reader vaultport.VaultReader, paths []string, isGoneRun bool, each func(path string, isGone bool) error,
) (failed []string, err error) {
	var failures []error
	for _, path := range paths {
		stat, err := reader.StatFile(ctx, path)
		switch {
		case errors.Is(err, edit.ErrNotExist):
			if !isGoneRun {
				continue
			}
			if err := each(path, true); err != nil {
				return failed, errors.Join(append(failures, err)...)
			}
			continue
		case err != nil:
			failed = append(failed, path)
			failures = append(failures, fmt.Errorf("read %s: %w", path, err))
			continue
		case stat.Kind != domain.KindFile && stat.Kind != domain.KindNote:
			continue
		}
		if err := each(stat.Path, false); err != nil {
			return failed, errors.Join(append(failures, err)...)
		}
	}
	return failed, errors.Join(failures...)
}

func (u *Triggers) execute(
	ctx context.Context, v domain.Vault, isEvent func(string) bool, isBatched bool,
	walk func(vaultport.VaultReader, func(path string, isGone bool) error) error,
) (retry []string, err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	triggers := u.readTriggers(isEvent)
	if len(triggers) == 0 {
		return nil, nil
	}
	reader, err := u.deps.Readers.Open(v)
	if err != nil {
		return nil, err
	}
	p := &pass{
		types: u.deps.Types, run: u.deps.Run, reader: reader, versions: u.deps.Versions, vault: v, list: triggers,
		config: u.deps.Config.GetWhenConfig(), skipped: map[string]bool{}, retry: map[string]bool{},
	}
	if isBatched {
		p.batches = make([][]string, len(triggers))
	}
	err = walk(reader, func(path string, isGone bool) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		p.runFile(ctx, path, isGone)
		return nil
	})
	if ctx.Err() == nil {
		p.runBatches(ctx)
	}
	return slices.Sorted(maps.Keys(p.retry)), errors.Join(append(p.failures, err)...)
}

// trigger is one trigger with its condition and the condition of its handler read.
type trigger struct {
	entry     plugin.TriggerEntry
	condition when.Condition
	handler   when.Condition
}

func (u *Triggers) readTriggers(isEvent func(string) bool) []trigger {
	handlers := map[[2]string]string{}
	for _, entry := range slices.Concat(u.deps.Plugins.ListHandlers(), u.deps.Plugins.ListCommands()) {
		handlers[[2]string{entry.PluginID, entry.Command.ID}] = entry.Command.When
	}
	var out []trigger
	for _, entry := range u.deps.Plugins.ListTriggers() {
		if !isEvent(entry.Trigger.Event) {
			continue
		}
		handlerWhen, isHandled := handlers[[2]string{entry.PluginID, entry.Trigger.Command}]
		if !isHandled {
			continue
		}
		condition, err := u.conditions.Parse(entry.Trigger.When)
		if err != nil {
			continue
		}
		handler, err := u.conditions.Parse(handlerWhen)
		if err != nil {
			continue
		}
		out = append(out, trigger{entry: entry, condition: condition, handler: handler})
	}
	return out
}

// pass is one run of the triggers over files of a vault. skipped holds the plugins that could not be
// reached in this pass. batches, set in a pass over frontmatter, holds the paths each trigger of list
// runs over once every path is read, and retry the paths of the batches the pass left undone.
type pass struct {
	types    contenttypeport.ContentTypeTable
	run      pluginport.CommandRun
	reader   vaultport.VaultReader
	versions Versions
	vault    domain.Vault
	list     []trigger
	config   map[string]any
	skipped  map[string]bool
	batches  [][]string
	failures []error
	retry    map[string]bool
}

// file is the values of one file as a pass reads them: the note and the versions are read once, and
// only where a condition depends on them.
type file struct {
	path           string
	values         plugin.TargetValues
	isNoteRead     bool
	isVersionsRead bool
}

func (p *pass) runFile(ctx context.Context, path string, isGone bool) {
	current := &file{
		path:   path,
		values: plugin.NewTargetValues(path, false, p.types.GetContentType(path)).WithConfig(p.config),
	}
	if isGone {
		current.values = current.values.WithNote("", nil).WithVersions(nil)
		current.isNoteRead, current.isVersionsRead = true, true
	}
	for index, one := range p.list {
		if p.skipped[one.entry.PluginID] {
			continue
		}
		isTrue, err := p.isTrue(ctx, current, one)
		if err != nil {
			p.failures = append(p.failures, fmt.Errorf("read %s: %w", path, err))
			if p.batches != nil {
				p.retry[path] = true
			}
			return
		}
		switch {
		case !isTrue:
		case p.batches != nil:
			p.batches[index] = append(p.batches[index], path)
		default:
			p.runCall(ctx, one, plugin.CommandCall{PluginID: one.entry.PluginID, Command: one.entry.Trigger.Command, Path: path}, path)
		}
	}
}

func (p *pass) runBatches(ctx context.Context) {
	for index, paths := range p.batches {
		one := p.list[index]
		for batch := range slices.Chunk(paths, FrontmatterBatchSize) {
			if p.skipped[one.entry.PluginID] {
				p.keepRetry(batch)
				continue
			}
			call := plugin.CommandCall{PluginID: one.entry.PluginID, Command: one.entry.Trigger.Command, Paths: batch}
			p.runCall(ctx, one, call, fmt.Sprintf("%d notes from %s", len(batch), batch[0]))
		}
	}
}

func (p *pass) runCall(ctx context.Context, one trigger, call plugin.CommandCall, target string) {
	_, err := p.run.Execute(ctx, p.vault, call)
	if err == nil {
		return
	}
	if errors.As(err, new(*plugin.UnavailableError)) {
		p.skipped[one.entry.PluginID] = true
		p.keepRetry(call.Paths)
	}
	p.failures = append(p.failures, fmt.Errorf("%s of %s over %s: %w", one.entry.Trigger.Command, one.entry.PluginID, target, err))
}

func (p *pass) keepRetry(paths []string) {
	for _, path := range paths {
		p.retry[path] = true
	}
}

func (p *pass) isTrue(ctx context.Context, current *file, one trigger) (bool, error) {
	result := evaluate(current.values, one)
	if result == when.Unknown && current.values.IsNote() && !current.isNoteRead {
		noteType, properties, err := edit.ReadHead(ctx, p.reader, current.path)
		if err != nil {
			return false, err
		}
		current.values, current.isNoteRead = current.values.WithNote(noteType, properties), true
		result = evaluate(current.values, one)
	}
	if result == when.Unknown && !current.isVersionsRead {
		versions, err := p.versions.ListTargetVersions(ctx, p.vault, current.path)
		if err != nil {
			return false, err
		}
		current.values, current.isVersionsRead = current.values.WithVersions(versions), true
		result = evaluate(current.values, one)
	}
	return result == when.True, nil
}

func evaluate(values plugin.TargetValues, one trigger) when.Result {
	got := values.Get()
	results := []when.Result{one.condition.Evaluate(got), one.handler.Evaluate(got)}
	switch {
	case slices.Contains(results, when.False):
		return when.False
	case slices.Contains(results, when.Unknown):
		return when.Unknown
	}
	return when.True
}
