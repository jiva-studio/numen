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

func takeTargetLocked(s *Supervisors, id string, sup *lifecycle.Supervisor, pid int) Target {
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
	return takeTargetLocked(s, pluginID, sup, sup.GetProcessID()), true
}

func takeIdleTargets(s *Supervisors, defaultLimit time.Duration, now time.Time) []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	var targets []Target
	for id, sup := range s.byPlugin {
		pid := sup.GetProcessID()
		if sup.BeginDeactivation(getIdleLimitLocked(s, id, defaultLimit), now) {
			targets = append(targets, takeTargetLocked(s, id, sup, pid))
		}
	}
	return targets
}

func takeRunningTargets(s *Supervisors) []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	var targets []Target
	for id, sup := range s.byPlugin {
		if sup.GetState() != lifecycle.StateDormant {
			targets = append(targets, takeTargetLocked(s, id, sup, sup.GetProcessID()))
		}
	}
	return targets
}

func takeReleasedTarget(s *Supervisors, pluginID string, now time.Time) []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	sup, ok := s.byPlugin[pluginID]
	if !ok {
		return nil
	}
	if _, isRunning := sup.GetUsage(); !isRunning {
		return nil
	}
	s.released[pluginID] = true
	notifyLocked(s)
	pid := sup.GetProcessID()
	if !sup.BeginDeactivation(0, now) {
		return nil
	}
	return []Target{takeTargetLocked(s, pluginID, sup, pid)}
}

func getIdleLimitLocked(s *Supervisors, pluginID string, defaultLimit time.Duration) time.Duration {
	switch {
	case s.released[pluginID]:
		return 0
	case s.idleLimits[pluginID] > 0:
		return s.idleLimits[pluginID]
	}
	return defaultLimit
}
