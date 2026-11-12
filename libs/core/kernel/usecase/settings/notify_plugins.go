package settings

import (
	"context"
	"fmt"
	"slices"
	"strings"

	settingsdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/settings"
	"github.com/jiva-studio/numen/libs/core/kernel/port/settingsport"
)

// NotifyPlugins gives the plugins the settings that changed.
type NotifyPlugins struct {
	codec          settingsport.ValueCodec
	apply          settingsport.PluginSettingApplier
	config         settingsport.PluginConfig
	resolveProfile settingsport.ProfileResolver
	signal         func()
	report         func(error)
}

// NotifyPluginsDeps is what NotifyPlugins is made of. Signal is called once the plugins have received the
// batch. It is optional.
type NotifyPluginsDeps struct {
	Codec          settingsport.ValueCodec
	Apply          settingsport.PluginSettingApplier
	Config         settingsport.PluginConfig
	ResolveProfile settingsport.ProfileResolver
	Signal         func()
	Report         func(error)
}

// NewNotifyPlugins makes the use case.
func NewNotifyPlugins(deps NotifyPluginsDeps) *NotifyPlugins {
	return &NotifyPlugins{
		codec: deps.Codec, apply: deps.Apply, config: deps.Config,
		resolveProfile: deps.ResolveProfile, signal: deps.Signal, report: deps.Report,
	}
}

func getPerformancePath() []string {
	return []string{settingsdomain.AppOwner, "performance", "profile"}
}

// Execute gives the plugins a batch of changed settings: their own settings and the
// performance profile. A failure is reported and the rest of the batch is still given.
func (u *NotifyPlugins) Execute(batch []settingsdomain.Setting) {
	u.updatePluginSettings(batch)
	if u.signal != nil {
		defer u.signal()
	}
	for _, one := range batch {
		switch {
		case slices.Equal(one.Path, getPerformancePath()):
			value, err := u.codec.Decode(one.JSON)
			profile, isText := value.(string)
			if err != nil || !isText {
				u.report(fmt.Errorf("the performance profile is not a name: %s", one.JSON))
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
		value, err := u.codec.Decode(one.JSON)
		if err != nil {
			continue
		}
		if err := u.apply(context.Background(), one.Path[0], one.Path[1], value); err != nil {
			u.report(fmt.Errorf("update plugin setting %s: %w", strings.Join(one.Path, "."), err))
		}
	}
}
