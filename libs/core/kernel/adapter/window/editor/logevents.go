package editor

import (
	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"strconv"
)

func ReportVault(record func(domainlogs.Entry)) func(domain.Vault, string, error) {
	return func(v domain.Vault, event string, err error) {
		if record == nil {
			return
		}
		level := domainlogs.LevelInfo
		if err != nil || event == "Vault watch stopped" {
			level = domainlogs.LevelWarning
		}
		properties := map[string]string{"vault": strconv.Quote(string(v.ID))}
		if err != nil {
			properties["error"] = strconv.Quote(err.Error())
		}
		record(domainlogs.Entry{Level: level, Source: domainlogs.SourceCore, Message: event, Properties: properties})
	}
}
