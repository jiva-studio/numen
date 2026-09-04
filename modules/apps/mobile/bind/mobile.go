// Package bind is the core as a phone reaches it.
//
// One call starts a server on the loopback and answers with the port it
// listens on; everything asked of it after that is the schema in
// modules/libs/protocol, which is what the window asks too.
package bind

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/webui"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// running is the one server this process holds. Starting again while it stands
// answers with the port it already has.
var (
	mu      sync.Mutex
	running *held
)

type held struct {
	port   int
	stop   context.CancelFunc
	opened *webui.Opened
	server *http.Server
}

// Start opens the vault under dir and serves it. The port is the answer: the
// caller asked for no particular one.
//
// dir is the folder the platform gave the application to write in. Everything
// this installation keeps — the vault, the index, the settings — sits under it.
func Start(dir string) (int, error) {
	mu.Lock()
	defer mu.Unlock()
	if running != nil {
		return running.port, nil
	}

	// The libraries that look for a person's own folders are told where this
	// application's are, so nothing is looked for outside the sandbox.
	for _, named := range []string{"HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		if err := os.Setenv(named, dir); err != nil {
			return 0, err
		}
	}

	root := filepath.Join(dir, "vault")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return 0, err
	}

	cfg := configured(dir, os.Stderr)

	if err := seed(root); err != nil {
		return 0, err
	}
	// The list holds a folder under the name it resolves to, and that is the
	// name the vault is opened by.
	filed, err := known(cfg, root)
	if err != nil {
		return 0, err
	}

	ctx, stop := context.WithCancel(context.Background())
	opened, err := webui.Open(ctx, cfg, filed, io.Discard)
	if err != nil {
		stop()
		return 0, err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		stop()
		_ = opened.Close()
		return 0, err
	}

	server := &http.Server{
		Handler: allowing(withoutFiles(opened.API.Serving(http.NotFoundHandler()))),
	}
	go func() { _ = server.Serve(listener) }()

	running = &held{
		port:   listener.Addr().(*net.TCPAddr).Port,
		stop:   stop,
		opened: opened,
		server: server,
	}
	return running.port, nil
}

// Stop closes the server and the vault behind it. Stopping what is not running
// does nothing.
func Stop() error {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return nil
	}
	standing := running
	running = nil
	err := standing.server.Close()
	standing.stop()
	if closed := standing.opened.Close(); err == nil {
		err = closed
	}
	return err
}

// Port is what the server listens on, and zero while none does.
func Port() int {
	mu.Lock()
	defer mu.Unlock()
	if running == nil {
		return 0
	}
	return running.port
}

// configured is what this installation starts from: everything it keeps sits
// under the folder the platform gave it, and what the core went wrong at and
// carried on past goes where this process's own errors go, which is the log the
// platform collects.
func configured(dir string, out io.Writer) container.Config {
	return container.Config{
		IndexPath:    filepath.Join(dir, "index.db"),
		RegistryPath: filepath.Join(dir, "vaults.json"),
		SettingsPath: filepath.Join(dir, "settings.yaml"),
		ThemesPath:   filepath.Join(dir, "themes"),
		Trouble:      func(err error) { fmt.Fprintln(out, "numen:", err) },
	}
}

// known puts the vault on this installation's list, and answers with the path
// the list files it under. A vault already on it stays where it is.
func known(cfg container.Config, root string) (string, error) {
	registry, err := cfg.Registry()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if held, found, err := registry.Find(root); err != nil {
		return "", err
	} else if found {
		return held.Path, nil
	}
	added, err := usecase.Add{
		Identity: cfg.VaultIdentity(),
		Registry: registry,
		Now:      time.Now,
	}.Execute(root, "numen")
	if err != nil {
		return "", err
	}
	return added.Path, nil
}

// Seeded is the note a vault this package filled opens on. It stands in every
// seat at once: a parent above it, two children below, a sibling beside and a
// jump across.
const Seeded = "Physics.md"

// seed writes a small graph into a vault that holds none, so that a picture has
// something to draw.
func seed(root string) error {
	held, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range held {
		if strings.HasSuffix(entry.Name(), ".md") {
			return nil
		}
	}
	written := map[string]string{
		"Sciences.md": "---\ntitle: Sciences\nlinks:\n" +
			"  - to: \"[[Physics]]\"\n    role: child\n" +
			"  - to: \"[[Optics]]\"\n    role: child\n---\n\n" +
			"# Sciences\n\nWhat is asked of the world, sorted by the asking.\n",
		"Physics.md": "---\ntitle: Physics\nlinks:\n" +
			"  - to: \"[[Sciences]]\"\n    role: parent\n" +
			"  - to: \"[[Entropy]]\"\n    role: child\n" +
			"  - to: \"[[Tides]]\"\n    role: child\n    type: measures\n" +
			"  - to: \"[[Vellum]]\"\n    role: jump\n---\n\n" +
			"# Physics\n\nMatter, and what it does when nobody is looking.\n",
		"Optics.md":  "# Optics\n\nLight, bent and counted.\n",
		"Entropy.md": "# Entropy\n\nWhat a measure counts is the ways a thing can be arranged.\n",
		"Tides.md":   "# Tides\n\nTwo bulges, one turning planet, and a day with four of them in it.\n",
		"Vellum.md":  "# Vellum\n\nA skin scraped thin enough to write on and thick enough to fold.\n",
	}
	for name, text := range written {
		if err := os.WriteFile(filepath.Join(root, name), []byte(text), 0o644); err != nil {
			return fmt.Errorf("seeding %s: %w", name, err)
		}
	}
	return nil
}

// withoutFiles keeps the files of the vault, and what is made from them, off
// this server. Nothing the phone draws asks for either, and what is served here
// is served to any origin at all: a caller that reached it could otherwise set
// models running over the person's books and take a transcript away.
func withoutFiles(next http.Handler) http.Handler {
	kept := []string{
		"/assets/",
		"/" + numenv1connect.AssetServiceName + "/",
		"/" + numenv1connect.ArtifactServiceName + "/",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, one := range kept {
			if strings.HasPrefix(r.URL.EscapedPath(), one) {
				http.NotFound(w, r)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// allowing lets the page the platform serves ask this server, which sits on
// another origin than the one the webview loaded.
func allowing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		head := w.Header()
		head.Set("Access-Control-Allow-Origin", "*")
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
