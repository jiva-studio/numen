package pluginhealth

import "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"

// ReportSearchSource lists a failed read of the description of the search source as the search item of
// the plugin behind the source.
type ReportSearchSource struct {
	items *Items
}

// NewReportSearchSource returns the use case over the list.
func NewReportSearchSource(items *Items) *ReportSearchSource {
	return &ReportSearchSource{items: items}
}

// Execute sets the search item of the plugin, with the error as its text and the settings of the plugin
// as its action, and removes it for a read that succeeded.
func (u *ReportSearchSource) Execute(pluginID string, readErr error) {
	if readErr == nil {
		u.items.Remove(pluginID, plugin.HealthKeySearch)
		return
	}
	text, detail := plugin.FitHealthText(readErr.Error())
	u.items.Set(plugin.HealthItem{
		PluginID: pluginID,
		Key:      plugin.HealthKeySearch,
		Severity: plugin.HealthError,
		Text:     text,
		Detail:   detail,
		Action: &plugin.HealthAction{
			Command: plugin.AppCommandOpenSettings,
			Values:  map[string]string{plugin.OpenSettingsPlugin: pluginID},
		},
	})
}
