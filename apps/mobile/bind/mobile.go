// Package bind exposes the core to the mobile platform.
//
// One call starts a server on the loopback interface and returns its listening
// port. All requests to it follow the schema in libs/core/protocol, the same
// schema the desktop window uses.
package bind

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/kernel/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/kernel/container"
)

// Core holds the one server the platform starts. Starting again while active
// answers with the port it already has.
type Core struct {
	mu      sync.Mutex
	running *runningCore
}

// NewCore returns a core with no server running.
func NewCore() *Core { return &Core{} }

type runningCore struct {
	port   int
	isDown atomic.Bool
	stop   context.CancelFunc
	opened *editor.Installation
	server *http.Server
}

// Start opens the vault under dir and serves it. It returns the port, which the
// server chooses.
//
// dir is the folder the platform provides for application data. The vault, the
// index and the settings are stored under it.
func (c *Core) Start(dir string) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running != nil {
		return c.running.port, nil
	}

	if err := setHomeFolders(dir); err != nil {
		return 0, err
	}

	root := filepath.Join(dir, "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return 0, err
	}

	cfg := makeConfig(dir, os.Stderr)

	// The list holds a folder under the name it resolves to, and that is the
	// name the vault is opened by.
	filed, err := registerVault(cfg, root)
	if err != nil {
		return 0, err
	}

	ctx, stop := context.WithCancel(context.Background())
	opened, err := container.OpenWindow(cfg, ctx, filed, os.Stderr)
	if err != nil {
		stop()
		return 0, err
	}

	// The themes come from the installation. The config has no plugin host, so
	// the catalog offers the default theme and the person's own. If the themes
	// folder cannot be created, only the default theme is loaded. A theme
	// named in the settings that is missing from the catalog is reported on
	// the platform log.
	if wrong := container.SetThemes(cfg, opened, opened.RecordThemeWarning); wrong != nil {
		fmt.Fprintln(os.Stderr, "themes:", wrong)
	}
	listener, err := listenLocal(ctx)
	if err != nil {
		stop()
		return 0, errors.Join(err, opened.Close())
	}

	// The phone shows the vault, its notes and the tree they are filed in, and
	// needs no other service. The socket is reachable by every process on the
	// phone, so only those three services are mounted. The settings file, which
	// contains the model API keys, and the files on disk next to the notes
	// (books and the model sets that run over them) are not served.
	server := &http.Server{
		ReadHeaderTimeout: readHeaderTimeout,
		Handler: allowOrigin(opened.NewHandler(
			http.NotFoundHandler(),
			numenv1connect.VaultServiceName,
			numenv1connect.NoteServiceName,
			numenv1connect.FileServiceName,
		)),
	}
	c.running = &runningCore{
		port:   listener.port,
		stop:   stop,
		opened: opened,
		server: server,
	}
	current := c.running
	// Start has already returned the port, so a serve that ends on its own
	// leaves the caller with a port that has no listener. The error is written to
	// the stream the platform collects, and the port is reported as 0.
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		current.isDown.Store(true)
		cfg.ErrorHandler(err)
	}()
	return c.running.port, nil
}

func setHomeFolders(dir string) error {
	for _, named := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		if err := os.Setenv(named, dir); err != nil {
			return err
		}
	}
	return nil
}

// localListener is a listener on a loopback port, and the port.
type localListener struct {
	net.Listener
	port int
}

func listenLocal(ctx context.Context) (localListener, error) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		return localListener{}, err
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return localListener{}, errors.Join(fmt.Errorf("listening on %s, which is no TCP address", listener.Addr()), listener.Close())
	}
	return localListener{Listener: listener, port: addr.Port}, nil
}

// Stop closes the server and the vault behind it. Stopping what is not running
// does nothing.
func (c *Core) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running == nil {
		return nil
	}
	current := c.running
	c.running = nil
	err := current.server.Close()
	current.stop()
	if closed := current.opened.Close(); err == nil {
		err = closed
	}
	return err
}

// Port returns the listening port of the server, or zero when none is running.
func (c *Core) Port() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.running == nil || c.running.isDown.Load() {
		return 0
	}
	return c.running.port
}

func makeConfig(dir string, out io.Writer) container.Config {
	return container.NewConfig(container.Config{
		Resources: container.NewResources(),
		ErrorHandler: func(err error) {
			if _, writeErr := fmt.Fprintln(out, "numen:", err); writeErr != nil {
				return
			}
		},
		Paths: container.Paths{IndexPath: filepath.Join(dir, "index.db"), RegistryPath: filepath.Join(dir, "vaults.json"), ThemesPath: filepath.Join(dir, "themes")},
	})
}

func registerVault(cfg container.Config, root string) (string, error) {
	registered, err := cfg.EnsureVault(root, "numen", false)
	if err != nil {
		return "", err
	}
	return registered.Path, nil
}

// Page is the origin from which the platform serves this application's page:
// `androidScheme` in capacitor.config.ts, with no port. The core serves on the
// loopback interface, so requests from the page are cross-origin.
const Page = "http://localhost"

// readHeaderTimeout is how long a client of the local service may take to send its request headers.
const readHeaderTimeout = 10 * time.Second

func allowOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		head := w.Header()
		head.Set("Access-Control-Allow-Origin", Page)
		head.Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		head.Set("Access-Control-Allow-Headers", strings.Join([]string{
			"Content-Type", "Connect-Protocol-Version", "Connect-Timeout-Ms",
			"Connect-Accept-Encoding", "Connect-Content-Encoding", "X-User-Agent",
		}, ", "))
		head.Set("Access-Control-Expose-Headers", strings.Join([]string{
			"Connect-Content-Encoding", "Connect-Accept-Encoding",
			"Grpc-Status", "Grpc-Message", "Grpc-Status-Details-Bin",
		}, ", "))
		head.Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
