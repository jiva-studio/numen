package container

import (
	"cmp"
	"fmt"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/none"
	settingsadapter "github.com/jiva-studio/numen/libs/core/kernel/adapter/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/theme"
	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	machinedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	themedomain "github.com/jiva-studio/numen/libs/core/kernel/domain/theme"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwatch"
	machineadapter "github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/machine"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/settingevents"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/themefiles"
	"github.com/jiva-studio/numen/libs/core/kernel/port/connectionport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"

	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
	themeusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/theme"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// ApplySettings reads the settings file and returns the configuration with its values applied, and
// the notices that reading it raised. An unconfigured installation applies the default settings.
func (c Config) ApplySettings(exec *PluginExecution, prefs *PluginPreferences, res *Resources) (Config, []string, error) {
	selected, err := settingsSettings(settingsOf(c, exec, prefs, res))
	if err != nil {
		return c, nil, err
	}
	return applySettingsFile(c, exec, prefs, res, selected), selected.Notices, nil
}

func applySettingsFile(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources, s settingsadapter.File) Config {
	c.machine = Machine{ResolvedCompute: settingsusecase.NewResolveCompute(machineadapter.SpecsReader{}).Execute(s.Performance.Profile)}
	prefs = getPluginPreferences(prefs)
	configurePluginHost(settingsOf(c, exec, prefs, res), prefs, &res.subscribedHosts, s, c.machine.ResolvedCompute)
	return c
}

// settingsWiring wires the settings, setting change events and themes of an installation.
type settingsWiring struct {
	SettingsPath  string
	ThemesPath    string
	RegistryPath  string
	Watches       *vaultwatch.Hub
	SettingsFiles *settingsadapter.Files
	SettingEvents *settingevents.Events

	PluginChanges   *settingsusecase.PluginChanges
	Infos           themefiles.PluginHost
	Declarations    pluginSettingDeclarer
	ConnectionTypes connectionport.Types
	PluginInfos     settingsusecase.PluginInfos
	Checks          settingsusecase.PluginSettings
	Detection       connectionport.Detected
	ErrorHandler    func(error)
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

func configurePluginHost(c settingsWiring, prefs *PluginPreferences, hosts *settingsusecase.HostSubscriptions, s settingsadapter.File, compute machinedomain.ResolvedCompute) {
	var subscribe settingsusecase.ChangeSubscriber
	if c.SettingEvents != nil {
		subscribe = feed(c).Subscribe
	}
	settingsusecase.NewApplyToPlugins(settingsusecase.ApplyToPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Config: prefs.Configuration, Write: openWriteSettings(c), Notify: notifyPlugins(c, prefs),
		Subscribe: prefs.Views.Follow(subscribe), Hosts: hosts,
	}).Execute(s.Plugins, getPluginPerformance(compute))
}

func getDeclarations(c settingsWiring) settingsdomain.Declarations {
	return func(owner string) ([]manifest.Setting, bool) {
		if owner == settingsdomain.AppOwner {
			return settingsadapter.GetAppDeclaration(), true
		}
		return c.Declarations.GetPluginSettingDeclarations(owner)
	}
}

func getRegistries(c settingsWiring) settingsdomain.Registries {
	themes := themeRegistry(c).GetNames()
	return settingsadapter.AddDetectedRegistries(func(name string) ([]string, bool) {
		if name == settingsdomain.RegistryConnections {
			return listConnectionIDs(c)
		}
		return themes(name)
	}, c.ConnectionTypes, c.Detection)
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

func writeLogLevel(c settingsWiring) func(domainlogs.Level) error {
	write := openWriteSettings(c)
	return func(level domainlogs.Level) error {
		return write.Execute([]settingsdomain.Setting{{
			Path: []string{settingsdomain.AppOwner, "logging", "level"},
			JSON: strconv.Quote(level.String()),
		}})
	}
}

func readSettings(c settingsWiring) func() (settingsdomain.Snapshot, error) {
	return settingsStore(c).ReadJSON
}

func getSettingsWriter(c settingsWiring) func(entries []settingsdomain.Setting) error {
	return openWriteSettings(c).Execute
}

func settingsSettingsFile(c settingsWiring) (string, error) {
	return settingsadapter.LocateFile(c.SettingsPath, c.RegistryPath)
}

func openWatchSettings(c settingsWiring) *settingsusecase.WatchSettingChanges {
	return settingsusecase.NewWatchSettingChanges(func() (<-chan []settingsdomain.Setting, func()) { return feed(c).Subscribe() })
}

func openChoosePluginPath(c settingsWiring, dialog settingsusecase.PathDialog) *settingsusecase.ChoosePluginPath {
	return settingsusecase.NewChoosePluginPath(dialog, c.Infos.GetPluginInfos, getSettingsWriter(c))
}

func settingsStore(c settingsWiring) *settingsadapter.Store {
	var declared settingsdomain.Declarations
	if c.PluginInfos != nil {
		declared = getDeclarations(c)
	}
	return settingsadapter.NewStore(c.SettingsFiles, func() (string, error) { return settingsSettingsFile(c) }, declared, getRegistries(c))
}

func feed(c settingsWiring) settingevents.Feed {
	list := listSettingValues(c)
	return settingevents.Feed{
		Events:  c.SettingEvents,
		GetPath: func() (string, error) { return settingsSettingsFile(c) },
		Read: func() (settingsdomain.Values, error) {
			values, err := list.Execute()
			settingsadapter.AddDetectedConnectionIDs(values, c.Detection)
			return values, err
		},
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
		settingsadapter.JSONCodec{}, settingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, cmp.Or[settingsusecase.PluginInfos](c.PluginInfos, none.PluginInfos{}),
	)
}

func openWriteSettings(c settingsWiring) *settingsusecase.WriteSettings {
	check := settingsusecase.NewCheckSettings(
		settingsadapter.JSONCodec{}, getDeclarations(c), getRegistries(c), cmp.Or[settingsusecase.PluginSettings](c.Checks, none.PluginSettings{}),
	)
	return settingsusecase.NewWriteSettings(check, settingsStore(c), feed(c))
}

func notifyPlugins(c settingsWiring, prefs *PluginPreferences) *settingsusecase.NotifyPlugins {
	signal := c.PluginChanges.Signal
	return settingsusecase.NewNotifyPlugins(settingsusecase.NotifyPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Apply: prefs.ApplySetting, Config: prefs.Configuration,
		ResolveProfile: resolveProfile, Signal: signal,
		Report: func(err error) { c.ErrorHandler(err) },
	})
}

func resolveProfile(profile string) plugindomain.Performance {
	return getPluginPerformance(settingsusecase.NewResolveCompute(machineadapter.SpecsReader{}).Execute(profile))
}

func settingsThemes(c settingsWiring, scales Scales, say func(string)) (*theme.Service, error) {
	files, err := openThemeFiles(c)
	sources := getThemeSources(c, files)
	store := settingsadapter.NewAppearanceStore(
		settingsusecase.NewReadAppearance(settingsStore(c)).Execute,
		settingsusecase.NewWriteAppearance(settingsadapter.JSONCodec{}, openWriteSettings(c)).Execute,
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
	return themeusecase.Sources{Presets: files, Own: files, Plugins: themefiles.NewPluginThemes(c.Infos)}
}

func openThemeFiles(c settingsWiring) (themefiles.Files, error) {
	if folder, isSet := settingsadapter.LocateThemes(c.ThemesPath, c.RegistryPath); isSet {
		return themefiles.OpenAt(folder, c.Watches.GetShared())
	}
	return themefiles.Open(c.Watches.GetShared())
}

func getPluginPerformance(compute machinedomain.ResolvedCompute) plugindomain.Performance {
	return plugindomain.Performance{Profile: compute.Profile, Threads: compute.CoreBudget, Nice: compute.Nice}
}

func (c Config) openThemes(settingsDeps settingsWiring, tasks *run.Tasks) *theme.Service {
	service, err := settingsThemes(settingsDeps, c.Scales, editor.NewThemeWarning(tasks))
	if err != nil {
		settingsDeps.ErrorHandler(err)
	}
	return service
}
