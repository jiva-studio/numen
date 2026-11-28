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
	selected, err := createSettings(createSettingsWiring(c, exec, prefs, res))
	if err != nil {
		return c, nil, err
	}
	return applySettingsFile(c, exec, prefs, res, selected), selected.Header.Notices, nil
}

func applySettingsFile(c Config, exec *PluginExecution, prefs *PluginPreferences, res *Resources, s settingsadapter.File) Config {
	c.machine = Machine{ResolvedCompute: settingsusecase.NewResolveCompute(machineadapter.SpecsReader{}).Execute(s.Performance.Profile)}
	prefs = getPluginPreferences(prefs)
	configurePluginHost(createSettingsWiring(c, exec, prefs, res), prefs, &res.subscriptions.Hosts, s, c.machine.ResolvedCompute)
	return c
}

// settingsParts wires the settings, setting change events and themes of an installation.
type settingsParts struct {
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
	Detection       connectionport.DetectedConnections
	HandleError     func(error)
}

func createSettings(c settingsParts) (settingsadapter.File, error) {
	path, err := getSettingsFile(c)
	if err != nil {
		return settingsadapter.File{}, err
	}
	return createSettingsStore(c).ReadCheckedAt(path)
}

func configurePluginHost(c settingsParts, prefs *PluginPreferences, hosts *settingsusecase.HostSubscriptions, s settingsadapter.File, compute machinedomain.ResolvedCompute) {
	var subscribe settingsusecase.ChangeSubscriber
	if c.SettingEvents != nil {
		subscribe = getFeed(c).Subscribe
	}
	settingsusecase.NewApplyToPlugins(settingsusecase.ApplyToPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Config: prefs.Configuration, Write: openWriteSettings(c), Notify: notifyPlugins(c, prefs),
		Subscribe: prefs.Views.Follow(subscribe), Hosts: hosts,
	}).Execute(s.Plugins, getPluginPerformance(compute))
}

func getDeclarations(c settingsParts) settingsdomain.Declarations {
	return func(owner string) ([]manifest.Setting, bool) {
		if owner == settingsdomain.AppOwner {
			return settingsadapter.GetAppDeclaration(), true
		}
		return c.Declarations.GetPluginSettingDeclarations(owner)
	}
}

func getRegistries(c settingsParts) settingsdomain.Registries {
	themes := getThemeRegistry(c).GetNames()
	return settingsadapter.AddDetectedRegistries(func(name string) ([]string, bool) {
		if name == settingsdomain.RegistryConnections {
			ids, err := createSettingsStore(c).ListConnectionIDs()
			if err != nil {
				c.HandleError(err)
				return nil, false
			}
			return ids, true
		}
		return themes(name)
	}, c.ConnectionTypes, c.Detection)
}

func getThemeRegistry(c settingsParts) settingsadapter.RegistryOptions {
	return settingsadapter.BuildThemeRegistry(func() []themedomain.Theme { return listThemes(c) })
}

func listThemes(c settingsParts) []themedomain.Theme {
	files, err := openThemeFiles(c)
	if err != nil {
		c.HandleError(err)
	}
	return themeusecase.NewListThemes(getThemeSources(c, files)).Execute()
}

func listAppSettings(c settingsParts) []plugindomain.Setting {
	return settingsusecase.NewListAppSettings(
		settingsadapter.JSONCodec{}, createSettingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{},
		getThemeRegistry(c),
	).Execute()
}

func syncSetting(c settingsParts) naming.SyncSetting {
	return func() naming.SyncTitleAndFilename {
		target, err := createSettings(c)
		if err != nil {
			c.HandleError(fmt.Errorf("read settings: %w", err))
		}
		return naming.SyncTitleAndFilename(target.Sync())
	}
}

func writeLogLevel(c settingsParts) func(domainlogs.Level) error {
	write := openWriteSettings(c)
	return func(level domainlogs.Level) error {
		return write.Execute([]settingsdomain.Setting{{
			Path: []string{settingsdomain.AppOwner, "logging", "level"},
			JSON: strconv.Quote(level.String()),
		}})
	}
}

func getSettingsWriter(c settingsParts) func(entries []settingsdomain.Setting) error {
	return openWriteSettings(c).Execute
}

func getSettingsFile(c settingsParts) (string, error) {
	return settingsadapter.LocateFile(c.SettingsPath, c.RegistryPath)
}

func openWatchSettings(c settingsParts) *settingsusecase.WatchSettingChanges {
	return settingsusecase.NewWatchSettingChanges(func() (<-chan []settingsdomain.Setting, func()) { return getFeed(c).Subscribe() })
}

