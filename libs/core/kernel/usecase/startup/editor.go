package startup

import (
	"context"
	"errors"
)

// Closer is an index the start opened and closes where a later step fails.
type Closer interface {
	Close() error
}

// OpenEditor opens the editor window over the registry and the index, and connects the plugins to the vault
// the window shows and to the window. OpenWindow closes the index where it fails; InstallVault lets the plugins
// work the vault the window shows, and ShareWindow gives them the vault list, the text versions and the window.
type OpenEditor[Registry any, Index Closer, Window any] struct {
	OpenRegistry func() (Registry, error)
	OpenIndex    func(ctx context.Context) (Index, error)
	OpenTexts    func(index Index) error
	OpenWindow   func(ctx context.Context, registry Registry, index Index) (Window, error)
	FollowVault  func(window Window)
	InstallVault func(registry Registry, index Index, window Window) error
	ShareWindow  func(index Index, window Window)
}

// Execute returns the window opened, with the plugins connected to it.
func (u OpenEditor[Registry, Index, Window]) Execute(ctx context.Context) (Window, error) {
	var none Window
	registry, err := u.OpenRegistry()
	if err != nil {
		return none, err
	}
	index, err := u.OpenIndex(ctx)
	if err != nil {
		return none, err
	}
	if err := u.OpenTexts(index); err != nil {
		return none, errors.Join(err, index.Close())
	}
	window, err := u.OpenWindow(ctx, registry, index)
	if err != nil {
		return none, err
	}
	u.FollowVault(window)
	if err := u.InstallVault(registry, index, window); err != nil {
		return none, err
	}
	u.ShareWindow(index, window)
	return window, nil
}
