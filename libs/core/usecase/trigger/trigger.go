// Package trigger runs the triggers the enabled plugins declare: the plugin's own handler of the
// trigger's command runs over each file where the trigger's condition and the handler's own condition
// hold. Whether the work is done is read from the data, as target.versions, and nothing is recorded.
package trigger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Plugins is the triggers and the handlers of the enabled plugins.
type Plugins interface {
	pluginport.PluginTriggers
	ListHandlers() []pluginport.PluginCommandEntry
}

// CommandRun runs a handler checked against its declaration, as a run a person starts is: its
// backend, its condition and its arguments.
type CommandRun interface {
	Execute(ctx context.Context, v domain.Vault, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error)
}

// Versions gives the text versions of a file as target.versions holds them.
type Versions interface {
	// ListTargetVersions lists the versions of the file, each with its kind, its plugin and whether it
	// is live.
	ListTargetVersions(ctx context.Context, v domain.Vault, path string) ([]pluginport.TargetVersion, error)
}

// Triggers runs the triggers over the files of a vault. One pass over a vault runs at a time.
type Triggers struct {
	plugins  Plugins
	run      CommandRun
	readers  vaultport.VaultReaders
	types    contenttypeport.ContentTypeTable
	versions Versions
	config   pluginport.WhenConfig

	conditions *pluginport.PluginConditions
	mu         sync.Mutex
}

// NewTriggers runs the triggers of the plugins with run over the files the readers open, the table
// giving a file its type, the versions giving target.versions and config giving the settings.
func NewTriggers(
	plugins Plugins, run CommandRun, readers vaultport.VaultReaders, types contenttypeport.ContentTypeTable,
	versions Versions, config pluginport.WhenConfig,
) *Triggers {
	return &Triggers{
		plugins: plugins, run: run, readers: readers, types: types, versions: versions, config: config,
		conditions: pluginport.NewConditions(),
	}
}

// ExecuteAll runs every trigger over each file of the vault where its condition and the condition of
// its handler hold. It returns the failures of the runs, joined; a failed file is tried again in the
// next pass.
func (u *Triggers) ExecuteAll(ctx context.Context, v domain.Vault) error {
	return u.execute(ctx, v, func(reader vaultport.VaultReader, each func(domain.FileStat) error) error {
		return reader.WalkFiles(ctx, each)
	})
}

// Execute runs every trigger over the files at the paths, as ExecuteAll does over the vault. A path
// that holds no file is passed over.
func (u *Triggers) Execute(ctx context.Context, v domain.Vault, paths []string) error {
	return u.execute(ctx, v, func(reader vaultport.VaultReader, each func(domain.FileStat) error) error {
		for _, path := range paths {
			stat, err := reader.StatFile(ctx, path)
			if errors.Is(err, vaultport.ErrNotExist) {
				continue
			}
			if err != nil {
				return fmt.Errorf("read %s: %w", path, err)
			}
			if err := each(stat); err != nil {
				return err
			}
		}
		return nil
	})
}

func (u *Triggers) execute(
	ctx context.Context, v domain.Vault, walk func(vaultport.VaultReader, func(domain.FileStat) error) error,
) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	triggers := u.readTriggers()
	if len(triggers) == 0 {
		return nil
	}
	reader, err := u.readers.Open(v)
	if err != nil {
		return err
	}
	p := &pass{triggers: u, vault: v, list: triggers, config: u.config.GetWhenConfig(), skipped: map[string]bool{}}
	err = walk(reader, func(stat domain.FileStat) error {
		if stat.Kind != domain.KindFile && stat.Kind != domain.KindNote {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		p.runFile(ctx, stat.Path)
		return nil
	})
	return errors.Join(append(p.failures, err)...)
}

// trigger is one trigger with its condition and the condition of its handler read.
type trigger struct {
	entry     pluginport.PluginTriggerEntry
	condition pluginport.PluginCondition
	handler   pluginport.PluginCondition
}

func (u *Triggers) readTriggers() []trigger {
	handlers := map[[2]string]string{}
	for _, entry := range u.plugins.ListHandlers() {
		handlers[[2]string{entry.PluginID, entry.Command.ID}] = entry.Command.When
	}
	var out []trigger
	for _, entry := range u.plugins.ListTriggers() {
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
// reached in this pass.
type pass struct {
	triggers *Triggers
	vault    domain.Vault
	list     []trigger
	config   map[string]any
	skipped  map[string]bool
	failures []error
}

// file is the values of one file as a pass reads them: the note and the versions are read once, and
// only where a condition depends on them.
type file struct {
	path           string
	values         pluginport.TargetValues
	isNoteRead     bool
	isVersionsRead bool
}

func (p *pass) runFile(ctx context.Context, path string) {
	current := &file{
		path:   path,
		values: pluginport.NewTargetValues(path, false, p.triggers.types.GetContentType(path)).WithConfig(p.config),
	}
	for _, one := range p.list {
		if p.skipped[one.entry.PluginID] {
			continue
		}
		isTrue, err := p.isTrue(ctx, current, one)
		if err != nil {
			p.failures = append(p.failures, fmt.Errorf("read %s: %w", path, err))
			return
		}
		if !isTrue {
			continue
		}
		_, err = p.triggers.run.Execute(ctx, p.vault, pluginport.PluginCommandCall{
			PluginID: one.entry.PluginID, Command: one.entry.Trigger.Command, Path: path,
		})
		if err == nil {
			continue
		}
		if errors.As(err, new(*pluginport.UnavailableError)) {
			p.skipped[one.entry.PluginID] = true
		}
		p.failures = append(p.failures, fmt.Errorf("%s of %s over %s: %w", one.entry.Trigger.Command, one.entry.PluginID, path, err))
	}
}

func (p *pass) isTrue(ctx context.Context, current *file, one trigger) (bool, error) {
	result := evaluate(current.values, one)
	if result == pluginport.ConditionUnknown && current.values.IsNote() && !current.isNoteRead {
		noteType, properties, err := vaultport.ReadNote(ctx, p.triggers.readers, p.vault, current.path)
		if err != nil {
			return false, err
		}
		current.values, current.isNoteRead = current.values.WithNote(noteType, properties), true
		result = evaluate(current.values, one)
	}
	if result == pluginport.ConditionUnknown && !current.isVersionsRead {
		versions, err := p.triggers.versions.ListTargetVersions(ctx, p.vault, current.path)
		if err != nil {
			return false, err
		}
		current.values, current.isVersionsRead = current.values.WithVersions(versions), true
		result = evaluate(current.values, one)
	}
	return result == pluginport.ConditionTrue, nil
}

func evaluate(values pluginport.TargetValues, one trigger) pluginport.PluginConditionResult {
	got := values.Get()
	results := []pluginport.PluginConditionResult{one.condition.Evaluate(got), one.handler.Evaluate(got)}
	switch {
	case slices.Contains(results, pluginport.ConditionFalse):
		return pluginport.ConditionFalse
	case slices.Contains(results, pluginport.ConditionUnknown):
		return pluginport.ConditionUnknown
	}
	return pluginport.ConditionTrue
}
