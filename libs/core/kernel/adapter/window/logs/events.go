package logs

import (
	"strconv"

	domainlogs "github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

func RecordError(record func(domainlogs.Entry), message string, err error) {
	if record == nil || err == nil {
		return
	}
	record(domainlogs.Entry{Level: domainlogs.LevelError, Source: domainlogs.SourceCore, Message: message, Properties: map[string]string{"error": strconv.Quote(err.Error())}})
}

func ReportError(record func(domainlogs.Entry), report func(error), message string) func(error) {
	return func(err error) {
		RecordError(record, message, err)
		if report != nil {
			report(err)
		}
	}
}
