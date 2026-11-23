package appstate

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// PluginCaches keeps what a plugin worked out for a vault, one file to a plugin, a name and a vault,
// in the platform's cache location.
type PluginCaches struct{ dir string }

// OpenPluginCaches uses the platform's cache location.
func OpenPluginCaches() (*PluginCaches, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return NewPluginCaches(filepath.Join(dir, "numen", "plugins")), nil
}

// OpenPluginCachesIn keeps the caches in the plugins folder of root, and in the platform's own location while
// root is empty.
func OpenPluginCachesIn(root string) (*PluginCaches, error) {
	if root == "" {
		return OpenPluginCaches()
	}
	return NewPluginCaches(filepath.Join(root, "plugins")), nil
}

// NewPluginCaches is OpenPluginCaches with an explicit folder, so tests do not
// touch the machine's own folder.
func NewPluginCaches(dir string) *PluginCaches { return &PluginCaches{dir: dir} }

// Read returns what is kept, and false where nothing is.
func (c *PluginCaches) Read(
	ctx context.Context, pluginID string, vaultID domain.VaultID, name string,
) ([]byte, bool, error) {
	store, err := c.getFiles(pluginID, name)
	if err != nil {
		return nil, false, err
	}
	raw, err := store.Read(ctx, vaultID)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

// Write replaces what is kept, whole.
func (c *PluginCaches) Write(
	ctx context.Context, pluginID string, vaultID domain.VaultID, name string, content []byte,
) error {
	store, err := c.getFiles(pluginID, name)
	if err != nil {
		return err
	}
	return store.Write(ctx, vaultID, content)
}

func (c *PluginCaches) getFiles(pluginID, name string) (*VaultFiles, error) {
	for _, part := range []string{pluginID, name} {
		if part == "" || strings.ContainsAny(part, `/\`) || strings.HasPrefix(part, ".") || strings.Contains(part, "..") {
			return nil, fmt.Errorf("%q is not a name a cache is kept under", part)
		}
	}
	return NewVaultFiles(filepath.Join(c.dir, pluginID, name)), nil
}
