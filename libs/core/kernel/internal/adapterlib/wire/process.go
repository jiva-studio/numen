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
			StartedBy:          describeHolder(status.StartedBy),
			Holders:            describeHolders(status.Holders),
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

func describeHolder(holder plugin.Holder) *v1.PluginLeaseHolder {
	return &v1.PluginLeaseHolder{Who: holder.Who, Why: holder.Why}
}

func describeHolders(holders []plugin.Holder) []*v1.PluginLeaseHolder {
	out := make([]*v1.PluginLeaseHolder, 0, len(holders))
	for _, holder := range holders {
		out = append(out, describeHolder(holder))
	}
	return out
}