func createSettingsStore(c settingsParts) *settingsadapter.Store {
	var declared settingsdomain.Declarations
	if c.PluginInfos != nil {
		declared = getDeclarations(c)
	}
	return settingsadapter.NewStore(c.SettingsFiles, func() (string, error) { return getSettingsFile(c) }, declared, getRegistries(c))
}

func getFeed(c settingsParts) settingevents.Feed {
	list := settingsusecase.NewListSettingValues(
		settingsadapter.JSONCodec{}, createSettingsStore(c), settingsadapter.Defaults{}, settingsadapter.AppDeclaration{}, cmp.Or[settingsusecase.PluginInfos](c.PluginInfos, none.PluginInfos{}),
	)
	return settingevents.Feed{
		Events:  c.SettingEvents,
		GetPath: func() (string, error) { return getSettingsFile(c) },
		Read: func() (settingsdomain.Values, error) {
			values, err := list.Execute()
			settingsadapter.AddDetectedConnectionIDs(values, c.Detection)
			return values, err
		},
		GetInitial: func() settingsdomain.Values {
			values, err := list.ListDefaults()
			if err != nil {
				c.HandleError(err)
			}
			return values
		},
		Report: func(err error) { c.HandleError(err) },
		Hub:    c.Watches.GetShared(),
	}
}

func openWriteSettings(c settingsParts) *settingsusecase.WriteSettings {
	check := settingsusecase.NewCheckSettings(
		settingsadapter.JSONCodec{}, getDeclarations(c), getRegistries(c), cmp.Or[settingsusecase.PluginSettings](c.Checks, none.PluginSettings{}),
	)
	return settingsusecase.NewWriteSettings(check, createSettingsStore(c), getFeed(c))
}

func notifyPlugins(c settingsParts, prefs *PluginPreferences) *settingsusecase.NotifyPlugins {
	return settingsusecase.NewNotifyPlugins(settingsusecase.NotifyPluginsDeps{
		Codec: settingsadapter.JSONCodec{}, Apply: prefs.ApplySetting, Config: prefs.Configuration,
		ResolveProfile: resolveProfile, Signal: c.PluginChanges.Signal,
		Report: func(err error) { c.HandleError(err) },
	})
}

func resolveProfile(profile string) plugindomain.Performance {
	return getPluginPerformance(settingsusecase.NewResolveCompute(machineadapter.SpecsReader{}).Execute(profile))
}

func createThemeService(c settingsParts, scales Scales, say func(string)) (*theme.Service, error) {
	files, err := openThemeFiles(c)
	sources := getThemeSources(c, files)
	store := settingsadapter.NewAppearanceStore(
		settingsusecase.NewReadAppearance(createSettingsStore(c)).Execute,
		settingsusecase.NewWriteAppearance(settingsadapter.JSONCodec{}, openWriteSettings(c)).Execute,
		say,
	)
	sizes := themeusecase.NewScales(scales.InterfaceScale, scales.TextScale)
	return theme.NewService(
		themeusecase.NewGetThemes(sources, store, store, sizes,
			themeusecase.Bounds(settingsadapter.GetInterfaceScaleBounds()),
			themeusecase.Bounds(settingsadapter.GetTextScaleBounds())),
		themeusecase.NewReadTheme(sources),
		themeusecase.NewWriteAppearance(sources, store, store, sizes),
		themeusecase.NewWatchThemes(sources, files, 0, getPluginChangeSubscriber(c)),
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

func getThemeSources(c settingsParts, files themefiles.Files) themeusecase.Sources {
	return themeusecase.Sources{Presets: files, Own: files, Plugins: themefiles.NewPluginThemes(c.Infos)}
}

func openThemeFiles(c settingsParts) (themefiles.Files, error) {
	if folder, isSet := settingsadapter.LocateThemes(c.ThemesPath, c.RegistryPath); isSet {
		return themefiles.OpenAt(folder, c.Watches.GetShared())
	}
	return themefiles.Open(c.Watches.GetShared())
}

func getPluginPerformance(compute machinedomain.ResolvedCompute) plugindomain.Performance {
	return plugindomain.Performance{Profile: compute.Profile, Threads: compute.CoreBudget, Nice: compute.Nice}
}

func (c Config) openThemes(settingsDeps settingsParts, tasks *run.Tasks) *theme.Service {
	service, err := createThemeService(settingsDeps, c.Scales, editor.NewThemeWarning(tasks))
	if err != nil {
		settingsDeps.HandleError(err)
	}
	return service
}

func getPluginChangeSubscriber(c settingsParts) themeusecase.PluginChangeSubscriber {
	if c.PluginChanges == nil {
		return nil
	}
	return c.PluginChanges.Subscribe
}
