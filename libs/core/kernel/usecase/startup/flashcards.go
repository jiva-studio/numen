package startup

import (
	"context"
)

// OpenFlashcards opens the registry and the index of the flashcards window, binds the notes, the vault
// list, the vault indexer, the plugin vault, the change feed and the themes to the window, and reports
// what it cannot follow. The editor window may write the same index
// at the same time.
type OpenFlashcards[Registry any, Index Closer, Vaults any] struct {
	Label         string
	OpenRegistry  func() (Registry, error)
	OpenIndex     func(ctx context.Context) (Index, error)
	OpenVaults    func(index Index) Vaults
	BindWindow    func(ctx context.Context, registry Registry, index Index, vaults Vaults)
	InstallVault  func(registry Registry, index Index) error
	FollowChanges func(ctx context.Context) (<-chan struct{}, error)
	ReportChanges func(ctx context.Context, changes <-chan struct{})
	BindThemes    func(say func(response string)) error
	Notify        func(message string)
}

// FlashcardsOpened is what the start opened and the application closes when it ends. Themes is why the
// themes could not be loaded, or nil.
type FlashcardsOpened[Index Closer, Vaults any] struct {
	Index  Index
	Vaults Vaults
	Themes error
}

// Execute returns what the start opened. Where the plugin vault cannot be installed it returns what was open
// up to then with the failure.
func (u OpenFlashcards[Registry, Index, Vaults]) Execute(ctx context.Context) (FlashcardsOpened[Index, Vaults], error) {
	var none FlashcardsOpened[Index, Vaults]
	registry, err := u.OpenRegistry()
	if err != nil {
		return none, err
	}
	index, err := u.OpenIndex(ctx)
	if err != nil {
		return none, err
	}
	vaults := u.OpenVaults(index)
	u.BindWindow(ctx, registry, index, vaults)
	result := FlashcardsOpened[Index, Vaults]{Index: index, Vaults: vaults}
	if err := u.InstallVault(registry, index); err != nil {
		return result, err
	}
	u.followChanges(ctx)
	result.Themes = u.BindThemes(u.sayTheme)
	return result, nil
}

func (u OpenFlashcards[Registry, Index, Vaults]) followChanges(ctx context.Context) {
	changes, err := u.FollowChanges(ctx)
	if err != nil {
		u.Notify(u.Label + ": the index is not being followed: " + err.Error())
		return
	}
	u.ReportChanges(ctx, changes)
}

func (u OpenFlashcards[Registry, Index, Vaults]) sayTheme(response string) {
	if response != "" {
		u.Notify("themes: " + response)
	}
}
