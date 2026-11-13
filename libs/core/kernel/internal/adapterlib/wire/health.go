package wire

import (
	"maps"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// ReadHealthItem returns the item a request carries. The plugin is left to the caller, which knows who
// reports it.
func ReadHealthItem(in *v1.HealthItem) plugin.HealthItem {
	item := plugin.HealthItem{
		Key:      in.GetKey(),
		Severity: readHealthSeverity(in.GetSeverity()),
		Text:     in.GetText(),
		Detail:   in.GetDetail(),
	}
	if action := in.GetAction(); action != nil {
		item.Action = &plugin.HealthAction{Command: action.GetCommand(), Values: maps.Clone(action.GetValues()), Title: action.GetTitle()}
	}
	return item
}

// DescribeHealthItems returns the items as the schema sends them.
func DescribeHealthItems(items []plugin.HealthItem) []*v1.HealthItem {
	out := make([]*v1.HealthItem, 0, len(items))
	for _, item := range items {
		described := &v1.HealthItem{
			PluginId: item.PluginID,
			Key:      item.Key,
			Severity: describeHealthSeverity(item.Severity),
			Text:     item.Text,
			Detail:   item.Detail,
		}
		if item.Action != nil {
			described.Action = &v1.HealthAction{Command: item.Action.Command, Values: maps.Clone(item.Action.Values), Title: item.Action.Title}
		}
		out = append(out, described)
	}
	return out
}

func readHealthSeverity(in v1.HealthSeverity) plugin.HealthSeverity {
	switch in {
	case v1.HealthSeverity_HEALTH_SEVERITY_WARNING:
		return plugin.HealthWarning
	case v1.HealthSeverity_HEALTH_SEVERITY_ERROR:
		return plugin.HealthError
	case v1.HealthSeverity_HEALTH_SEVERITY_OK:
		return plugin.HealthOK
	case v1.HealthSeverity_HEALTH_SEVERITY_UNSPECIFIED:
		return plugin.HealthUnspecified
	}
	return plugin.HealthUnspecified
}

func describeHealthSeverity(in plugin.HealthSeverity) v1.HealthSeverity {
	switch in {
	case plugin.HealthWarning:
		return v1.HealthSeverity_HEALTH_SEVERITY_WARNING
	case plugin.HealthError:
		return v1.HealthSeverity_HEALTH_SEVERITY_ERROR
	case plugin.HealthOK:
		return v1.HealthSeverity_HEALTH_SEVERITY_OK
	case plugin.HealthUnspecified:
		return v1.HealthSeverity_HEALTH_SEVERITY_UNSPECIFIED
	}
	return v1.HealthSeverity_HEALTH_SEVERITY_UNSPECIFIED
}
