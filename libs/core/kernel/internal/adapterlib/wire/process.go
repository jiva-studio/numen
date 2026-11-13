package wire

import (
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// DescribePluginProcesses returns the plugins that run as the schema sends them. A zero time is sent
// as zero.
func DescribePluginProcesses(statuses []plugin.ProcessStatus) []*v1.PluginProcess {
	out := make([]*v1.PluginProcess, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, &v1.PluginProcess{
			PluginId:           status.PluginID,
			PluginName:         status.PluginName,
			StartTimeUnixMs:    describeUnixMilli(status.StartTime),
			LastUseTimeUnixMs:  describeUnixMilli(status.LastUseTime),
			LeaseCount:         int32(status.LeaseCount),
			IsInProcess:        status.IsInProcess,
			IdleStopTimeUnixMs: describeUnixMilli(status.IdleStopTime),
			Health:             describeHealthSeverity(status.Health),
			HealthText:         status.HealthText,
		})
	}
	return out
}

func describeUnixMilli(instant time.Time) int64 {
	if instant.IsZero() {
		return 0
	}
	return instant.UnixMilli()
}
