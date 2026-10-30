package wire

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PluginRoute is the path prefix of everything a window loads from a plugin. /plugin/<id>/<rest>
// is the file <rest> of the plugin's folder where the folder holds one, and otherwise reaches the
// backend of the plugin at /plugin/<rest>.
const PluginRoute = "/plugin/"

func setPluginAssetContentType(w http.ResponseWriter, path string) {
	switch filepath.Ext(path) {
	case ".js", ".mjs":
		w.Header().Set("Content-Type", "application/javascript")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".json":
		w.Header().Set("Content-Type", "application/json")
	case ".md":
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	case ".svg":
		w.Header().Set("Content-Type", "image/svg+xml")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	case ".wasm":
		w.Header().Set("Content-Type", "application/wasm")
	}
}

// ServePluginFile writes a file of a plugin's folder named on the plugin route and reports
// whether it answered the request. A path that names no file of the folder is left unanswered,
// for the plugin's backend. getPluginDir is the folder of an installed plugin, and nil where the
// window has no plugins.
func ServePluginFile(w http.ResponseWriter, r *http.Request, getPluginDir func(id string) (string, bool)) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return true
	}
	if getPluginDir == nil {
		return false
	}
	assetPath := strings.TrimPrefix(r.URL.Path, PluginRoute)
	parts := strings.SplitN(assetPath, "/", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" || !filepath.IsLocal(parts[1]) {
		http.Error(w, "invalid plugin path", http.StatusBadRequest)
		return true
	}

	pluginDir, ok := getPluginDir(parts[0])
	if !ok {
		http.Error(w, "plugin not found", http.StatusNotFound)
		return true
	}

	file, info, err := openPluginAsset(r.Context(), pluginDir, parts[1])
	if err != nil {
		switch {
		case errors.Is(err, os.ErrNotExist):
			return false
		case errors.Is(err, os.ErrPermission), errors.Is(err, errPluginAssetEscapes):
			http.Error(w, "forbidden: path escapes plugin directory", http.StatusForbidden)
		default:
			http.Error(w, "cannot read plugin file", http.StatusInternalServerError)
		}
		return true
	}
	content, err := io.ReadAll(file)
	if err = errors.Join(err, file.Close()); err != nil {
		http.Error(w, "cannot read plugin file", http.StatusInternalServerError)
		return true
	}

	setPluginAssetContentType(w, parts[1])
	http.ServeContent(w, r, info.Name(), info.ModTime(), bytes.NewReader(content))
	return true
}

// errPluginAssetEscapes marks a link that leads out of the plugin directory.
var errPluginAssetEscapes = errors.New("plugin asset path escapes the plugin directory")

// openPluginAsset opens a regular file below pluginDir. The directory is opened once and
// the name is resolved inside it, so no link can lead out of it.
func openPluginAsset(ctx context.Context, pluginDir, name string) (_ *os.File, _ os.FileInfo, err error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	root, err := os.OpenRoot(pluginDir)
	if err != nil {
		return nil, nil, err
	}
	defer closeAndJoin(&err, root)
	file, err := root.Open(name)
	if err != nil {
		return nil, nil, classifyPluginAssetError(root, name, err)
	}
	info, err := file.Stat()
	if err != nil {
		return nil, nil, errors.Join(err, file.Close())
	}
	if info.IsDir() {
		return nil, nil, errors.Join(os.ErrNotExist, file.Close())
	}
	return file, info, nil
}

// classifyPluginAssetError names the reason root.Open failed. A name whose links lead out of
// the plugin directory is an escape; every other failure keeps its own error.
func classifyPluginAssetError(root *os.Root, name string, openErr error) error {
	if errors.Is(openErr, os.ErrNotExist) || errors.Is(openErr, os.ErrPermission) {
		return openErr
	}
	if isPluginAssetEscape(root, name) {
		return errPluginAssetEscapes
	}
	return openErr
}

// maxPluginAssetLinks bounds the links followed while checking one name.
const maxPluginAssetLinks = 40

// isPluginAssetEscape follows the links on name one component at a time and reports whether
// the walk reaches an absolute target or climbs above the root.
func isPluginAssetEscape(root *os.Root, name string) bool {
	pending := strings.Split(filepath.ToSlash(name), "/")
	var resolved []string
	links := 0
	for len(pending) > 0 {
		component := pending[0]
		pending = pending[1:]
		switch component {
		case "", ".":
			continue
		case "..":
			if len(resolved) == 0 {
				return true
			}
			resolved = resolved[:len(resolved)-1]
			continue
		}
		path := strings.Join(append(resolved, component), "/")
		target, err := root.Readlink(path)
		if err != nil {
			resolved = append(resolved, component)
			continue
		}
		links++
		if links > maxPluginAssetLinks {
			return false
		}
		target = filepath.ToSlash(target)
		if filepath.IsAbs(target) || strings.HasPrefix(target, "/") {
			return true
		}
		pending = append(strings.Split(target, "/"), pending...)
	}
	return false
}
