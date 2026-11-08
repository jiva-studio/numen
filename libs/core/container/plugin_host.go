package container

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/commandbridge"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/whenconfig"
	"github.com/jiva-studio/numen/libs/core/usecase/trigger"
	"io/fs"
	"net/http"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/appstate"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/derivedfile"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/hostvault"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/manifestlist"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/servicebridge"
	"github.com/jiva-studio/numen/libs/core/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/core/usecase/plugincommand"
	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/secret"
	"github.com/jiva-studio/numen/libs/core/usecase/tabkind"
	"github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// pluginGracePeriod is how long a plugin has to exit after a stop request.
const pluginGracePeriod = 2 * time.Second

// errSafeMode is an installation started in safe mode, which starts no plugin.
var errSafeMode = errors.New("numen is running in safe mode")

const pluginFailureTaskPrefix = "plugin-failed:"

func (c Config) resolvePlugin(capability string) (pluginport.PluginManager, error) {
	if c.PluginHost == nil {
		return nil, &pluginport.UnavailableError{Capability: capability, Reason: pluginport.PluginNotInstalled}
	}
	if c.SafeMode {
		return nil, &pluginport.UnavailableError{Capability: capability, Reason: pluginport.PluginDisabled, Err: errSafeMode}
	}
	if _, err := c.PluginHost.ResolveCapability(capability); err != nil {
		return nil, err
	}
	return c.PluginHost, nil
}

func (c Config) hasPlugin(capability string) bool {
	_, err := c.resolvePlugin(capability)
	return err == nil
}

// SetSafeMode is the configuration started in safe mode or out of it, and its plugin host with it.
// In safe mode the host stops every plugin it runs and starts none.
func (c Config) SetSafeMode(enabled bool) Config {
	c.SafeMode = enabled
	if c.PluginHost != nil {
		c.PluginHost.SetSafeMode(enabled)
	}
	return c
}

// SetPluginVaultPath is the vault the plugins' paths are inside.
func (c Config) SetPluginVaultPath(path string) {
	if c.PluginHost != nil {
		c.PluginHost.SetVaultPath(path)
	}
}

// StopPlugins stops every plugin the host runs.
func (c Config) StopPlugins(ctx context.Context) error {
	if c.PluginHost == nil {
		return nil
	}
	return c.PluginHost.TeardownAll(ctx, pluginGracePeriod)
}

// GetPluginCommands is the use cases over the commands the plugins declare. Where there is no plugin
// host they list none.
func (c Config) GetPluginCommands() plugincommand.Commands {
	if c.PluginHost == nil {
		return plugincommand.NewCommands(nil, nil, nil, nil)
	}
	return plugincommand.NewCommands(c.openPluginCommands(), c.PluginHost, c.VaultReaders(), c.getWhenConfig())
}

// OpenChooseTabKind is the choice of the kind of tab a note opens in, over the tab kinds of the enabled
// plugins. Where there is no plugin host there is none.
func (c Config) OpenChooseTabKind() *tabkind.Choose {
	if c.PluginHost == nil {
		return nil
	}
	choose, err := tabkind.NewChoose(manifestlist.Open(c.PluginHost), c.PluginHost, c.getWhenConfig())
	if err != nil {
		c.handleError(err)
		return nil
	}
	return choose
}

// OpenPluginCommands opens the index of the vault and lets the plugins work it, its text versions
// and the queue through the host with no window, and answers how to run a command of theirs and
// how to close what it opened. The close waits until the plugins' work started meanwhile has
// finished, or ctx ends, and then closes the index.
func (c Config) OpenPluginCommands(ctx context.Context, v domain.Vault) (plugincommand.Run, func() error, error) {
	db, err := c.OpenIndex(ctx)
	if err != nil {
		return plugincommand.Run{}, nil, err
	}
	_, work, err := c.openPluginWork(db, v)
	if err != nil {
		return plugincommand.Run{}, nil, errors.Join(err, db.Close())
	}
	return c.GetPluginCommands().Run, closeAfterWork(ctx, work, db.Close), nil
}

func closeAfterWork(ctx context.Context, work *run.PluginWork, closeIndex func() error) func() error {
	return func() error {
		return errors.Join(work.WaitIdle(ctx), closeIndex())
	}
}

