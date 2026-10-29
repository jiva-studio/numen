// Package bind exposes the core to the mobile platform.
//
// One call starts a server on the loopback interface and returns its listening
// port. All requests to it follow the schema in modules/libs/protocol, the same
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

	"github.com/jiva-studio/numen/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/libs/core/container"
)

// running is the one server this process holds. Starting again while active
// answers with the port it already has.
var (
	mu      sync.Mutex
	running *runningCore
)

type runningCore struct {
	port int
	// isDown is set when the server has stopped on its own, after which the
	// port serves nothing.
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
func Start(dir string) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	if running != nil {
		return running.port, nil
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
	made, err := cfg.GetEditorAssembly(ctx)
	if err != nil {
		stop()
		return 0, err
	}
	opened, err := editor.Open(ctx, made, filed, os.Stderr)
	if err != nil {
		stop()
		return 0, err
	}

	// The themes come from the installation. If the themes folder cannot be
	// created, only the themes bundled with this binary are loaded. A theme
	// named in the settings that is missing from the catalogue is reported on
	// the platform log.
	themes, wrong := cfg.Themes(opened.SayTheme)
	if wrong != nil {
		fmt.Fprintln(os.Stderr, "themes:", wrong)
	}
	opened.Services.Themes = themes
	listener, err := listenLocal()
	if err != nil {
		stop()
		_ = opened.Close()
		return 0, err
	}

	// The phone shows the vault, its notes and the tree they are filed in, and
	// needs no other service. The socket is reachable by every process on the
	// phone, so only those three services are mounted. The settings file, which
	// contains the model API keys, and the files on disk next to the notes
	// (books and the model sets that run over them) are not served.
	server := &http.Server{
		Handler: allowOrigin(opened.Services.NewHandler(
			http.NotFoundHandler(),
			numenv1connect.VaultServiceName,
			numenv1connect.NoteServiceName,
			numenv1connect.FileServiceName,
		)),
	}
	running = &runningCore{
		port:   listener.port,
		stop:   stop,
		opened: opened,
		server: server,
	}
	current := running
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
	return running.port, nil
}

// setHomeFolders points the libraries that resolve user folders at this
// application's folders, so nothing is resolved outside the sandbox.
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

// listenLocal is a listener on a free loopback port.
func listenLocal() (localListener, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return localListener{}, err
	}
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return localListener{}, fmt.Errorf("listening on %s, which is no TCP address", listener.Addr())
	}
	return localListener{Listener: listener, port: addr.Port}, nil
}

// Stop closes the server and the vault behind it. Stopping what is not running
// does nothing.
func Stop() error {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return nil
	}
	current := running
	running = nil
	err := current.server.Close()
	current.stop()
	if closed := current.opened.Close(); err == nil {
		err = closed
	}
	return err
}

// Port returns the listening port of the server, or zero when none is running.
func Port() int {
	mu.Lock()
	defer mu.Unlock()
	if running == nil || running.isDown.Load() {
		return 0
	}
	return running.port
}

// makeConfig returns the configuration for this installation: all stored data
// is under the folder the platform provides, and errors the core continues past
// are written to the stream the platform collects.
func makeConfig(dir string, out io.Writer) container.Config {
	return container.Config{
		IndexPath:    filepath.Join(dir, "index.db"),
		RegistryPath: filepath.Join(dir, "vaults.json"),
		ThemesPath:   filepath.Join(dir, "themes"),
		ErrorHandler: func(err error) { fmt.Fprintln(out, "numen:", err) },
	}
}

// registerVault puts the vault on this installation's list, and answers with
// the path the list files it under. A vault already on it stays where it is.
func registerVault(cfg container.Config, root string) (string, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if target, found, err := registry.FindVault(root); err != nil {
		return "", err
	} else if found {
		return target.Path, nil
	}
	added, err := cfg.AddVault(registry, root, "numen")
	if err != nil {
		return "", err
	}
	return added.Path, nil
}

// Page is the origin from which the platform serves this application's page:
// `androidScheme` in capacitor.config.ts, with no port. The core serves on the
// loopback interface, so requests from the page are cross-origin.
const Page = "http://localhost"

// allowOrigin adds the CORS headers that permit requests from the origin the
// webview loaded.
//
// It allows only Page. The socket is on the loopback interface and reachable by
// every process on the phone, so any page in a browser can send requests
// to it; allowing a single origin makes the browser block such requests and
// their responses. CORS is not authentication: it does not
// restrict non-browser clients.
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
