package supervision

import (
	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"strconv"
)

func RecordDependencyStatus(record func(domainlogs.Entry), id string, status plugin.DependencyStatus) {
	if record == nil {
		return
	}
	entry := domainlogs.Entry{Source: domainlogs.SourceCore, Properties: map[string]string{"dependency": strconv.Quote(id)}}
	switch status.State {
	case plugin.DependencyStateReady:
		entry.Level, entry.Message = domainlogs.LevelInfo, "Plugin dependency ready"
	case plugin.DependencyStateFailed:
		entry.Level, entry.Message = domainlogs.LevelError, "Plugin dependency failed"
		entry.Properties["error"] = strconv.Quote(status.Error)
	default:
		return
	}
	record(entry)
}

func ReportPluginFailure(record func(domainlogs.Entry)) func(plugin.Failure) {
	return func(failure plugin.Failure) {
		if record == nil {
			return
		}
		record(domainlogs.Entry{
			Level: domainlogs.LevelError, Source: domainlogs.SourceCore, Message: "Plugin failed",
			Properties: map[string]string{"plugin": strconv.Quote(failure.PluginID), "reason": strconv.Quote(failure.Reason)},
		})
	}
}
