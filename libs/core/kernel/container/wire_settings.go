package container

import (
	"context"
	"fmt"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/settings/hardware"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	agentdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/agent"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/themebridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
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

// ApplySettings reads the settings file and returns the configuration with its values applied, and
// the notices that reading it raised. An unconfigured installation applies the default settings.
func (c Config) ApplySettings() (Config, []string, error) {
	selected, err := settingsSettings(settingsOf(c))
	if err != nil {
		return c, nil, err
	}
	return applySettingsFile(c, selected), selected.Notices, nil
}

func applySettingsFile(c Config, s settingsadapter.File) Config {
	c.Hardware, c.ResolvedCompute = resolveCompute(settingsOf(c), s)
	configurePluginHost(settingsOf(c), s)
	return c
}

func models(c Config) func() []settingsdomain.Model {
	return modelsForProvider(c, pluginsAgentProvider(plugins(c)))
}

func modelsForProvider(c Config, provider agentport.AgentProvider) func() []settingsdomain.Model {
	return settingsModelsForProvider(settingsOf(c), provider)
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func SyncSetting(c Config) query.SyncSetting {
	return syncSetting(settingsOf(c))
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func GetSettingsWriter(c Config) func(entries []settingsdomain.Setting) error {
	return getSettingsWriter(settingsOf(c))
}

// SetThemes gives the window the themes it can use: the catalog of themes, and the settings file that
// stores the selection. The window keeps the themes the binary ships when the answer is a fault.
func SetThemes(c Config, inst *editor.Installation, say func(string)) error {
	service, err := settingsThemes(settingsOf(c), say)
	inst.SetThemes(service)
	return err
}

func UnsubscribeHost(c Config) {
	unsubscribeHost(settingsOf(c))
}

// agentAt is the key of the agent section in the settings file.
const agentAt = "agent"

// settingsWiring wires the settings, setting change events and themes of an installation.
type settingsPlaces struct {
	SettingsPath              string
	ThemesPath                string
	RegistryPath              string
	InterfaceScale, TextScale float64
	Hardware                  settingsdomain.MachineSpecs
	ResolvedCompute           settingsdomain.ResolvedCompute
}

type settingsStores struct {
	Watches         *vaultwatch.Hub
	SettingsFiles   *settingsadapter.Files
	SettingEvents   *settingevents.Events
	Programs        *settingsadapter.ProgramList
	SecretGate      *secret.Gate
	SubscribedHosts *settingsusecase.HostSubscriptions
	PluginChanges   *settingsusecase.PluginChanges
}

type settingsPluginRefs struct {
	PluginHost       *PluginHost
	KeyStore         pluginport.KeyStore
	Plugins          func() pluginsWiring
	GetAgentPrograms func(agentport.AgentProvider, settingsadapter.Agent) []agentdomain.Program
}

type settingsWiring struct {
	settingsPlaces
	settingsStores
	settingsPluginRefs
	ErrorHandler func(error)
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

func resolveCompute(_ settingsWiring, s settingsadapter.File) (settingsdomain.MachineSpecs, settingsdomain.ResolvedCompute) {
	compute := settingsusecase.NewResolveCompute(hardware.Reader{}).Execute(s.Performance.Profile)
	return compute.Machine, compute.Resolved
}

func configurePluginHost(c settingsWiring, s settingsadapter.File) {
	hosts := c.SubscribedHosts
	if hosts == nil {
		hosts = &settingsusecase.HostSubscriptions{}
	}
	var subscribe settingsport.ChangeSubscriber
	if c.SettingEvents != nil {
		subscribe = feed(c).Subscribe
	}
	settingsusecase.NewApplyToPlugins(settingsusecase.ApplyToPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Config: c.PluginHost.Configuration, Write: settingsWriteSettings(c), Notify: notifyPlugins(c),
		Subscribe: c.PluginHost.Views.Follow(subscribe), Hosts: hosts, Setup: func() { setUpPluginHost(c) },
	}).Execute(s.Plugins, c.ResolvedCompute.Profile)
}

func setUpPluginHost(c settingsWiring) {
	c.PluginHost.Configuration.SetKeyStore(c.KeyStore)
	if c.Plugins != nil {
		setSecretSource(c.Plugins())
		setCompleter(c.Plugins())
	}
}

func unsubscribeHost(c settingsWiring) {
	if c.SubscribedHosts == nil {
		return
	}
	c.SubscribedHosts.Unsubscribe(c.PluginHost.Configuration)
}

func notifyHost(c settingsWiring, batch []settingsdomain.Setting) {
	notifyPlugins(c).Execute(batch)
}

func settingsPlugins(c settingsWiring) func() []plugindomain.Info {
	return c.PluginHost.Runtime.GetPluginInfos
}

func settingsModelsForProvider(c settingsWiring, provider agentport.AgentProvider) func() []settingsdomain.Model {
	return settingsusecase.NewListAppModels(func() []settingsdomain.Model {
		target := settingsGetSettingsOrDefaults(c)
		return settingsadapter.AgentModels(agentAt, target.Agent, c.GetAgentPrograms(provider, target.Agent))
	}).Execute
}

func getDeclarations(c settingsWiring) settingsadapter.Declarations {
	return func(owner string) ([]manifest.Setting, bool) {
		if owner == settingsdomain.AppOwner {
			return settingsadapter.GetAppDeclaration(), true
		}
		return c.PluginHost.Configuration.GetPluginSettingDeclarations(owner)
	}
}

func getRegistries(c settingsWiring) settingsadapter.Registries {
	return themeRegistry(c).GetNames()
}

func themeRegistry(c settingsWiring) settingsadapter.RegistryOptions {
	return settingsadapter.ThemeRegistry(func() []themedomain.Theme { return listThemes(c) })
}

func listThemes(c settingsWiring) []themedomain.Theme {
	files, err := openThemeFiles(c)
	if err != nil {
		settingsHandleError(c, err)
	}
	return themeusecase.NewListThemes(getThemeSources(c, files)).Execute()
}

func listAppSettings(c settingsWiring) []plugindomain.Setting {
	return settingsusecase.NewListAppSettings(
		settingsadapter.JSONCodec{}, settingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{},
		themeRegistry(c),
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
	return settingsadapter.LocateFile(c.SettingsPath, c.RegistryPath)
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
	return settingsadapter.NewStore(c.SettingsFiles, func() (string, error) { return settingsSettingsFile(c) }, getDeclarations(c), getRegistries(c), c.PluginHost.Views.Secrets)
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
	return settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, settingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, c.PluginHost.Views.Infos, c.Programs,
	)
}

func settingsWriteSettings(c settingsWiring) *settingsusecase.WriteSettings {
	check := settingsusecase.NewCheckSettings(
		settingsadapter.JSONCodec{}, settingsadapter.NewValidator(getDeclarations(c), getRegistries(c)), c.PluginHost.Views.Checks,
	)
	return settingsusecase.NewWriteSettings(check, settingsStore(c), feed(c))
}

func notifyPlugins(c settingsWiring) *settingsusecase.NotifyPlugins {
	signal := c.PluginChanges.Signal
	return settingsusecase.NewNotifyPlugins(settingsusecase.NotifyPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Apply: c.PluginHost.ApplySetting, Config: c.PluginHost.Configuration,
		RefreshSecrets: func(ctx context.Context) error { return refreshPluginSecrets(c, ctx) },
		ResolveProfile: resolveProfile, Signal: signal,
		Report: func(err error) { settingsHandleError(c, err) },
	})
}

func refreshPluginSecrets(c settingsWiring, ctx context.Context) error {
	if err := c.SecretGate.Enter(ctx); err != nil {
		return err
	}
	defer c.SecretGate.Leave()
	return c.PluginHost.Configuration.Refresh(ctx)
}

func resolveProfile(profile string) string {
	return settingsusecase.NewResolveCompute(hardware.Reader{}).Execute(profile).Resolved.Profile
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
	return settingsusecase.NewCheckScale(settingsadapter.GetInterfaceScaleBounds()).Execute(name, scale)
}

// CheckTextScale returns an error when scale is outside the text scale bounds.
func CheckTextScale(name string, scale float64) error {
	return settingsusecase.NewCheckScale(settingsadapter.GetTextScaleBounds()).Execute(name, scale)
}

func getThemeSources(c settingsWiring, files themefiles.Files) themeusecase.Sources {
	return themeusecase.Sources{Presets: files, Own: files, Plugins: themebridge.Open(c.PluginHost.Runtime)}
}

func openThemeFiles(c settingsWiring) (themefiles.Files, error) {
	if folder, isSet := settingsadapter.LocateThemes(c.ThemesPath, c.RegistryPath); isSet {
		return themefiles.OpenAt(folder, c.Watches.GetShared())
	}
	return themefiles.Open(c.Watches.GetShared())
}
