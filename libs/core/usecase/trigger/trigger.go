// Package trigger runs the triggers the enabled plugins declare: on file.added and file.changed, the
// plugin's own handler of the trigger's command runs over the file while the trigger's condition and
// the handler's own condition hold, once for each content of the file.
package trigger

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/notes/markdown"

	"github.com/jiva-studio/numen/libs/core/port/contenttypeport"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// Plugins is the triggers and the handlers of the enabled plugins and the backends that run the
// handlers.
type Plugins interface {
	pluginport.PluginTriggers
	ListHandlers() []pluginport.PluginCommandEntry
	RunCommand(ctx context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error)
}

// Triggers runs the triggers over the files of a vault. One run over a vault is in progress at a time.
type Triggers struct {
	plugins Plugins
	readers vaultport.VaultReaders
	types   contenttypeport.ContentTypeTable
	records pluginport.TriggerRecords
	config  func() (map[string]any, error)

	mu sync.Mutex
}

// NewTriggers runs the triggers of the plugins over the files the readers open, the table giving a
// file its type, recording what ran in records; config gives the settings as when conditions read
// them.
func NewTriggers(
	plugins Plugins, readers vaultport.VaultReaders, types contenttypeport.ContentTypeTable,
	records pluginport.TriggerRecords, config func() (map[string]any, error),
) *Triggers {
	return &Triggers{plugins: plugins, readers: readers, types: types, records: records, config: config}
}

// trigger is one trigger with its condition and the condition of its handler read.
type trigger struct {
	entry     pluginport.PluginTriggerEntry
	condition pluginport.PluginCondition
	handler   pluginport.PluginCondition
}

// Execute runs every trigger over each file of the vault where its condition and the condition of its
// handler hold and it has not run over the file's current content. A file the triggers have not seen
// before is added; a file whose content differs from what a trigger last ran over is changed. A
// trigger on file.added runs over a file it has never run over, and a trigger on file.changed over a
// file whose content changed since it last ran, or that it has never run over and that the vault held
// before this run. A run that fails is not recorded, and is tried again on the next run.
func (u *Triggers) Execute(ctx context.Context, v domain.Vault) error {
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
	record, err := u.records.Open(v)
	if err != nil {
		return err
	}
	state, err := record.Read(ctx)
	if err != nil {
		return fmt.Errorf("read the record of the triggers: %w", err)
	}
	config, err := u.config()
	if err != nil {
		return fmt.Errorf("read the settings: %w", err)
	}
	pass := &pass{triggers: u, reader: reader, list: triggers, config: config, state: newState(state), found: map[string]bool{}}
	var failures []error
	err = reader.WalkFiles(ctx, func(stat domain.FileStat) error {
		if stat.Kind != domain.KindFile && stat.Kind != domain.KindNote {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		failures = append(failures, pass.runFile(ctx, stat)...)
		return nil
	})
	if err != nil {
		// The runs made before the walk ended are kept, so the next pass does not run them again.
		return errors.Join(err, pass.writeRecord(context.WithoutCancel(ctx), record))
	}
	pass.forgetUnseen()
	if err := pass.writeRecord(ctx, record); err != nil {
		return err
	}
	return errors.Join(failures...)
}

func (p *pass) writeRecord(ctx context.Context, record pluginport.TriggerRecord) error {
	if !p.isChanged {
		return nil
	}
	if err := record.Write(ctx, p.state); err != nil {
		return fmt.Errorf("write the record of the triggers: %w", err)
	}
	return nil
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
		condition, err := pluginport.ParseCondition(entry.Trigger.When)
		if err != nil {
			continue
		}
		handler, err := pluginport.ParseCondition(handlerWhen)
		if err != nil {
			continue
		}
		out = append(out, trigger{entry: entry, condition: condition, handler: handler})
	}
	return out
}

func newState(state pluginport.TriggerState) pluginport.TriggerState {
	if state.Files == nil {
		state.Files = map[string]pluginport.TriggerFile{}
	}
	if state.Runs == nil {
		state.Runs = map[string]string{}
	}
	return state
}

// pass is one run of the triggers over the files of a vault.
type pass struct {
	triggers  *Triggers
	reader    vaultport.VaultReader
	list      []trigger
	config    map[string]any
	state     pluginport.TriggerState
	found     map[string]bool
	isChanged bool
}

