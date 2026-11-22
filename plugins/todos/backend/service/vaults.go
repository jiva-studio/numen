package service

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/edit"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/queryfns"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/tasktools"
)

var errNoVault = errors.New("the call names no vault")

// Notes is the reach into the notes of one vault.
type Notes interface {
	ports.FileReader
	ports.FileWriter
	ports.NoteLister
	ports.Frontmatter
}

type scope struct {
	tasks *index.Index
	scope tasktools.Scope
}

// Vaults keeps one index of tasks for each vault a tool or a query function names, read the first
// time the vault is named and brought up to date at each call.
type Vaults struct {
	open func(id domain.VaultID) Notes

	mu      sync.Mutex
	scopes  map[domain.VaultID]*scope
	ignored string
}

// NewVaults returns the vaults reached through open.
func NewVaults(open func(id domain.VaultID) Notes) *Vaults {
	return &Vaults{open: open, mu: sync.Mutex{}, scopes: map[domain.VaultID]*scope{}, ignored: ""}
}

// SetIgnoredFolders sets the folders, separated by commas, whose notes the vaults do not search.
func (v *Vaults) SetIgnoredFolders(ctx context.Context, folders string) {
	v.mu.Lock()
	v.ignored = folders
	opened := make([]*scope, 0, len(v.scopes))
	for _, one := range v.scopes {
		opened = append(opened, one)
	}
	v.mu.Unlock()
	for _, one := range opened {
		ignore(one.tasks.SetIgnoredFolders(ctx, folders))
	}
}

// GetScope returns the tasks, the editor and the tick of the vault, with its tasks up to date.
func (v *Vaults) GetScope(ctx context.Context, vaultID string) (tasktools.Scope, error) {
	opened, err := v.get(ctx, vaultID)
	if err != nil {
		return tasktools.Scope{}, err
	}
	return opened.scope, nil
}

// GetTasks returns the tasks of the vault, up to date.
func (v *Vaults) GetTasks(ctx context.Context, vaultID string) (queryfns.Tasks, error) {
	opened, err := v.get(ctx, vaultID)
	if err != nil {
		return nil, err
	}
	return opened.tasks, nil
}

func (v *Vaults) get(ctx context.Context, vaultID string) (*scope, error) {
	if vaultID == "" {
		return nil, errNoVault
	}
	id := domain.VaultID(vaultID)
	v.mu.Lock()
	opened, isOpen := v.scopes[id]
	if !isOpen {
		opened = v.create(ctx, id)
		v.scopes[id] = opened
	}
	v.mu.Unlock()
	if err := opened.tasks.Refresh(ctx, nil, nil); err != nil {
		v.drop(id, opened)
		return nil, convertError("read the tasks", err)
	}
	return opened, nil
}

func (v *Vaults) create(ctx context.Context, id domain.VaultID) *scope {
	notes := v.open(id)
	tasks := index.NewForNotes(notes, notes, notes)
	ignore(tasks.SetIgnoredFolders(ctx, v.ignored))
	return &scope{tasks: tasks, scope: tasktools.Scope{
		Tasks: tasks, Editor: edit.New(notes, notes, tasks), Completer: complete.New(notes, notes, tasks),
	}}
}

func (v *Vaults) drop(id domain.VaultID, failed *scope) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if !failed.tasks.GetState().IsReady && v.scopes[id] == failed {
		delete(v.scopes, id)
	}
}
