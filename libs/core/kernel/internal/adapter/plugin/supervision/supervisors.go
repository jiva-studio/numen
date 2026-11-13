package supervision

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/localendpoint"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const rpcShutdownGrace = 2 * time.Second

// Supervisors keeps, for each plugin, the supervisor of its process, the idle limit its manifest sets,
// the process that runs and the client that calls it. One lock guards them, so that a plugin is never
// started twice and never stopped half way. The watchers of RunningProcesses are signaled under either
// side of the lock and changed only under the write side.
type Supervisors struct {
	mu           sync.RWMutex
	byPlugin     map[string]*lifecycle.Supervisor
	runs         map[string]*processRun
	clients      *transport.Clients
	report       func(error)
	onDrop       func(pluginID string)
	idleLimits   map[string]time.Duration
	stopReleased func(pluginID string)
	watchers     []chan struct{}
}

// processRun is the process of a plugin that runs: its endpoint, the ports it reported for its
// listeners, when it started and whether the plugin released it.
type processRun struct {
	endpoint      transport.Endpoint
	listenerPorts map[string]int
	startTime     time.Time
	isReleased    bool
}

// NewSupervisors returns the supervisors of a host that has no plugin. report receives the problems
// a supervisor meets, and onDrop is called with the id of a plugin whose process ended.
func NewSupervisors(report func(error), onDrop func(pluginID string)) *Supervisors {
	return &Supervisors{
		byPlugin:   make(map[string]*lifecycle.Supervisor),
		runs:       make(map[string]*processRun),
		clients:    transport.NewClients(),
		idleLimits: make(map[string]time.Duration),
		report:     report,
		onDrop:     onDrop,
	}
}

// Register returns the supervisor of the plugin, which it makes where there is none, and gives it the
// permissions and the idle limit the manifest declares.
func (s *Supervisors) Register(p manifest.Discovered) *lifecycle.Supervisor {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := p.Manifest.ID
	sup, exists := s.byPlugin[id]
	if !exists {
		sup = lifecycle.New(s.report)
		sup.SetOnExit(func() { s.Drop(id) })
		s.byPlugin[id] = sup
	}
	sup.Permissions.Set(p.Manifest.Permissions)
	s.idleLimits[id] = p.Manifest.Backend.GetIdleStopTime()
	return sup
}

// Put sets the supervisor of the plugin to one made elsewhere.
func (s *Supervisors) Put(pluginID string, sup *lifecycle.Supervisor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byPlugin[pluginID] = sup
}

func (s *Supervisors) Get(pluginID string) (*lifecycle.Supervisor, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sup, ok := s.byPlugin[pluginID]
	return sup, ok
}

// Forget removes the supervisor of the plugin.
func (s *Supervisors) Forget(pluginID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byPlugin, pluginID)
}

// GetEndpoint returns the endpoint of the running process of the plugin.
func (s *Supervisors) GetEndpoint(pluginID string) (transport.Endpoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[pluginID]
	if !ok {
		return transport.Endpoint{}, false
	}
	return run.endpoint, true
}

func getEndpointLocked(s *Supervisors, pluginID string) transport.Endpoint {
	if run, ok := s.runs[pluginID]; ok {
		return run.endpoint
	}
	return transport.Endpoint{}
}

func getStartTimeLocked(s *Supervisors, pluginID string) time.Time {
	if run, ok := s.runs[pluginID]; ok {
		return run.startTime
	}
	return time.Time{}
}

func isReleasedLocked(s *Supervisors, pluginID string) bool {
	run, ok := s.runs[pluginID]
	return ok && run.isReleased
}

// FindByToken returns the id of the running plugin the token was given to.
func (s *Supervisors) FindByToken(token string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var pluginID string
	var found bool
	for id, run := range s.runs {
		if localendpoint.IsTokenEqual(run.endpoint.Token, token) {
			pluginID, found = id, true
		}
	}
	return pluginID, found
}

// IsActive reports whether the process of the plugin is active or idle.
func (s *Supervisors) IsActive(pluginID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sup, ok := s.byPlugin[pluginID]
	if !ok {
		return false
	}
	state := sup.GetState()
	return state == lifecycle.StateActive || state == lifecycle.StateIdle
}

// ListRunning returns the ids of the plugins whose supervisor is not dormant.
func (s *Supervisors) ListRunning() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.byPlugin))
	for id, sup := range s.byPlugin {
		if sup.GetState() != lifecycle.StateDormant {
			ids = append(ids, id)
		}
	}
	return ids
}

// ListEndpointOwners returns the ids of the plugins that have an endpoint.
func (s *Supervisors) ListEndpointOwners() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.runs))
	for id := range s.runs {
		ids = append(ids, id)
	}
	return ids
}

// GetListenerPort returns the port the process of the plugin reported for a listener.
func (s *Supervisors) GetListenerPort(pluginID, name string) (int, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[pluginID]
	if !ok {
		return 0, false
	}
	port, ok := run.listenerPorts[name]
	return port, ok
}

// Release gives up a hold on the plugin if its endpoint is still the one the token names.
func (s *Supervisors) Release(pluginID, token string, now time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sup, ok := s.byPlugin[pluginID]
	if !ok || getEndpointLocked(s, pluginID).Token != token {
		return
	}
	returnHoldLocked(s, pluginID, sup, now)
}

// ReturnHold gives up a hold on the plugin. A released process that holds no other is stopped at once.
func (s *Supervisors) ReturnHold(pluginID string, sup *lifecycle.Supervisor, now time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	returnHoldLocked(s, pluginID, sup, now)
}

