package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/bridge"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

func newAbsentHost(reason plugindomain.UnavailableReason, cause error) *PluginHost {
	runtime, settings := bridge.AbsentRuntime{Reason: reason, Cause: cause}, bridge.AbsentSettings{}
	return &PluginHost{
		Runtime: PluginRuntime{
			Activator: runtime, PluginRetrier: runtime, PluginDiscoverer: runtime, PluginLister: runtime,
			PluginResolver: runtime, PluginEventEmitter: runtime, PluginBinding: runtime, TextPlugins: runtime,
			PluginFailures: runtime, ConfigWatcher: runtime, ClientLookup: transport.NewClientLookup(nil),
		},
		Configuration: PluginConfiguration{
			PluginSettingDeclarer: settings, SecretConfigurator: settings, SettingsPush: settings,
			SettingChecker: settings, PersistHook: settings,
		},
		Watch:               PluginWatch{PluginShutdown: settings, DependencyReporter: settings, ContentTypeTable: contenttype.Open()},
		Endpoint:            hostendpoint.NewServices(),
		ApplySetting:        func(context.Context, string, string, any) error { return nil },
		Views:               PluginViews{Follow: func(settingsport.ChangeSubscriber) settingsport.ChangeSubscriber { return nil }},
		SetStorage:          func(string, string) {},
		RefreshDependencies: func() {},
	}
}
