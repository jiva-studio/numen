package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/source"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// TextMenu is the menu the files of one vault offer about their text, the tasks the plugins report
// while they carry an entry out, and the cut that makes a file searchable from its live text.
type TextMenu struct {
	Menu  *textversion.FileMenu
	Tasks *run.Tasks
	Cut   func(ctx context.Context, v domain.Vault, path string) error
	Close func() error
}

// OpenTextMenu opens the index of the vault and lets the plugins that make text report their text
// versions to the host with no window.
func (c Config) OpenTextMenu(ctx context.Context, v domain.Vault) (TextMenu, error) {
	c.SetPluginVaultPath(v.Path)
	versions, err := c.NewTextVersions()
	if err != nil {
		return TextMenu{}, err
	}
	c.TextVersions = versions
	registry, err := c.Registry()
	if err != nil {
		return TextMenu{}, err
	}
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return TextMenu{}, err
	}
	extractor, err := c.Extract(db.Sources(), db.SourcesKnown(), v)
	if err != nil {
		_ = db.Close()
		return TextMenu{}, err
	}
	inFront := func() (domain.Vault, bool) { return v, true }
	vault, err := c.OpenPluginVault(registry, db, inFront, PluginVaultHooks{})
	if err != nil {
		_ = db.Close()
		return TextMenu{}, err
	}
	tasks := run.NewTasks()
	c.SetPluginWork(tasks, &run.Lock{})
	c.SetPluginVault(vault)
	c.SetPluginTexts(ctx, inFront)
	menu, err := textversion.NewFileMenu(versions, c.VaultReaders())
	if err != nil {
		_ = db.Close()
		return TextMenu{}, err
	}
	cutSource := source.NewCutSource(extractor)
	return TextMenu{Menu: menu, Tasks: tasks, Cut: cutSource.Execute, Close: db.Close}, nil
}