// OpenPluginWork lets the plugins work one vault, its text versions and the queue through the host
// with no window, over the index given. The configuration it returns reads the text of a file
// through the text versions, as a window does.
func (c Config) OpenPluginWork(ctx context.Context, db *Index, v domain.Vault) (Config, error) {
	opened, _, err := c.openPluginWork(db, v)
	return opened, err
}

func (c Config) openPluginWork(db *Index, v domain.Vault) (Config, *run.PluginWork, error) {
	c.SetPluginVaultPath(v.Path)
	versions, err := c.NewTextVersions(db)
	if err != nil {
		return c, nil, err
	}
	c.TextVersions = versions
	registry, err := c.Registry()
	if err != nil {
		return c, nil, err
	}
	current := func() (domain.Vault, bool) { return v, true }
	vault, err := c.OpenPluginVault(registry, db, current, PluginVaultHooks{})
	if err != nil {
		return c, nil, err
	}
	work := run.NewPluginWork(&run.Lock{}, run.NewTasks(), c.getPluginName)
	if c.PluginHost != nil {
		c.PluginHost.SetPluginWork(work)
	}
	c.SetPluginVault(vault)
	c.SetPluginTexts()
	return c, work, nil
}

// PluginServices is where a window forwards the calls it receives for the services plugins serve.
type PluginServices = interface {
	http.Handler
	// Serves reports whether an enabled plugin serves the service the path names.
	Serves(urlPath string) bool
	// ServeRoute forwards a request on the plugin route, /plugin/<id>/..., to that plugin.
	ServeRoute(w http.ResponseWriter, r *http.Request)
	// ListFrameOrigins lists the origins the pages of every enabled plugin frame.
	ListFrameOrigins() []string
	// GetFrameOrigins lists the origins the pages of one enabled plugin frame.
	GetFrameOrigins(pluginID string) []string
}

// GetPluginServices is the forwarding of the services plugins serve to the windows. Where there is
// no plugin host it serves none.
func (c Config) GetPluginServices() PluginServices {
	if c.PluginHost == nil {
		return servicebridge.NoServices{}
	}
	return servicebridge.Open(c.PluginHost)
}

// GetPluginRetrier starts again, on request from the window, a plugin whose last start failed.
func (c Config) GetPluginRetrier() func(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
	return func(ctx context.Context, pluginID string) (*pluginport.PluginFailure, error) {
		if c.PluginHost == nil {
			return nil, &pluginport.UnavailableError{PluginID: pluginID, Reason: pluginport.PluginNotInstalled}
		}
		return c.PluginHost.RetryPlugin(ctx, pluginID)
	}
}

// ReportPluginFailures keeps one task in the list for each plugin whose last start failed, with the reason as its
// error, and takes it out when the host forgets the failure.
func (c Config) ReportPluginFailures(tasks *run.Tasks) {
	if c.PluginHost == nil {
		return
	}
	c.PluginHost.SetOnPluginFailed(func(failure pluginport.PluginFailure) {
		tasks.Set(run.Task{
			ID:         pluginFailureTaskPrefix + failure.PluginID,
			Title:      "Plugin stopped",
			About:      failure.Name,
			Error:      failure.Reason,
			PluginID:   failure.PluginID,
			PluginName: failure.Name,
		})
	})
	c.PluginHost.SetOnFailureCleared(func(pluginID string) {
		tasks.Remove(pluginFailureTaskPrefix + pluginID)
		c.getResources().pluginChanges.Signal()
	})
}

// SetPluginWork opens the lock the models are held under and the list of tasks to the plugins.
func (c Config) SetPluginWork(tasks *run.Tasks, models *run.Lock) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginWork(run.NewPluginWork(models, tasks, c.getPluginName))
	}
}

func (c Config) getPluginName(pluginID string) string {
	if found, isKnown := c.PluginHost.GetPlugin(pluginID); isKnown && found.Manifest.Name != "" {
		return found.Manifest.Name
	}
	return pluginID
}

// SetPluginVault is the vault the plugins use through the host. Without one, a plugin that requires
// the vault receives an error stating that the host provides none.
func (c Config) SetPluginVault(vault pluginport.PluginVault) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginVault(vault)
	}
}

// SetPluginVaultList sets the list of vaults exposed to plugins. Without one, calls that change
// the list fail with ErrUnavailable.
func (c Config) SetPluginVaultList(list pluginport.PluginVaultList) {
	if c.PluginHost != nil {
		c.PluginHost.SetPluginVaultList(list)
	}
}

