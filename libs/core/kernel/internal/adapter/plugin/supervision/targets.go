package supervision

import (
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
)

// Target is a plugin whose process is to be stopped, with what stopping it needs.
type Target struct {
	ID       string
	PID      int
	Sup      *lifecycle.Supervisor
	Endpoint transport.Endpoint
}

func takeLocked(s *Supervisors, id string, sup *lifecycle.Supervisor, pid int) Target {
	target := Target{ID: id, PID: pid, Sup: sup, Endpoint: s.endpoints[id]}
	forgetEndpointLocked(s, id)
	return target
}

func takeTarget(s *Supervisors, pluginID string) (Target, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sup, ok := s.byPlugin[pluginID]
	if !ok {
		return Target{}, false
	}
	return takeLocked(s, pluginID, sup, sup.GetProcessID()), true
}

func takeIdle(s *Supervisors, threshold time.Duration, now time.Time) []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	var targets []Target
	for id, sup := range s.byPlugin {
		pid := sup.GetProcessID()
		if sup.BeginDeactivation(threshold, now) {
			targets = append(targets, takeLocked(s, id, sup, pid))
		}
	}
	return targets
}

func takeRunning(s *Supervisors) []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	var targets []Target
	for id, sup := range s.byPlugin {
		if sup.GetState() != lifecycle.StateDormant {
			targets = append(targets, takeLocked(s, id, sup, sup.GetProcessID()))
		}
	}
	return targets
}
