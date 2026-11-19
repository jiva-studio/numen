// Package logevents describes diagnostic events recorded by the application.
package logevents

import (
	"strconv"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func RecordError(record func(logs.Entry), message string, err error) {
	if record == nil || err == nil {
		return
	}
	record(logs.Entry{Level: logs.LevelError, Source: logs.SourceCore, Message: message, Properties: map[string]string{"error": strconv.Quote(err.Error())}})
}

func ReportError(record func(logs.Entry), report func(error), message string) func(error) {
	return func(err error) {
		RecordError(record, message, err)
		if report != nil {
			report(err)
		}
	}
}

func RecordDependencyStatus(record func(logs.Entry), id string, status plugin.DependencyStatus) {
	if record == nil {
		return
	}
	entry := logs.Entry{Source: logs.SourceCore, Properties: map[string]string{"dependency": strconv.Quote(id)}}
	switch status.State {
	case plugin.DependencyStateReady:
		entry.Level, entry.Message = logs.LevelInfo, "Plugin dependency ready"
	case plugin.DependencyStateFailed:
		entry.Level, entry.Message = logs.LevelError, "Plugin dependency failed"
		entry.Properties["error"] = strconv.Quote(status.Error)
	default:
		return
	}
	record(entry)
}

func ReportPluginFailure(record func(logs.Entry)) func(plugin.Failure) {
	return func(failure plugin.Failure) {
		if record == nil {
			return
		}
		record(logs.Entry{
			Level: logs.LevelError, Source: logs.SourceCore, Message: "Plugin failed",
			Properties: map[string]string{"plugin": strconv.Quote(failure.PluginID), "reason": strconv.Quote(failure.Reason)},
		})
	}
}

func ReportVault(record func(logs.Entry)) func(domain.Vault, string, error) {
	return func(v domain.Vault, event string, err error) {
		if record == nil {
			return
		}
		level := logs.LevelInfo
		if err != nil || event == "Vault watch stopped" {
			level = logs.LevelWarning
		}
		properties := map[string]string{"vault": strconv.Quote(string(v.ID))}
		if err != nil {
			properties["error"] = strconv.Quote(err.Error())
		}
		record(logs.Entry{Level: level, Source: logs.SourceCore, Message: event, Properties: properties})
	}
}