// file is one file as a pass reads it: its content is read whole once, and only where a condition
// reads the note.
type file struct {
	stat    domain.FileStat
	hash    string
	isNew   bool
	raw     []byte
	values  map[string]any
	readErr error
}

func (p *pass) runFile(ctx context.Context, stat domain.FileStat) []error {
	p.found[stat.Path] = true
	current, err := p.readFile(ctx, stat)
	if err != nil {
		return []error{err}
	}
	var failures []error
	for _, one := range p.list {
		key := getRunKey(one.entry, stat.Path)
		last, hasRun := p.state.Runs[key]
		if hasRun && last == current.hash {
			continue
		}
		if !p.isTrue(ctx, current, one.condition) || !p.isTrue(ctx, current, one.handler) {
			continue
		}
		if isDue(one.entry.Trigger.On, hasRun, current.isNew) {
			_, err := p.triggers.plugins.RunCommand(ctx, pluginport.PluginCommandCall{
				PluginID: one.entry.PluginID, Command: one.entry.Trigger.Command, Path: stat.Path,
			})
			if err != nil {
				failures = append(failures, fmt.Errorf("%s on %s over %s: %w", one.entry.Trigger.Command, one.entry.Trigger.On, stat.Path, err))
				continue
			}
		}
		p.state.Runs[key] = current.hash
		p.isChanged = true
	}
	return failures
}

func isDue(event string, hasRun, isNew bool) bool {
	if event == pluginport.EventFileAdded {
		return !hasRun
	}
	return hasRun || !isNew
}

func (p *pass) readFile(ctx context.Context, stat domain.FileStat) (*file, error) {
	known, isKnown := p.state.Files[stat.Path]
	if isKnown && known.Size == stat.Size && known.ModTime.Equal(stat.ModTime) {
		return &file{stat: stat, hash: known.Hash}, nil
	}
	hash, err := p.hashFile(ctx, stat.Path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", stat.Path, err)
	}
	p.state.Files[stat.Path] = pluginport.TriggerFile{Size: stat.Size, ModTime: stat.ModTime, Hash: hash}
	p.isChanged = true
	return &file{stat: stat, hash: hash, isNew: !isKnown}, nil
}

func (p *pass) hashFile(ctx context.Context, path string) (hash string, err error) {
	content, err := p.reader.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() {
		err = errors.Join(err, content.Close())
	}()
	return domain.HashReader(content)
}

func (p *pass) isTrue(ctx context.Context, current *file, condition pluginport.PluginCondition) bool {
	return condition.Evaluate(p.getValues(ctx, current, condition)) == pluginport.ConditionTrue
}

func (p *pass) getValues(ctx context.Context, current *file, condition pluginport.PluginCondition) map[string]any {
	if current.values != nil {
		return current.values
	}
	names := condition.ListNames()
	isReadingNote := slices.ContainsFunc(names, func(name string) bool {
		return name == "target.noteType" || strings.HasPrefix(name, "target.properties")
	})
	var noteType domain.NoteType
	properties := map[string]any{}
	if isReadingNote && vaultport.IsNote(current.stat.Path) {
		if current.raw == nil && current.readErr == nil {
			current.raw, current.readErr = p.reader.Read(ctx, current.stat.Path)
		}
		if current.readErr == nil {
			note := markdown.Parse(current.stat, current.raw)
			noteType, properties = note.Type, maps.Clone(note.Frontmatter)
		}
	}
	values := pluginport.GetTargetValues(current.stat.Path, string(p.triggers.types.GetContentType(current.stat.Path)), string(noteType))
	values["target.properties"] = properties
	values["config"] = p.config
	if isReadingNote {
		current.values = values
	}
	return values
}

func (p *pass) forgetUnseen() {
	for path := range p.state.Files {
		if !p.found[path] {
			delete(p.state.Files, path)
			p.isChanged = true
		}
	}
	for key := range p.state.Runs {
		if !p.found[getRunPath(key)] {
			delete(p.state.Runs, key)
			p.isChanged = true
		}
	}
}

// runKeySeparator joins the parts of the key of a trigger's runs over a file; no part holds it.
const runKeySeparator = "\x00"

func getRunKey(entry pluginport.PluginTriggerEntry, path string) string {
	return strings.Join([]string{entry.PluginID, entry.Trigger.On, entry.Trigger.Command, path}, runKeySeparator)
}

func getRunPath(key string) string {
	return key[strings.LastIndex(key, runKeySeparator)+1:]
}