// OpenPluginCache is where a plugin keeps what it worked out for a vault: the folder the
// configuration names for schedules, else the platform's cache location.
func (c Config) OpenPluginCache() (pluginport.PluginCache, error) {
	if c.PluginCachePath != "" {
		return appstate.PluginCachesAt(filepath.Join(c.PluginCachePath, "plugins")), nil
	}
	retained, err := appstate.OpenPluginCaches()
	if err != nil {
		return nil, err
	}
	return retained, nil
}

// PluginVaultHooks is what a window adds to the vault it lets the plugins work.
type PluginVaultHooks struct {
	Written    func()
	IndexState func(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error)
	Moved      func(ctx context.Context) (<-chan struct{}, error)
	Notes      note.Scenarios
}

func hasReaders(notes note.Scenarios) bool {
	return notes.Move.Readers() != nil
}

func getNoteChanges(notes note.Scenarios) hostvault.NoteChanges {
	return hostvault.NoteChanges{
		Rename: notes.Rename, Move: notes.Move, Remove: notes.RemoveNote, Links: notes.Linking, AddLinks: notes.AddLinks,
	}
}

// OpenPluginVault is the vault the plugins work through the host over the vaults of a list and the
// index that holds them: the one current names, else the one a call names by identity.
func (c Config) OpenPluginVault(
	registry vaultport.VaultRegistry, db *Index, current func() (domain.Vault, bool), hooks PluginVaultHooks,
) (pluginport.PluginVault, error) {
	cache, err := c.OpenPluginCache()
	if err != nil {
		return nil, err
	}
	refresh := makeRefresh(c, db, db.Notes())
	queries := db.Queries()
	index := func(ctx context.Context, v domain.Vault, paths []string) error {
		_, err := refresh.Execute(ctx, v, paths)
		return err
	}
	scenarios := c.Notes(queries, db.Links(), db.Sources(), db.SourcesKnown(), index)
	arranger := hostvault.Arranger{
		Notes:  getNoteChanges(scenarios),
		Checks: vault.NewStandardGetProblems(db.Problems()),
	}
	if hasReaders(hooks.Notes) {
		arranger.Window, arranger.WindowVault = getNoteChanges(hooks.Notes), current
	}
	vaults := hostvault.Vault{
		Current:     current,
		Readers:     c.VaultReaders(),
		Writers:     c.VaultWriters(),
		Notes:       queries,
		Links:       db.Links(),
		Connections: note.NewConnections(scenarios),
		Problems:    db.Problems(),
		Arranger:    arranger,
		Stores:      derivedfile.PluginStores{Options: c.vaultOptions()},
		Cache:       cache,
		Watcher:     c.VaultWatcher(),
		Written:     hooks.Written,
		Vaults:      registry.List,
		Find: func(id domain.VaultID) (domain.Vault, error) {
			found, isFound, err := registry.FindVault(string(id))
			if err != nil {
				return domain.Vault{}, err
			}
			if !isFound {
				return domain.Vault{}, fmt.Errorf("vault %q: %w", id, fs.ErrNotExist)
			}
			return found, nil
		},
		IndexState: hooks.IndexState,
		Moved:      hooks.Moved,
		Index:      index,
	}
	if vaults.IndexState == nil {
		vaults.IndexState = func(ctx context.Context, v domain.Vault) (pluginport.PluginIndexState, error) {
			has, err := queries.Has(ctx, v.ID)
			if err != nil {
				return pluginport.PluginIndexState{}, err
			}
			if has {
				return pluginport.PluginIndexState{Status: pluginport.PluginIndexed}, nil
			}
			return pluginport.PluginIndexState{Status: pluginport.PluginIndexing}, nil
		}
	}
	if vaults.Moved == nil {
		vaults.Moved = c.Moves
	}
	return vaults, nil
}

// GetPluginDir is the folder of an installed plugin, and false for one not installed.
func (c Config) GetPluginDir(id string) (string, bool) {
	if c.PluginHost == nil {
		return "", false
	}
	p, ok := c.PluginHost.GetPlugin(id)
	if !ok {
		return "", false
	}
	return p.Dir, true
}

