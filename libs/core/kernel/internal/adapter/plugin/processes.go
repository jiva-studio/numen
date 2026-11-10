package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/storage"

	"connectrpc.com/connect"
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/process"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/security"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

// Launcher is a function that spawns a plugin process and provides its stdout stream.
type Launcher func(ctx context.Context, p manifest.Discovered, port int, token string) (*os.Process, io.ReadCloser, error)

type stdinKey struct {
	pluginID string
	pid      int
}

// Processes starts the process of a plugin, or the handler of a built-in one, and holds the pipes
// it opened for the process until the process exits. The host forgets the endpoint of a plugin whose
// process exited, through dropEndpoint.
type Processes struct {
	t            *table
	clock        machineport.Clock
	errs         *errorSink
	services     *Services
	storage      *storage.Storage
	dropEndpoint func(pluginID string)

	mu       sync.RWMutex
	launcher Launcher
	limits   process.Limits

	stdinsMu sync.Mutex
	stdins   map[stdinKey]io.Closer
}

func newProcesses(t *table, clock machineport.Clock, errs *errorSink, services *Services, storage *storage.Storage) *Processes {
	return &Processes{
		t: t, clock: clock, errs: errs, services: services, storage: storage,
		limits: process.DefaultLimits(),
		stdins: make(map[stdinKey]io.Closer),
	}
}

// SetProcessLimits sets execution limits for launched plugin processes.
func (p *Processes) SetProcessLimits(limits process.Limits) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.limits = limits
}

// SetActivationPolicy sets the activation deadline of plugins.
func (p *Processes) SetActivationPolicy(policy lifecycle.ActivationPolicy) {
	p.t.mu.Lock()
	defer p.t.mu.Unlock()
	p.t.activation = policy
}

// SetLauncher overrides the process launcher for testing.
func (p *Processes) SetLauncher(l Launcher) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.launcher = l
}

func (p *Processes) start(ctx context.Context, plug manifest.Discovered, sup *lifecycle.Supervisor) (plugindomain.Endpoint, error) {
	p.mu.RLock()
	launcher := p.launcher
	p.mu.RUnlock()
	p.t.mu.RLock()
	builtIn, isBuiltIn := p.t.getBuiltInLocked(plug.Manifest.ID)
	isBuiltIn = isBuiltIn && !p.t.onDisk[plug.Manifest.ID]
	p.t.mu.RUnlock()
	if launcher != nil || !isBuiltIn {
		return p.startPluginProcess(ctx, plug, sup)
	}
	return p.startBuiltIn(ctx, plug, builtIn, sup)
}

func (p *Processes) cleanupAbortedProcess(proc *os.Process, stdout io.Closer, pluginID string) {
	p.killProcess(proc, pluginID)
	p.closeResource("stdout of plugin "+pluginID, stdout)
	p.closeStdin(pluginID, proc.Pid)
}

func (p *Processes) killProcess(proc *os.Process, pluginID string) {
	if err := proc.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		p.errs.report(fmt.Errorf("kill plugin %s: %w", pluginID, err))
	}
	if _, err := proc.Wait(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		p.errs.report(fmt.Errorf("wait for plugin %s: %w", pluginID, err))
	}
}

func (p *Processes) closeResource(name string, c io.Closer) {
	if err := c.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		p.errs.report(fmt.Errorf("close %s: %w", name, err))
	}
}

func (p *Processes) closeStdin(pluginID string, pid int) {
	key := stdinKey{pluginID: pluginID, pid: pid}
	p.stdinsMu.Lock()
	c, ok := p.stdins[key]
	delete(p.stdins, key)
	p.stdinsMu.Unlock()
	if ok {
		p.closeResource("stdin of plugin "+pluginID, c)
	}
}