func returnHoldLocked(s *Supervisors, pluginID string, sup *lifecycle.Supervisor, now time.Time) {
	sup.Release(now)
	notifyLocked(s)
	if usage, isRunning := sup.GetUsage(); isRunning && usage.IsIdle && isReleasedLocked(s, pluginID) && s.stopReleased != nil {
		go s.stopReleased(pluginID)
	}
}

// Clients returns the clients that call the processes, by the token each process was given.
func (s *Supervisors) Clients() *transport.Clients {
	return s.clients
}

// Drop forgets the endpoint and the client of a plugin whose process ended.
func (s *Supervisors) Drop(pluginID string) {
	s.mu.Lock()
	s.clients.Remove(getEndpointLocked(s, pluginID).Token)
	forgetEndpointLocked(s, pluginID)
	s.mu.Unlock()
	s.onDrop(pluginID)
}

func forgetEndpointLocked(s *Supervisors, pluginID string) {
	delete(s.runs, pluginID)
}

// Claim is what Begin found.
type Claim struct {
	Endpoint transport.Endpoint
	Running  bool
	State    lifecycle.ProcessState
	Started  bool
}

// Begin takes a hold on the running plugin, or reports the state of the supervisor, or sets a dormant
// supervisor to activating.
func (s *Supervisors) Begin(pluginID string, sup *lifecycle.Supervisor, now time.Time) Claim {
	s.mu.Lock()
	defer s.mu.Unlock()
	if endpoint, ok := acquireRunningLocked(s, pluginID, sup, now); ok {
		return Claim{Endpoint: endpoint, Running: true, State: lifecycle.StateDormant, Started: false}
	}
	if state := sup.GetState(); state != lifecycle.StateDormant {
		return Claim{Endpoint: NoEndpoint(), Running: false, State: state, Started: false}
	}
	sup.SetActivating()
	return Claim{Endpoint: NoEndpoint(), Running: false, State: lifecycle.StateDormant, Started: true}
}

// AcquireRunning takes a hold on the plugin and returns its endpoint if a process of it runs. The hold
// cancels a release of the process.
func (s *Supervisors) AcquireRunning(pluginID string, sup *lifecycle.Supervisor, now time.Time) (transport.Endpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return acquireRunningLocked(s, pluginID, sup, now)
}

func acquireRunningLocked(s *Supervisors, pluginID string, sup *lifecycle.Supervisor, now time.Time) (transport.Endpoint, bool) {
	run, ok := s.runs[pluginID]
	if !ok || IsNoEndpoint(run.endpoint) || run.endpoint.Token == "" || !sup.Acquire(now) {
		return NoEndpoint(), false
	}
	run.isReleased = false
	notifyLocked(s)
	return run.endpoint, true
}

// ProcessAttachment is what AttachProcess records for a started plugin process.
type ProcessAttachment struct {
	PluginID   string
	Supervisor *lifecycle.Supervisor
	Process    *os.Process
	Endpoint   transport.Endpoint
	Listeners  map[string]int
	Now        time.Time
	OnExit     func()
}

// AttachProcess records a process the supervisor now tracks, with its endpoint and the ports of its
// listeners, and takes the first hold on it. It fails where the start was aborted meanwhile. onExit
// runs after the endpoint is forgotten when the process exits.
func (s *Supervisors) AttachProcess(ctx context.Context, attach ProcessAttachment) error {
	pluginID, sup, proc, endpoint := attach.PluginID, attach.Supervisor, attach.Process, attach.Endpoint
	now, listeners, onExit := attach.Now, attach.Listeners, attach.OnExit
	s.mu.Lock()
	defer s.mu.Unlock()
	if sup.GetState() != lifecycle.StateActivating {
		return lifecycle.ErrActivationAborted
	}
	if err := sup.Attach(ctx, proc, now); err != nil {
		return fmt.Errorf("attach plugin %s: %w", pluginID, err)
	}
	sup.SetOnExit(func() {
		s.Drop(pluginID)
		onExit()
	})
	sup.Acquire(now)
	s.runs[pluginID] = &processRun{endpoint: endpoint, listenerPorts: listeners, startTime: now, isReleased: false}
	sup.SetRPCShutdown(createRPCShutdown(endpoint, s.clients.Get(endpoint), rpcShutdownGrace))
	sup.FinishActivating()
	exit := sup.GetExitSignal()
	go func() {
		<-exit
		s.NotifyChange()
	}()
	notifyLocked(s)
	return nil
}

// AttachInProcess records the endpoint of a plugin served inside this process and activates it. It
// fails where the start was aborted meanwhile.
func (s *Supervisors) AttachInProcess(pluginID string, sup *lifecycle.Supervisor, endpoint transport.Endpoint, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sup.GetState() != lifecycle.StateActivating {
		return lifecycle.ErrActivationAborted
	}
	s.runs[pluginID] = &processRun{endpoint: endpoint, listenerPorts: nil, startTime: now, isReleased: false}
	sup.ActivateInProcess(now)
	notifyLocked(s)
	return nil
}

// SetEndpoint records the endpoint of a plugin as a process of it would have reported it.
func (s *Supervisors) SetEndpoint(pluginID string, endpoint transport.Endpoint) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run, ok := s.runs[pluginID]; ok {
		run.endpoint = endpoint
		return
	}
	s.runs[pluginID] = &processRun{endpoint: endpoint, listenerPorts: nil, startTime: time.Time{}, isReleased: false}
}

// NotifyChange tells the watchers of RunningProcesses that a plugin took or returned a lease, or
// stopped, outside the calls of the supervisors.
func (s *Supervisors) NotifyChange() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	notifyLocked(s)
}

func notifyLocked(s *Supervisors) {
	for _, watcher := range s.watchers {
		select {
		case watcher <- struct{}{}:
		default:
		}
	}
}
