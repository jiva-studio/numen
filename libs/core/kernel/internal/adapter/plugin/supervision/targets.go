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
		if sup.BeginDeactivation(defaultLimit, now) {
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

// takeReleasedTarget marks the plugin released, signals the watchers, and returns it as a target where it
// already holds no lease.
func takeReleasedTarget(s *Supervisors, pluginID string, now time.Time) []Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	sup, ok := s.byPlugin[pluginID]
	if !ok || !sup.MarkReleased() {
		return nil
	}
	notifyLocked(s)
	pid := sup.GetProcessID()
	if !sup.BeginDeactivation(IdleThreshold, now) {
		return nil
	}
	return []Target{takeTargetLocked(s, pluginID, sup, pid)}
}
