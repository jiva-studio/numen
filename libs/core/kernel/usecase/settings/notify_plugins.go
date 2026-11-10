package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// NotifyPlugins gives the plugins the settings that changed.
type NotifyPlugins struct {
	plugins        settingsport.PluginSettings
	config         settingsport.PluginConfig
	refreshSecrets settingsport.SecretRefresh
	resolveProfile settingsport.ProfileResolver
	signal         func()
	report         func(error)
}

// NewNotifyPlugins makes the use case. signal is called once the plugins have been told, and is
// nil where nobody waits for it.
func NewNotifyPlugins(
	plugins settingsport.PluginSettings, config settingsport.PluginConfig, refreshSecrets settingsport.SecretRefresh,
	resolveProfile settingsport.ProfileResolver, signal func(), report func(error),
) *NotifyPlugins {
	return &NotifyPlugins{
		plugins: plugins, config: config, refreshSecrets: refreshSecrets,
		resolveProfile: resolveProfile, signal: signal, report: report,
	}
}

var (
	secretsPath     = []string{settingsdomain.SecretsKey}
	performancePath = []string{settingsdomain.AppOwner, "performance", "profile"}
)

// Execute gives the plugins a batch of changed settings: their own settings, the secrets and the
// performance profile. A failure is reported and the rest of the batch is still given.
func (u *NotifyPlugins) Execute(batch []settingsdomain.Setting) {
	u.updatePluginSettings(batch)
	if u.signal != nil {
		defer u.signal()
	}
	for _, one := range batch {
		switch {
		case slices.Equal(one.Path, secretsPath):
			if err := u.refreshSecrets(context.Background()); err != nil {
				u.report(fmt.Errorf("configure the plugins after the secrets changed: %w", err))
			}
		case slices.Equal(one.Path, performancePath):
			var profile string
			if err := json.Unmarshal([]byte(one.JSON), &profile); err != nil {
				u.report(err)
				continue
			}
			u.config.SetPerformanceProfile(context.Background(), u.resolveProfile(profile))
		}
	}
}

func (u *NotifyPlugins) updatePluginSettings(entries []settingsdomain.Setting) {
	// The enabled entries go last: enabling a plugin generates the values of its settings that hold
	// none, and a value in the same batch is in memory by then.
	isEnabledEntry := func(one settingsdomain.Setting) bool {
		return len(one.Path) == 2 && one.Path[1] == settingsdomain.HostFieldEnabled
	}
	ordered := slices.Concat(
		slices.DeleteFunc(slices.Clone(entries), isEnabledEntry),
		slices.DeleteFunc(slices.Clone(entries), func(one settingsdomain.Setting) bool { return !isEnabledEntry(one) }),
	)
	for _, one := range ordered {
		if !settingsdomain.IsPluginSetting(one.Path) {
			continue
		}
		var value any
		if err := json.Unmarshal([]byte(one.JSON), &value); err != nil {
			continue
		}
		if err := u.plugins.UpdatePluginSetting(context.Background(), one.Path[0], one.Path[1], value); err != nil {
			u.report(fmt.Errorf("update plugin setting %s: %w", strings.Join(one.Path, "."), err))
		}
	}
}
