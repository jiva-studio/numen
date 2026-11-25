package container

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginmanifest"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/plugin/hostendpoint"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/absent"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginhealth"
	settingsusecase "github.com/jiva-studio/numen/libs/core/kernel/usecase/settings"
)

func newAbsentExecution(reason plugindomain.UnavailableReason, cause error) *PluginExecution {
	runtime := absent.Runtime{Reason: reason, Cause: cause}
	settings := absent.Settings{}
	return &PluginExecution{
		Runtime: pluginRuntime{
			Activator: runtime, pluginRetrier: runtime, Listing: absent.PluginLister{}, Resolver: runtime,
			Manifests: pluginmanifest.Open(runtime, runtime, runtime), pluginEventEmitter: runtime, pluginBinding: runtime,
			PluginFailures: runtime, ConfigWatcher: runtime, ClientLookup: transport.NewClientLookup(nil),
		},
		Watch:     PluginWatch{pluginLifecycle: settings, dependencyReporter: settings, ContentTypeTable: contenttype.Open()},
		Health:    pluginhealth.NewItems(),
		Processes: absent.Processes{},
	}
}

func newAbsentPreferences() *PluginPreferences {
	settings := absent.Settings{}
	return &PluginPreferences{
		Configuration: pluginConfiguration{
			pluginSettingDeclarer: settings, configRefresher: settings, PluginConfig: settings,
			PluginSettings: settings,
		},
		ApplySetting:        func(context.Context, string, string, any) error { return nil },
		Views:               pluginViews{Follow: func(settingsusecase.ChangeSubscriber) settingsusecase.ChangeSubscriber { return nil }},
		SetStorage:          func(string, string) {},
		RefreshDependencies: func() {},
	}
}

func newAbsentEndpoints() *PluginEndpoints {
	lender := hostendpoint.NewLender()
	return &PluginEndpoints{Endpoint: hostendpoint.NewServices(lender.Get), Lender: lender}
}
