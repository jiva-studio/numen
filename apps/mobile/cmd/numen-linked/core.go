package main

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	updater "github.com/jiva-studio/numen/plugins/tools-updater"
	updaterservice "github.com/jiva-studio/numen/plugins/tools-updater/backend/service"
)

// target is the window whose plugins the core serves.
const target = "desktop.editor"

// readHeaderTimeout is how long a client may take to send its request headers.
const readHeaderTimeout = 10 * time.Second

// linkedCore is the core serving on a loopback port: its address, the token a request carries, and what Close stops.
type linkedCore struct {
	url    string
	token  string
	watch  container.PluginWatch
	stop   context.CancelFunc
	opened *editor.Installation
	server *http.Server
}

func startCore(dir string, errOut io.Writer) (*linkedCore, error) {
	root := filepath.Join(dir, "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	report := func(err error) { writeLine(errOut, "numen-linked:", err) }
	cfg := newConfig(dir)
	res := container.NewResources()
	res.ErrorHandler = report
	management, supervision := container.NewPluginRoles(container.GetClock(res), target)
	if err := container.InitializePluginRoles(cfg.IndexPath, res, management, supervision, report); err != nil {
		return nil, err
	}
	if err := management.Discovery.RegisterBuiltIn(updater.Manifest, serveLinked(updaterservice.Mount)); err != nil {
		return nil, err
	}
	execution, preferences, endpoints := container.CreatePluginHost(management, supervision)
	execution.Runtime.SetSafeMode(cfg.SafeMode)
	shown, err := cfg.EnsureVault(res, root, "numen", false)
	if err != nil {
		return nil, errors.Join(err, container.StopPlugins(execution.Watch, context.Background()))
	}
	ctx, stop := context.WithCancel(context.Background())
	opened, err := cfg.OpenWindow(execution, preferences, endpoints, res, ctx, container.WindowRequest{Named: shown.Path, Out: errOut})
	if err != nil {
		stop()
		return nil, errors.Join(err, container.StopPlugins(execution.Watch, context.Background()))
	}
	core := &linkedCore{watch: execution.Watch, stop: stop, opened: opened}
	if err := core.serve(ctx, report); err != nil {
		return nil, errors.Join(err, core.Close())
	}
	return core, nil
}

func newConfig(dir string) container.Config {
	return container.Config{
		Paths: container.Paths{
			IndexPath:       filepath.Join(dir, "index.db"),
			RegistryPath:    filepath.Join(dir, "vaults.json"),
			SettingsPath:    filepath.Join(dir, "numen.json"),
			ThemesPath:      filepath.Join(dir, "themes"),
			PluginCachePath: filepath.Join(dir, "plugin-cache"),
		},
	}
}

func serveLinked(mount func(*server.Server) error) func(map[string]string) (http.Handler, error) {
	return func(vars map[string]string) (http.Handler, error) { return server.NewLinkedHandler(vars, mount) }
}

func (c *linkedCore) serve(ctx context.Context, report func(error)) error {
	handler, token, err := c.opened.NewLocalHandler(http.NotFoundHandler())
	if err != nil {
		return err
	}
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	c.url, c.token = "http://"+listener.Addr().String(), token
	c.server = &http.Server{ReadHeaderTimeout: readHeaderTimeout, Handler: handler}
	go func() {
		if err := c.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			report(err)
		}
	}()
	return nil
}

// Close stops the server, the window and the plugins.
func (c *linkedCore) Close() error {
	var err error
	if c.server != nil {
		err = c.server.Close()
	}
	c.stop()
	return errors.Join(err, c.opened.Close(), container.StopPlugins(c.watch, context.Background()))
}
