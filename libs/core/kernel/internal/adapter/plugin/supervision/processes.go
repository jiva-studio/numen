package supervision

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p manifest.Discovered, port int, token string) (*os.Process, io.ReadCloser, error)

// BuiltIn is what serves the backend of a plugin linked into the binary, from the variables a launched
// process of it is given.
type BuiltIn struct {
	Serve func(vars map[string]string) (http.Handler, error)
}

// Launch is what a start of a plugin needs to know of it, and the first lease on it.
type Launch struct {
	Plugin       manifest.Discovered
	BuiltIn      *BuiltIn
	Requirements []manifest.DependencyRequirement
	VaultPath    string
	Lease        lifecycle.Lease
}

type stdinKey struct {
	pluginID string
	pid      int
}

// processTable is what a start of a process reads and writes of the supervisors.
type processTable interface {
	Clients() *transport.Clients
	AttachProcess(ctx context.Context, attach ProcessAttachment) error
	AttachInProcess(pluginID string, sup *lifecycle.Supervisor, endpoint transport.Endpoint, lease lifecycle.Lease, now time.Time) error
}

// launchServices is what a start of a process reads of the services the host gives the plugins.
type launchServices interface {
	GetDependencies() pluginport.DependencyRegistry
	OpenHostEndpoint(ctx context.Context) (int, error)
}

// Processes starts the process of a plugin, or the handler of a built-in one, and holds the pipes it
// opened for the process until the process exits.
type Processes struct {
	sups     processTable
	clock    machineport.Clock
	report   func(error)
	services launchServices
	storage  *storage.Storage

	mu       sync.RWMutex
	launcher Launcher
	limits   process.Limits
	policy   lifecycle.ActivationPolicy

	stdinsMu sync.Mutex
	stdins   map[stdinKey]io.Closer
}

// NewProcesses returns the starter of the processes the supervisors track.
func NewProcesses(sups processTable, clock machineport.Clock, report func(error), services launchServices, files *storage.Storage) *Processes {
	return &Processes{
		sups: sups, clock: clock, report: report, services: services, storage: files,
		limits: process.DefaultLimits(), policy: lifecycle.DefaultActivationPolicy(),
		stdins: make(map[stdinKey]io.Closer),
	}
}

// SetProcessLimits sets execution limits for launched plugin processes.
func (p *Processes) SetProcessLimits(limits process.Limits) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limits = limits
}

// GetProcessLimits returns the execution limits for launched plugin processes.
func (p *Processes) GetProcessLimits() process.Limits {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.limits
}

// SetLauncher overrides the process launcher for testing.
func (p *Processes) SetLauncher(l Launcher) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.launcher = l
}

func (p *Processes) getLauncher() Launcher {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.launcher
}

// Start starts the plugin, in a process of its own or, for a built-in plugin, in this one, and
// returns its endpoint.
func (p *Processes) Start(ctx context.Context, launch Launch, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	if p.getLauncher() != nil || launch.BuiltIn == nil {
		return p.startPluginProcess(ctx, launch, sup)
	}
	return p.startBuiltIn(ctx, launch, sup)
}

func (p *Processes) cleanupAbortedProcess(proc *os.Process, stdout io.Closer, pluginID string) {
	killProcess(p.report, proc, pluginID)
	closeResource(p.report, "stdout of plugin "+pluginID, stdout)
	p.CloseStdin(pluginID, proc.Pid)
}

func killProcess(report func(error), proc *os.Process, pluginID string) {
	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		report(fmt.Errorf("kill plugin %s: %w", pluginID, err))
	}
	if _, err := proc.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		report(fmt.Errorf("wait for plugin %s: %w", pluginID, err))
	}
}

func closeResource(report func(error), name string, c io.Closer) {
	if err := c.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		report(fmt.Errorf("close %s: %w", name, err))
	}
}

// HoldStdin keeps the pipe to the stdin of the process until the process exits, which keeps the
// process from reading end of input.
func (p *Processes) HoldStdin(pluginID string, pid int, stdin io.Closer) {
	p.stdinsMu.Lock()
	defer p.stdinsMu.Unlock()
	if p.stdins == nil {
		p.stdins = make(map[stdinKey]io.Closer)
	}
	p.stdins[stdinKey{pluginID: pluginID, pid: pid}] = stdin
}

// CountStdins returns the number of pipes held.
func (p *Processes) CountStdins() int {
	p.stdinsMu.Lock()
	defer p.stdinsMu.Unlock()
	return len(p.stdins)
}

// CloseStdin closes the pipe held for the process and forgets it.
func (p *Processes) CloseStdin(pluginID string, pid int) {
	key := stdinKey{pluginID: pluginID, pid: pid}
	p.stdinsMu.Lock()
	c, ok := p.stdins[key]
	delete(p.stdins, key)
	p.stdinsMu.Unlock()
	if ok {
		closeResource(p.report, "stdin of plugin "+pluginID, c)
	}
}

// CloseAllStdins closes every pipe held and returns the failures.
func (p *Processes) CloseAllStdins() error {
	p.stdinsMu.Lock()
	stdins := p.stdins
	p.stdins = nil
	p.stdinsMu.Unlock()
	var failed []error
	for key, c := range stdins {
		if err := c.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			failed = append(failed, fmt.Errorf("close stdin of plugin %s: %w", key.pluginID, err))
		}
	}
	return errors.Join(failed...)
}

// SetActivationPolicy sets the deadline of the start of a plugin.
func (p *Processes) SetActivationPolicy(policy lifecycle.ActivationPolicy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.policy = policy
}

