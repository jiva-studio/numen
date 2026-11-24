// Package platform is what this machine supplies to the core that a settings
// file cannot name: the plugin host and the keychain the keys are kept in.
//
// Every entry point of this application begins from the configuration here, so
// a window and the command line reach the same machine.
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"

	"github.com/jiva-studio/numen/apps/desktop/internal/keychain"
	"github.com/jiva-studio/numen/apps/desktop/internal/plugindirs"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/identityfile"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/plugin/catalog/catalog"
	"github.com/jiva-studio/numen/libs/plugin/catalog/indexsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/localsource"
	"github.com/jiva-studio/numen/libs/plugin/catalog/packagecache"
)

// The windows a plugin names as the places it works in.
const (
	// TargetEditor is the editor window.
	TargetEditor = "desktop.editor"
	// TargetFlashcards is the flashcards window.
	TargetFlashcards = "desktop.flashcards"
)

// GetConfig returns the desktop launch paths.
func GetConfig() container.Config {
	var cfg container.Config
	if path, err := GetIndexPath(); err == nil {
		cfg.IndexPath = path
	}
	return cfg
}

// Installation holds the roles built when a desktop application starts.
type Installation struct {
	Execution   *container.PluginExecution
	Preferences *container.PluginPreferences
	Endpoints   *container.PluginEndpoints
	Resources   *container.Resources
	KeyFile     *keychain.KeyFile
}

// NewPluginServices builds the plugin host and shared resources of a desktop window or command.
func NewPluginServices(target string) Installation {
	dirs := plugindirs.GetDirs()
	res := container.NewResources()
	store := getKeyStore()
	passphrase := &keychain.Passphrase{}
	applyCredentialsFile(store, passphrase)
	res.KeyStore = store
	res.ChainStore = getChainStore()
	execution, preferences, endpoints, discoverErr := container.NewPluginHost(GetConfig().IndexPath, res, reportPluginHost, target, dirs...)
	reportPluginFailure("discover plugins", discoverErr)
	cacheRoot, dataRoot := getStorageRoots(runtime.GOOS, os.Getenv, os.UserCacheDir, os.UserHomeDir)
	preferences.SetStorage(cacheRoot, dataRoot)
	res.PluginCatalog = getCatalogConfig(target, dirs, execution.Runtime.Listing.IsPluginEnabled)
	return Installation{Execution: execution, Preferences: preferences, Endpoints: endpoints, Resources: res, KeyFile: keychain.NewKeyFile(store, passphrase)}
}

func reportPluginHost(err error) {
	reportPluginFailure("plugin host", err)
}

func reportPluginFailure(stage string, err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "numen: %s: %v\n", stage, err)
	}
}

func getChainStore() *identityfile.Chain {
	path, err := settings.GetPath()
	if err != nil {
		return identityfile.Open(os.TempDir())
	}
	return identityfile.Open(filepath.Dir(path))
}

// applyCredentialsFile gives the store the choice of keys.credentialsFile. A passphrase file waits for the
// window to give the passphrase.
func applyCredentialsFile(store *keychain.Store, passphrase *keychain.Passphrase) {
	path, err := settings.GetPath()
	if err != nil {
		return
	}
	switch settings.ReadCredentialsFile(path) {
	case settings.CredentialsFilePlaintext:
		store.UsePlaintextFile()
	case settings.CredentialsFilePassphrase:
		store.UsePassphraseFile(passphrase)
	}
}

func getKeyStore() *keychain.Store {
	path, err := settings.GetPath()
	if err != nil {
		return &keychain.Store{}
	}
	return keychain.NewSystemStore(filepath.Dir(path))
}

func getCatalogConfig(target string, pluginDirs []string, isEnabled localsource.SettingsFunc) catalog.Config {
	local := localsource.New(slices.Clone(pluginDirs), isEnabled)
	config := catalog.Config{Target: target, Sources: []catalog.Source{local, getIndexSource(pluginDirs)}, InstallDir: "", CacheDir: ""}
	if configDir, err := os.UserConfigDir(); err == nil {
		config.InstallDir = filepath.Join(configDir, "numen", "plugins")
	}
	if cacheDir, err := packagecache.DefaultDir(); err == nil {
		config.CacheDir = cacheDir
	}
	return config
}

func getIndexSource(pluginDirs []string) *indexsource.Source {
	// gomutants:disable-next-line reason="the capacity of the made value does not change what the function returns"
	indexPaths := make([]string, 0, len(pluginDirs))
	for _, dir := range pluginDirs {
		indexPaths = append(indexPaths, filepath.Join(dir, indexsource.FileName))
	}
	var manifestDirs []string
	if plugindirs.IsDev() {
		manifestDirs = plugindirs.GetDevDirs()
	}
	return indexsource.New(indexPaths, manifestDirs)
}