// GetFrontendPluginIDs are the plugins of the host's target that declare a frontend, in the order of
// their ids.
func (c Config) GetFrontendPluginIDs() []string {
	if c.PluginHost == nil {
		return nil
	}
	infos := c.PluginHost.GetPluginInfos()
	ids := make([]string, 0, len(infos))
	for _, info := range infos {
		if !info.HasFrontend {
			continue
		}
		ids = append(ids, info.ID)
	}
	return ids
}

// SetPluginWindow sets the window exposed to plugins. A host with no window sets none, and calls
// fail with ErrNoWindow.
func (c Config) SetPluginWindow(window pluginport.PluginWindow) {
	if c.PluginHost != nil {
		var host pluginport.PluginWindowHost = c.PluginHost
		host.SetPluginWindow(window)
	}
}

// credentialsLocator is a key store that keeps its values in a file when the machine has no keychain.
type credentialsLocator interface {
	// GetCredentialsPath returns the file the values are kept in, or the empty text while the
	// system keychain is in use.
	GetCredentialsPath(ctx context.Context) string
}

func (c Config) openSecretLedger() *settings.SecretLedger {
	return settings.NewSecretLedger(c.getResources().settingsFiles, c.settingsFile, c.readCheckedSettings, c.refreshSettingEvents)
}

// OpenSecrets returns the use cases on the secrets shared between plugins.
func (c Config) OpenSecrets() *secret.Secrets {
	var plugins pluginport.SecretPlugins = secret.NoPlugins{}
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	return secret.NewSecrets(c.openSecretLedger(), c.KeyStore, plugins, newSecretID, c.getCredentialsPath, c.getResources().secretGate)
}

func (c Config) getCredentialsPath() string {
	locator, isLocator := c.KeyStore.(credentialsLocator)
	if !isLocator {
		return ""
	}
	return locator.GetCredentialsPath(context.Background())
}

func newSecretID() (string, error) {
	var random [4]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", fmt.Errorf("read random bytes for the id of a secret: %w", err)
	}
	return "s" + hex.EncodeToString(random[:]), nil
}

func (c Config) setSecretSource() {
	ledger := c.openSecretLedger()
	c.PluginHost.SetSecretSource(pluginport.SecretSource{
		List:  ledger.GetSecrets,
		Prune: c.OpenSecrets().Prune.Execute,
	})
}

// WatchPluginChanges reports each time the plugin host applied a batch of settings, which enables,
// disables and configures plugins, and each time it forgot the failure of a plugin, as after a retry
// that succeeded. The returned function stops the reports.
func (c Config) WatchPluginChanges() (<-chan struct{}, func()) {
	return c.getResources().pluginChanges.Subscribe()
}

// commandManifests lists what the manifests of the enabled plugins declare.
type commandManifests interface {
	ListCommands() []pluginport.PluginCommandEntry
	ListSlots() []pluginport.PluginSlotEntry
	ListHandlers() []pluginport.PluginCommandEntry
	GetParts(pluginID string) pluginport.PluginParts
}

// commandRuns runs a command in the backend of its plugin.
type commandRuns interface {
	RunCommand(ctx context.Context, call pluginport.PluginCommandCall) (pluginport.PluginCommandResult, error)
}

// pluginCommands is the commands the manifests declare, listed from the manifests, and run in the
// backends of their plugins.
type pluginCommands struct {
	commandManifests
	commandRuns
}

func (c Config) openPluginCommands() pluginCommands {
	if c.PluginHost == nil {
		return pluginCommands{manifestlist.NoManifests{}, commandbridge.NoCommands{}}
	}
	return pluginCommands{manifestlist.Open(c.PluginHost), commandbridge.Open(c.PluginHost)}
}

func (c Config) openTriggers() *trigger.Triggers {
	if c.PluginHost == nil || c.SafeMode || c.TextVersions == nil {
		return nil
	}
	return trigger.NewTriggers(
		manifestlist.Open(c.PluginHost), c.GetPluginCommands().Run, c.VaultReaders(), c.PluginHost,
		c.TextVersions, c.getWhenConfig(),
	)
}

func (c Config) getWhenConfig() pluginport.WhenConfig {
	settings := whenconfig.Settings{App: c.listAppSettings}
	if c.PluginHost != nil {
		settings.Plugins = c.PluginHost.GetPluginInfos
	}
	return settings
}