func (p *Processes) getActivationTimeout() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.policy.Timeout
}

func (p *Processes) waitForReady(ctx context.Context, stdout io.Reader) (transport.Handshake, error) {
	if timeout := p.getActivationTimeout(); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return transport.WaitForHandshake(ctx, stdout)
}

func (p *Processes) startPluginProcess(ctx context.Context, launch Launch, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	plug := launch.Plugin
	token, err := transport.GenerateAuthToken()
	if err != nil {
		return NoEndpoint(), fmt.Errorf("generate token: %w", err)
	}
	if token == "" {
		return NoEndpoint(), errors.New("empty auth token")
	}
	launcher := p.getLauncher()
	if launcher == nil {
		launcher = func(ctx context.Context, _ manifest.Discovered, port int, token string) (*os.Process, io.ReadCloser, error) {
			return p.launchDefault(ctx, launch, port, token)
		}
	}

	proc, stdout, err := launcher(ctx, plug, 0, token)
	if err != nil {
		return NoEndpoint(), fmt.Errorf("launch plugin %s: %w", plug.Manifest.ID, err)
	}

	handshake, err := p.waitForReady(ctx, stdout)
	if err == nil {
		err = checkReportedListeners(plug.Manifest, handshake.Listeners)
	}
	if err != nil {
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return NoEndpoint(), fmt.Errorf("handshake plugin %s: %w", plug.Manifest.ID, err)
	}

	go func() {
		if _, err := io.Copy(io.Discard, stdout); err != nil && !errors.Is(err, os.ErrClosed) {
			p.report(fmt.Errorf("read output of plugin %s: %w", plug.Manifest.ID, err))
		}
		closeResource(p.report, "stdout of plugin "+plug.Manifest.ID, stdout)
	}()

	endpoint := transport.Endpoint{Host: "127.0.0.1", Port: handshake.Port, Token: token}
	p.sups.Clients().Add(token, transport.NewNetworkTransport())
	if err := transport.CheckPluginProtocol(ctx, p.sups.Clients().Get(endpoint), plug.Manifest, endpoint); err != nil {
		p.sups.Clients().Remove(token)
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return NoEndpoint(), err
	}

	err = p.sups.AttachProcess(ctx, ProcessAttachment{
		PluginID: plug.Manifest.ID, Supervisor: sup, Process: proc, Endpoint: endpoint,
		Listeners: handshake.Listeners, Now: p.clock.Now(),
		OnExit: func() { p.CloseStdin(plug.Manifest.ID, proc.Pid) },
		Lease:  launch.Lease,
	})
	if err != nil {
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return NoEndpoint(), err
	}
	return endpoint, nil
}

func resolvePluginBinary(plug manifest.Discovered) (string, error) {
	if plug.Manifest.Backend.Entrypoint == "" {
		return "", security.ErrEmptyEntrypoint
	}

	entrypoint := plug.Manifest.Backend.Entrypoint
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(entrypoint), ".exe") {
		entrypoint += ".exe"
	}

	binPath, err := security.ValidateExecutablePath(plug.Dir, entrypoint)
	if err != nil {
		return "", fmt.Errorf("validate plugin binary: %w", err)
	}
	return binPath, nil
}

func (p *Processes) launchDefault(ctx context.Context, launch Launch, port int, token string) (*os.Process, io.ReadCloser, error) {
	plug := launch.Plugin
	binPath, err := resolvePluginBinary(plug)
	if err != nil {
		return nil, nil, err
	}

	workDir, err := process.ValidateWorkingDirectory(plug.Dir, plug.Dir, launch.VaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("validate plugin working directory: %w", err)
	}

	cmd := process.NewCommand(context.WithoutCancel(ctx), binPath, port)
	cmd.Dir = workDir

	pluginVars, err := p.GetLaunchVars(ctx, plug.Manifest.ID, launch.Requirements, launch.VaultPath, port, token)
	if err != nil {
		return nil, nil, err
	}
	cmd.Env = process.SanitizeEnvironment(os.Environ(), pluginVars, plug.Manifest.GetRuntimeIDs())

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("pipe stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, errors.Join(fmt.Errorf("pipe stdout: %w", err), stdin.Close())
	}

	if err := process.HideInheritedFiles(); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("hide inherited files: %w", err), stdin.Close(), stdout.Close())
	}
	// A failed Start has closed both pipes already.
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("start %s: %w", binPath, err)
	}

	if _, err := io.WriteString(stdin, token+"\n"); err != nil {
		killProcess(p.report, cmd.Process, plug.Manifest.ID)
		return nil, nil, errors.Join(fmt.Errorf("write token: %w", err), stdin.Close(), stdout.Close())
	}

	if err := process.ApplyLimits(cmd.Process.Pid, p.GetProcessLimits()); err != nil {
		killProcess(p.report, cmd.Process, plug.Manifest.ID)
		return nil, nil, errors.Join(fmt.Errorf("apply process limits: %w", err), stdin.Close(), stdout.Close())
	}

	p.HoldStdin(plug.Manifest.ID, cmd.Process.Pid, stdin)
	return cmd.Process, stdout, nil
}

func checkReportedListeners(m manifest.Manifest, reported map[string]int) error {
	for _, declared := range m.Contributes.Listeners {
		if _, ok := reported[declared.Name]; !ok {
			return fmt.Errorf("listener %s: %w", declared.Name, ErrListenerNotReported)
		}
	}
	return nil
}

// ErrListenerNotReported marks a ready line that does not report a listener the plugin declares.
var ErrListenerNotReported = errors.New("the ready line does not report the listener")
