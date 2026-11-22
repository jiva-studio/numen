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

// NewRecordFunc returns the function that records an entry and hands the failure of recording to report, where there is one.
func NewRecordFunc(record Recorder, report func(error)) func(domainlogs.Entry) {
	return func(entry domainlogs.Entry) {
		if err := record.Execute(entry); err != nil && report != nil {
			report(err)
		}
	}
}
