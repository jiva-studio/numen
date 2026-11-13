package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/absent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
)

func newAbsentHost(reason plugindomain.UnavailableReason, cause error) *PluginHost {
	runtime, settings := absent.Runtime{Reason: reason, Cause: cause}, absent.Settings{}
	return &PluginHost{
		Runtime: PluginRuntime{
			Activator: runtime, PluginRetrier: runtime, PluginDiscoverer: runtime, PluginLister: runtime,
			PluginResolver: runtime, PluginEventEmitter: runtime, PluginBinding: runtime, TextPlugins: runtime,
			PluginFailures: runtime, ConfigWatcher: runtime, RunningPlugins: runtime, ClientLookup: transport.NewClientLookup(nil),
		},
		Configuration: PluginConfiguration{
			PluginSettingDeclarer: settings, ConfigRefresher: settings, SettingsPush: settings,
			SettingChecker: settings, PersistHook: settings,
		},
		Watch:               PluginWatch{PluginShutdown: settings, DependencyReporter: settings, ContentTypeTable: contenttype.Open()},
		Endpoint:            hostendpoint.NewServices(),
		ApplySetting:        func(context.Context, string, string, any) error { return nil },
		Views:               PluginViews{Follow: func(settingsport.ChangeSubscriber) settingsport.ChangeSubscriber { return nil }},
		SetStorage:          func(string, string) {},
		RefreshDependencies: func() {},
		Health:              pluginhealth.NewItems(),
		Processes:           absent.Processes{},
		Release:             absent.Processes{},
	}
}