func (p *Processes) closeAllStdins() error {
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

func (p *Processes) waitForReady(ctx context.Context, stdout io.Reader) (transport.Handshake, error) {
	p.t.mu.RLock()
	timeout := p.t.activation.Timeout
	p.t.mu.RUnlock()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	return transport.WaitForHandshake(ctx, stdout)
}

func (p *Processes) startPluginProcess(ctx context.Context, plug manifest.Discovered, sup *lifecycle.Supervisor) (plugindomain.Endpoint, error) {
	token, err := transport.GenerateAuthToken()
	if err != nil {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("generate token: %w", err)
	}
	if token == "" {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, errors.New("empty auth token")
	}
	p.mu.RLock()
	launcher := p.launcher
	p.mu.RUnlock()
	if launcher == nil {
		launcher = p.defaultLauncher
	}

	proc, stdout, err := launcher(ctx, plug, 0, token)
	if err != nil {
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("launch plugin %s: %w", plug.Manifest.ID, err)
	}

	handshake, err := p.waitForReady(ctx, stdout)
	if err == nil {
		err = checkReportedListeners(plug.Manifest, handshake.Listeners)
	}
	if err != nil {
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("handshake plugin %s: %w", plug.Manifest.ID, err)
	}

	go func() {
		if _, err := io.Copy(io.Discard, stdout); err != nil && !errors.Is(err, os.ErrClosed) {
			p.errs.report(fmt.Errorf("read output of plugin %s: %w", plug.Manifest.ID, err))
		}
		p.closeResource("stdout of plugin "+plug.Manifest.ID, stdout)
	}()

	endpoint := plugindomain.Endpoint{
		Host:  "127.0.0.1",
		Port:  handshake.Port,
		Token: token,
	}
	p.t.clients.Add(token, transport.NewNetworkTransport())
	if err := transport.CheckPluginProtocol(ctx, p.t.clients.Get(endpoint), plug.Manifest, endpoint); err != nil {
		p.t.clients.Remove(token)
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return emptyEndpoint(), err
	}

	now := p.clock.Now()
	p.t.mu.Lock()
	if sup.GetState() != lifecycle.StateActivating {
		p.t.mu.Unlock()
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, lifecycle.ErrActivationAborted
	}

	if err := sup.Attach(ctx, proc, now); err != nil {
		p.t.mu.Unlock()
		p.cleanupAbortedProcess(proc, stdout, plug.Manifest.ID)
		return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}, fmt.Errorf("attach plugin %s: %w", plug.Manifest.ID, err)
	}
	sup.SetOnExit(func() {
		p.dropEndpoint(plug.Manifest.ID)

		p.closeStdin(plug.Manifest.ID, proc.Pid)
	})
	sup.Acquire(now)

	p.t.endpoints[plug.Manifest.ID] = endpoint
	p.t.listenerPorts[plug.Manifest.ID] = handshake.Listeners
	sup.SetRPCShutdown(createRPCShutdown(endpoint, p.t.clients.Get(endpoint), 2*time.Second))
	sup.FinishActivating()
	p.t.mu.Unlock()

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

func (p *Processes) defaultLauncher(
	ctx context.Context,
	plug manifest.Discovered,
	port int,
	token string,
) (*os.Process, io.ReadCloser, error) {
	binPath, err := resolvePluginBinary(plug)
	if err != nil {
		return nil, nil, err
	}

	p.t.mu.RLock()
	vaultPath := p.t.vaultPath
	p.t.mu.RUnlock()
	p.mu.RLock()
	limits := p.limits
	p.mu.RUnlock()

	workDir, err := process.ValidateWorkingDirectory(plug.Dir, plug.Dir, vaultPath)
	if err != nil {
		return nil, nil, fmt.Errorf("validate plugin working directory: %w", err)
	}

	cmd := exec.CommandContext(context.WithoutCancel(ctx), binPath, process.GetPluginArgs(port)...)
	process.ConfigureSysProcAttr(cmd)
	cmd.Dir = workDir

	pluginVars, err := p.getLaunchVars(ctx, plug.Manifest.ID, p.getRequirements(plug), port, token)
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
	if err := cmd.Start(); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("start %s: %w", binPath, err), stdin.Close(), stdout.Close())
	}

	if _, err := io.WriteString(stdin, token+"\n"); err != nil {
		p.killProcess(cmd.Process, plug.Manifest.ID)
		return nil, nil, errors.Join(fmt.Errorf("write token: %w", err), stdin.Close(), stdout.Close())
	}

	if err := process.ApplyLimits(cmd.Process.Pid, limits); err != nil {
		p.killProcess(cmd.Process, plug.Manifest.ID)
		return nil, nil, errors.Join(fmt.Errorf("apply process limits: %w", err), stdin.Close(), stdout.Close())
	}

	p.stdinsMu.Lock()
	if p.stdins == nil {
		p.stdins = make(map[stdinKey]io.Closer)
	}
	p.stdins[stdinKey{pluginID: plug.Manifest.ID, pid: cmd.Process.Pid}] = stdin
	p.stdinsMu.Unlock()

	return cmd.Process, stdout, nil
}

func (p *Processes) getRequirements(plug manifest.Discovered) []manifest.DependencyRequirement {
	p.t.mu.RLock()
	defer p.t.mu.RUnlock()
	return plug.Manifest.GetDependencyRequirements(plug.Dir, p.t.getPluginSettingsLocked(plug.Manifest.ID))
}

func createRPCShutdown(endpoint plugindomain.Endpoint, client *http.Client, gracePeriod time.Duration) func(context.Context) error {
	if endpoint.Host == "" && endpoint.Port == 0 {
		return nil
	}
	return func(ctx context.Context) error {
		timeout := gracePeriod
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
		rpcCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		client := numenv1connect.NewPluginLifecycleServiceClient(
			client,
			transport.GetBaseURL(endpoint),
			connect.WithInterceptors(transport.AuthInterceptor(endpoint.Token)),
		)
		req := connect.NewRequest(&v1.ShutdownRequest{
			TimeoutSeconds: int32(gracePeriod.Seconds()),
		})
		_, err := client.Shutdown(rpcCtx, req)
		return err
	}
}

func emptyEndpoint() plugindomain.Endpoint {
	return plugindomain.Endpoint{Host: "", Port: 0, Token: ""}
}

func isEmptyEndpoint(e plugindomain.Endpoint) bool {
	return e.Host == "" && e.Port == 0
}
