package container

import (
	"context"
	"fmt"
	"path/filepath"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/shell"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/pluginchanges"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/port/agentport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (c Config) readSettingsFile() (settingsadapter.File, error) {
	return settingsSettings(c.settings())
}

func (c Config) getSettingsAt(path string) (settingsadapter.File, error) {
	return settingsGetSettingsAt(c.settings(), path)
}

func (c Config) getSettingsOrDefaults() settingsadapter.File {
	return settingsGetSettingsOrDefaults(c.settings())
}

// ApplySettings reads the settings file and returns the configuration with its values applied, and
// the notices that reading it raised. An unconfigured installation applies the default settings.
func (c Config) ApplySettings() (Config, []string, error) {
	selected, err := c.readSettingsFile()
	if err != nil {
		return c, nil, err
	}
	return c.applySettingsFile(selected), selected.Notices, nil
}

func (c Config) applySettingsFile(s settingsadapter.File) Config {
	c.Hardware, c.ResolvedCompute = resolveCompute(c.settings(), s)
	configurePluginHost(c.settings(), s)
	return c
}

// Plugins returns a function that lists the discovered plugins and their
// settings schemas.
func (c Config) Plugins() func() []plugindomain.Info {
	return settingsPlugins(c.settings())
}

// Models returns a function that lists the models each model setting accepts,
// and the programs the agent setting accepts.
func (c Config) Models() func() []settingsdomain.Model {
	return c.modelsForProvider(c.AgentProvider())
}

func (c Config) modelsForProvider(provider agentport.AgentProvider) func() []settingsdomain.Model {
	c.agentProvider = provider
	if c.getPrograms() == nil {
		c.programs = &settingsadapter.ProgramList{}
	}
	return settingsModelsForProvider(c.settings(), c.agentProvider)
}

// SubscribeSettingChanges starts listening to the settings that change. See settingsport.ChangeSubscriber.
func (c Config) SubscribeSettingChanges() (<-chan []settingsdomain.Setting, func()) {
	return subscribeSettingChanges(c.settings())
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func (c Config) SyncSetting() query.SyncSetting {
	return syncSetting(c.settings())
}

// ReadSettings returns a function that reads settings as JSON and returns the settings file path.
func (c Config) ReadSettings() func() (string, string, error) {
	return readSettings(c.settings())
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func (c Config) GetSettingsWriter() func(entries []settingsdomain.Setting) error {
	return getSettingsWriter(c.settings())
}

func (c Config) settingsFile() (string, error) {
	return settingsSettingsFile(c.settings())
}

// SetThemes gives the window the themes it can use: the catalog of themes, and the settings file that
// stores the selection. The window keeps the themes the binary ships when the answer is a fault.
func (c Config) SetThemes(inst *editor.Installation, say func(string)) error {
	service, err := settingsThemes(c.settings(), say)
	inst.SetThemes(service)
	return err
}

// SetShellThemes gives the shell the themes it can use, as SetThemes does the window.
func (c Config) SetShellThemes(api *shell.API, say func(string)) error {
	service, err := settingsThemes(c.settings(), say)
	api.Themes = service
	return err
}

// UnsubscribeHost stops the plugin host from receiving setting changes. It returns after the batch
// being delivered has been handled.
func (c Config) UnsubscribeHost() {
	unsubscribeHost(c.settings())
}

// agentAt is the key of the agent section in the settings file.
const agentAt = "agent"

// settingsWiring wires the settings, setting change events and themes of an installation.
type settingsWiring struct {
	SettingsPath              string
	ThemesPath                string
	RegistryPath              string
	InterfaceScale, TextScale float64
	Hardware                  hardware.Specs
	ResolvedCompute           hardware.ResolvedCompute
	PluginHost                *PluginHost
	KeyStore                  pluginport.KeyStore
	Watches                   *vaultwatch.Hub
	SettingsFiles             *settingsadapter.Files
	SettingEvents             *settingevents.Events
	Programs                  *settingsadapter.ProgramList
	SecretGate                *secret.Gate
	SubscribedHosts           *settingsusecase.HostSubscriptions
	PluginChanges             *pluginchanges.Signals
	ErrorHandler              func(error)
	AgentProvider             func() agentport.AgentProvider
	GetAgentPrograms          func(agentport.AgentProvider, settingsadapter.Agent) []agentdomain.Program
	SetSecretSource           func()
	SetCompleter              func()
}

func defaultSettings() settingsadapter.File {
	return settingsadapter.DefaultFile()
}

func settingsSettings(c settingsWiring) (settingsadapter.File, error) {
	path, err := settingsSettingsFile(c)
	if err != nil {
		return settingsadapter.File{}, err
	}
	return settingsStore(c).ReadCheckedAt(path)
}

func settingsGetSettingsAt(c settingsWiring, path string) (settingsadapter.File, error) {
	return settingsStore(c).ReadUncheckedAt(path)
}

func settingsGetSettingsOrDefaults(c settingsWiring) settingsadapter.File {
	if path, err := settingsSettingsFile(c); err == nil {
		if target, err := settingsStore(c).ReadCheckedAt(path); err == nil {
			return target
		}
	}
	return defaultSettings()
}

func readCheckedSettings(c settingsWiring) (settingsadapter.File, error) {
	return settingsStore(c).ReadCheckedFile()
}

func resolveCompute(_ settingsWiring, s settingsadapter.File) (hardware.Specs, hardware.ResolvedCompute) {
	specs := hardware.Detect()
	return specs, hardware.ResolveCompute(s.Performance.Profile, specs)
}

func configurePluginHost(c settingsWiring, s settingsadapter.File) {
	if c.PluginHost == nil {
		return
	}
	hosts := c.SubscribedHosts
	if hosts == nil {
		hosts = &settingsusecase.HostSubscriptions{}
	}
	var subscribe settingsport.ChangeSubscriber
	if c.SettingEvents != nil {
		subscribe = feed(c).Subscribe
	}
	settingsusecase.NewApplyToPlugins(settingsusecase.ApplyToPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Config: c.PluginHost, Write: settingsWriteSettings(c), Notify: notifyPlugins(c),
		Subscribe: subscribe, Hosts: hosts, Setup: func() { setUpPluginHost(c) },
	}).Execute(s.Plugins, c.ResolvedCompute.Profile)
}

func setUpPluginHost(c settingsWiring) {
	c.PluginHost.SetKeyStore(c.KeyStore)
	if c.SetSecretSource != nil {
		c.SetSecretSource()
	}
	if c.SetCompleter != nil {
		c.SetCompleter()
	}
}

func unsubscribeHost(c settingsWiring) {
	if c.SubscribedHosts == nil || c.PluginHost == nil {
		return
	}
	c.SubscribedHosts.Unsubscribe(c.PluginHost)
}

func notifyHost(c settingsWiring, batch []settingsdomain.Setting) {
	notifyPlugins(c).Execute(batch)
}

func settingsPlugins(c settingsWiring) func() []plugindomain.Info {
	return func() []plugindomain.Info {
		if c.PluginHost == nil {
			return nil
		}
		return c.PluginHost.GetPluginInfos()
	}
}

func settingsModelsForProvider(c settingsWiring, provider agentport.AgentProvider) func() []settingsdomain.Model {
	return func() []settingsdomain.Model {
		target := settingsGetSettingsOrDefaults(c)
		var programs []agentdomain.Program
		if c.GetAgentPrograms != nil {
			programs = c.GetAgentPrograms(provider, target.Agent)
		}
		return prefixModels(settingsadapter.OwnerApp, settingsadapter.AgentModels(agentAt, target.Agent, programs))
	}
}

func prefixModels(owner string, models []settingsdomain.Model) []settingsdomain.Model {
	for place := range models {
		models[place].Path = append([]string{owner}, models[place].Path...)
		for index := range models[place].Writes {
			models[place].Writes[index].Path = append([]string{owner}, models[place].Writes[index].Path...)
		}
	}
	return models
}

func getDeclarations(c settingsWiring) settingsadapter.Declarations {
	return func(owner string) ([]manifest.Setting, bool) {
		if owner == settingsadapter.OwnerApp {
			return settingsadapter.GetAppDeclaration(), true
		}
		if c.PluginHost == nil {
			return nil, false
		}
		return c.PluginHost.GetPluginSettingDeclarations(owner)
	}
}

func getRegistries(c settingsWiring) settingsadapter.Registries {
	return func(name string) ([]string, bool) {
		options, found := listRegistryOptions(c, name)
		names := make([]string, 0, len(options))
		for _, one := range options {
			names = append(names, one.Value)
		}
		return names, found
	}
}

func listRegistryOptions(c settingsWiring, name string) (manifest.SettingOptions, bool) {
	if name != settingsadapter.RegistryThemes {
		return nil, false
	}
	files, err := openThemeFiles(c)
	if err != nil {
		settingsHandleError(c, err)
	}
	var options manifest.SettingOptions
	for _, one := range themeusecase.NewListThemes(getThemeSources(c, files)).Execute() {
		options = append(options, manifest.SettingOption{Value: one.Name, Title: one.Title})
	}
	return options, true
}

func listAppSettings(c settingsWiring) []plugindomain.Setting {
	return settingsusecase.NewListAppSettings(
		settingsadapter.JSONCodec{}, settingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{},
		settingsadapter.RegistryOptions(func(name string) (manifest.SettingOptions, bool) { return listRegistryOptions(c, name) }),
	).Execute()
}

func subscribeSettingChanges(c settingsWiring) (<-chan []settingsdomain.Setting, func()) {
	return feed(c).Subscribe()
}

func refreshSettingEvents(c settingsWiring) {
	feed(c).Refresh()
}

func syncSetting(c settingsWiring) query.SyncSetting {
	return func() query.SyncTitleAndFilename {
		target, err := settingsSettings(c)
		if err != nil {
			settingsHandleError(c, fmt.Errorf("read settings: %w", err))
		}
		return query.SyncTitleAndFilename(target.Sync())
	}
}

func readSettings(c settingsWiring) func() (string, string, error) {
	return settingsStore(c).ReadJSON
}

func getSettingsWriter(c settingsWiring) func(entries []settingsdomain.Setting) error {
	return settingsWriteSettings(c).Execute
}

func settingsSettingsFile(c settingsWiring) (string, error) {
	if c.SettingsPath != "" {
		return c.SettingsPath, nil
	}
	if file, selected := getAdjacentPath(c, "numen.json"); selected {
		return file, nil
	}
	return settingsadapter.Path()
}

func getAdjacentPath(c settingsWiring, name string) (string, bool) {
	if c.RegistryPath == "" {
		return "", false
	}
	return filepath.Join(filepath.Dir(c.RegistryPath), name), true
}

func openWriteSettings(c settingsWiring) *settingsusecase.WriteSettings {
	return settingsWriteSettings(c)
}

func openWatchSettings(c settingsWiring) *settingsusecase.WatchSettingChanges {
	return settingsusecase.NewWatchSettingChanges(func() (<-chan []settingsdomain.Setting, func()) { return subscribeSettingChanges(c) })
}

func openChoosePluginPath(c settingsWiring, dialog machineport.PathDialog) *settingsusecase.ChoosePluginPath {
	return settingsusecase.NewChoosePluginPath(dialog, settingsPlugins(c), getSettingsWriter(c))
}

func openRetryPlugin(_ settingsWiring, retrier func(context.Context, string) (*plugindomain.Failure, error)) *settingsusecase.RetryPlugin {
	retry, err := settingsusecase.NewRetryPlugin(retrier)
	if err != nil {
		return nil
	}
	return retry
}

func settingsStore(c settingsWiring) *settingsadapter.Store {
	var secrets settingsadapter.SecretDeclarations
	if c.PluginHost != nil {
		secrets = c.PluginHost
	}
	return settingsadapter.NewStore(c.SettingsFiles, func() (string, error) { return settingsSettingsFile(c) }, getDeclarations(c), getRegistries(c), secrets)
}

func feed(c settingsWiring) settingevents.Feed {
	list := listSettingValues(c)
	return settingevents.Feed{
		Events:  c.SettingEvents,
		GetPath: func() (string, error) { return settingsSettingsFile(c) },
		Read:    list.Execute,
		Initial: func() settingsdomain.Values {
			values, err := list.ListDefaults()
			if err != nil {
				settingsHandleError(c, err)
			}
			return values
		},
		Report: func(err error) { settingsHandleError(c, err) },
		Hub:    c.Watches.GetShared(),
	}
}

func listSettingValues(c settingsWiring) *settingsusecase.ListSettingValues {
	var plugins settingsport.PluginInfos
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	var late settingsport.LatePrograms
	if c.Programs != nil {
		late = c.Programs
	}
	return settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, settingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, plugins, late,
	)
}

func settingsWriteSettings(c settingsWiring) *settingsusecase.WriteSettings {
	var plugins settingsport.PluginSettings
	if c.PluginHost != nil {
		plugins = c.PluginHost
	}
	check := settingsusecase.NewCheckSettings(
		settingsadapter.JSONCodec{}, settingsadapter.NewValidator(getDeclarations(c), getRegistries(c)), plugins,
	)
	return settingsusecase.NewWriteSettings(check, settingsStore(c), feed(c))
}

func notifyPlugins(c settingsWiring) *settingsusecase.NotifyPlugins {
	var signal func()
	if c.PluginChanges != nil {
		signal = c.PluginChanges.Signal
	}
	return settingsusecase.NewNotifyPlugins(
		settingsadapter.JSONCodec{}, c.PluginHost, func(ctx context.Context) error { return refreshPluginSecrets(c, ctx) },
		func(profile string) string { return resolveProfile(c, profile) }, signal, func(err error) { settingsHandleError(c, err) },
	)
}

func refreshPluginSecrets(c settingsWiring, ctx context.Context) error {
	if err := c.SecretGate.Enter(ctx); err != nil {
		return err
	}
	defer c.SecretGate.Leave()
	return c.PluginHost.Refresh(ctx)
}

func resolveProfile(c settingsWiring, profile string) string {
	return hardware.ResolveCompute(profile, c.Hardware).Profile
}

func settingsHandleError(c settingsWiring, err error) {
	if c.ErrorHandler != nil {
		c.ErrorHandler(err)
	}
}

func settingsThemes(c settingsWiring, say func(string)) (*theme.Service, error) {
	files, err := openThemeFiles(c)
	sources := getThemeSources(c, files)
	store := settingsadapter.NewAppearanceStore(
		settingsusecase.NewReadAppearance(settingsStore(c)).Execute,
		settingsusecase.NewWriteAppearance(settingsadapter.JSONCodec{}, settingsWriteSettings(c)).Execute,
		say,
	)
	scales := themeusecase.NewScales(c.InterfaceScale, c.TextScale)
	interfaceScale, textScale := settingsadapter.GetInterfaceScaleBounds(), settingsadapter.GetTextScaleBounds()
	return theme.NewService(
		themeusecase.NewGetThemes(sources, store, store, scales,
			themeusecase.Bounds{Min: interfaceScale.Min, Max: interfaceScale.Max},
			themeusecase.Bounds{Min: textScale.Min, Max: textScale.Max}),
		themeusecase.NewReadTheme(sources),
		themeusecase.NewWriteAppearance(sources, store, store, scales),
		themeusecase.NewWatchThemes(sources, files, 0),
	), err
}

// CheckInterfaceScale returns an error when scale is outside the interface scale bounds.
func CheckInterfaceScale(name string, scale float64) error {
	return settingsadapter.GetInterfaceScaleBounds().Check(name, scale)
}

// CheckTextScale returns an error when scale is outside the text scale bounds.
func CheckTextScale(name string, scale float64) error {
	return settingsadapter.GetTextScaleBounds().Check(name, scale)
}

func getThemeSources(c settingsWiring, files themefiles.Files) themeusecase.Sources {
	sources := themeusecase.Sources{Presets: files, Own: files, Plugins: nil}
	if c.PluginHost != nil {
		sources.Plugins = themebridge.Open(c.PluginHost)
	}
	return sources
}

func openThemeFiles(c settingsWiring) (themefiles.Files, error) {
	if c.ThemesPath != "" {
		return themefiles.OpenAt(c.ThemesPath, c.Watches.GetShared())
	}
	if folder, selected := getAdjacentPath(c, "themes"); selected {
		return themefiles.OpenAt(folder, c.Watches.GetShared())
	}
	return themefiles.Open(c.Watches.GetShared())
}
