package container

import (
	"context"
	"fmt"

	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	machineadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
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
	c.machine = Machine{ResolvedCompute: settingsusecase.NewResolveCompute(machineadapter.SpecsReader{}).Execute(s.Performance.Profile)}
	configurePluginHost(settingsOf(c), s)
	return c
}

// SyncSetting returns whether note title and filename synchronization is enabled.
func SyncSetting(c Config) naming.SyncSetting {
	return syncSetting(settingsOf(c))
}

// GetSettingsWriter patches specific settings paths in the configuration file. The readers of the
// setting-changed event receive the settings whose value changed.
func GetSettingsWriter(c Config) func(entries []settingsdomain.Setting) error {
	return getSettingsWriter(settingsOf(c))
}

// SetThemes gives the window the themes it can use: the catalog of themes, and the settings file that
// stores the selection. The window keeps the themes the binary ships when the answer is a fault.
func SetThemes(c Config, inst *editor.Installation, scales Scales, say func(string)) error {
	service, err := settingsThemes(settingsOf(c), scales, say)
	inst.SetThemes(service)
	return err
}

func UnsubscribeHost(c Config) {
	unsubscribeHost(settingsOf(c))
}

type settingsPlaces struct {
	SettingsPath    string
	ThemesPath      string
	RegistryPath    string
	ResolvedCompute machinedomain.ResolvedCompute
}

type settingsStores struct {
	Watches         *vaultwatch.Hub
	SettingsFiles   *settingsadapter.Files
	SettingEvents   *settingevents.Events
	SubscribedHosts *settingsusecase.HostSubscriptions
	PluginChanges   *settingsusecase.PluginChanges
}

// settingsWiring wires the settings, setting change events and themes of an installation.
type settingsWiring struct {
	settingsPlaces
	settingsStores
	PluginHost   *PluginHost
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

func readCheckedSettings(c settingsWiring) (settingsadapter.File, error) {
	return settingsStore(c).ReadCheckedFile()
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
		Subscribe: c.PluginHost.Views.Follow(subscribe), Hosts: hosts,
	}).Execute(s.Plugins, c.ResolvedCompute.Profile)
}

func unsubscribeHost(c settingsWiring) {
	if c.SubscribedHosts == nil {
		return
	}
	c.SubscribedHosts.Unsubscribe(c.PluginHost.Configuration)
}

func settingsPlugins(c settingsWiring) func() []plugindomain.Info {
	return c.PluginHost.Runtime.GetPluginInfos
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
	themes := themeRegistry(c).GetNames()
	return func(name string) ([]string, bool) {
		if name == settingsadapter.RegistryConnections {
			return listConnectionIDs(c)
		}
		return themes(name)
	}
}

func listConnectionIDs(c settingsWiring) ([]string, bool) {
	ids, err := settingsStore(c).ListConnectionIDs()
	if err != nil {
		c.ErrorHandler(err)
		return nil, false
	}
	return ids, true
}

func themeRegistry(c settingsWiring) settingsadapter.RegistryOptions {
	return settingsadapter.ThemeRegistry(func() []themedomain.Theme { return listThemes(c) })
}

func listThemes(c settingsWiring) []themedomain.Theme {
	files, err := openThemeFiles(c)
	if err != nil {
		c.ErrorHandler(err)
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

func syncSetting(c settingsWiring) naming.SyncSetting {
	return func() naming.SyncTitleAndFilename {
		target, err := settingsSettings(c)
		if err != nil {
			c.ErrorHandler(fmt.Errorf("read settings: %w", err))
		}
		return naming.SyncTitleAndFilename(target.Sync())
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
	return settingsusecase.NewRetryPlugin(retrier)
}

func settingsStore(c settingsWiring) *settingsadapter.Store {
	var declared settingsadapter.Declarations
	if c.PluginHost.Views.Infos != nil {
		declared = getDeclarations(c)
	}
	return settingsadapter.NewStore(c.SettingsFiles, func() (string, error) { return settingsSettingsFile(c) }, declared, getRegistries(c))
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
				c.ErrorHandler(err)
			}
			return values
		},
		Report: func(err error) { c.ErrorHandler(err) },
		Hub:    c.Watches.GetShared(),
	}
}

func listSettingValues(c settingsWiring) *settingsusecase.ListSettingValues {
	return settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, settingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, c.PluginHost.Views.Infos,
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
		ResolveProfile: resolveProfile, Signal: signal,
		Report: func(err error) { c.ErrorHandler(err) },
	})
}

func resolveProfile(profile string) string {
	return settingsusecase.NewResolveCompute(machineadapter.SpecsReader{}).Execute(profile).Profile
}

func settingsThemes(c settingsWiring, scales Scales, say func(string)) (*theme.Service, error) {
	files, err := openThemeFiles(c)
	sources := getThemeSources(c, files)
	store := settingsadapter.NewAppearanceStore(
		settingsusecase.NewReadAppearance(settingsStore(c)).Execute,
		settingsusecase.NewWriteAppearance(settingsadapter.JSONCodec{}, settingsWriteSettings(c)).Execute,
		say,
	)
	sizes := themeusecase.NewScales(scales.InterfaceScale, scales.TextScale)
	interfaceScale, textScale := settingsadapter.GetInterfaceScaleBounds(), settingsadapter.GetTextScaleBounds()
	return theme.NewService(
		themeusecase.NewGetThemes(sources, store, store, sizes,
			themeusecase.Bounds{Min: interfaceScale.Min, Max: interfaceScale.Max},
			themeusecase.Bounds{Min: textScale.Min, Max: textScale.Max}),
		themeusecase.NewReadTheme(sources),
		themeusecase.NewWriteAppearance(sources, store, store, sizes),
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
	return themeusecase.Sources{Presets: files, Own: files, Plugins: themefiles.NewPluginThemes(c.PluginHost.Runtime)}
}

func openThemeFiles(c settingsWiring) (themefiles.Files, error) {
	if folder, isSet := settingsadapter.LocateThemes(c.ThemesPath, c.RegistryPath); isSet {
		return themefiles.OpenAt(folder, c.Watches.GetShared())
	}
	return themefiles.Open(c.Watches.GetShared())
}
