package wire

import (
	"maps"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/logs"
)

func ReadLogEntry(in *v1.WriteLogRequest) logs.Entry {
	return logs.Entry{
		Level:      ReadLogLevel(in.GetLevel()),
		Source:     in.GetSource(),
		Message:    in.GetMessage(),
		Properties: maps.Clone(in.GetPropertiesJson()),
	}
}

func DescribeLogEntries(entries []logs.Entry) []*v1.LogEntry {
	out := make([]*v1.LogEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, &v1.LogEntry{
			TimeUnixMs:     entry.Time.UnixMilli(),
			Level:          DescribeLogLevel(entry.Level),
			Source:         entry.Source,
			Message:        entry.Message,
			PropertiesJson: maps.Clone(entry.Properties),
		})
	}
	return out
}

func ReadLogLevel(in v1.LogLevel) logs.Level {
	switch in {
	case v1.LogLevel_LOG_LEVEL_DEBUG:
		return logs.LevelDebug
	case v1.LogLevel_LOG_LEVEL_INFO:
		return logs.LevelInfo
	case v1.LogLevel_LOG_LEVEL_WARNING:
		return logs.LevelWarning
	case v1.LogLevel_LOG_LEVEL_ERROR:
		return logs.LevelError
	case v1.LogLevel_LOG_LEVEL_UNSPECIFIED:
		return logs.DefaultLevel
	}
	return logs.DefaultLevel
}

func DescribeLogLevel(in logs.Level) v1.LogLevel {
	switch in {
	case logs.LevelDebug:
		return v1.LogLevel_LOG_LEVEL_DEBUG
	case logs.LevelInfo:
		return v1.LogLevel_LOG_LEVEL_INFO
	case logs.LevelWarning:
		return v1.LogLevel_LOG_LEVEL_WARNING
	case logs.LevelError:
		return v1.LogLevel_LOG_LEVEL_ERROR
	}
	return v1.LogLevel_LOG_LEVEL_INFO
}
