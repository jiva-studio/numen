package editor

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/csp"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// getPolicy is the policy a page of the window is served under: media and frames from the loopback
// socket, and frames from the origins the enabled plugins declare, read at each request so a plugin
// switched off frames nothing from the next page on.
func (s *Services) getPolicy(played []string) string {
	frames := append([]string(nil), played...)
	if s.PluginServices != nil {
		frames = append(frames, s.PluginServices.ListFrameOrigins()...)
	}
	return csp.Sources{Media: played, Frames: frames}.Policy()
}

// pluginPagePolicy is the policy a plugin's page on the loopback socket is served under: its own
// scripts, styles and pictures, and frames from the origins the plugin declares alone.
func pluginPagePolicy(frames []string) string {
	return "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; " +
		"frame-src " + strings.Join(append([]string{"'self'"}, frames...), " ") +
		"; object-src 'none'; base-uri 'none'; form-action 'none'"
}

// pluginPageReferrer is the Referrer-Policy of a file of a plugin's store on the loopback socket. A
// site that checks which page frames its player is sent the origin of the socket.
const pluginPageReferrer = "strict-origin-when-cross-origin"

// pluginFileSegment is the first segment below the token of a file of a plugin's store. It names no
// vault.
const pluginFileSegment = "plugin"

// PluginFileAddress is where the file of a plugin's store of the vault is loaded from on the loopback
// socket, and nowhere where the socket is not answering. The address names the size of the file, so a
// file written again with another size is a different address.
func (l *Loopback) PluginFileAddress(vault domain.Vault, address file.Address, size int64) string {
	if l == nil || l.stopped.Load() {
		return ""
	}
	return l.address + "/" + l.token + "/" + pluginFileSegment + "/" + url.PathEscape(string(vault.ID)) +
		"/" + url.PathEscape(address.PluginID) + "/" + url.PathEscape(address.Path) +
		"?size=" + strconv.FormatInt(size, 10)
}

// servePluginFile serves a file of a plugin's store of a vault: rest is
// <vault id>/<plugin id>/<path>, each escaped.
//
// The window loads from the toolkit's own scheme, and a browser sends no
// Referer from a page with a non-http scheme; a video site that checks the
// Referer rejects playback in a frame of such a page. A page served here has an
// http origin, so the Referer is sent, and it frames only the origins its
// plugin declares.
func (l *Loopback) servePluginFile(w http.ResponseWriter, r *http.Request, rest string) {
	segments := strings.SplitN(rest, "/", 3)
	if len(segments) != 3 {
		http.Error(w, "not a file of a plugin", http.StatusBadRequest)
		return
	}
	vaultID, errVault := url.PathUnescape(segments[0])
	pluginID, errPlugin := url.PathUnescape(segments[1])
	path, errPath := url.PathUnescape(segments[2])
	size, errSize := strconv.ParseInt(r.URL.Query().Get("size"), 10, 64)
	if err := errors.Join(errVault, errPlugin, errPath, errSize); err != nil {
		http.Error(w, "not a file of a plugin", http.StatusBadRequest)
		return
	}
	target, ok := l.files.findVault(vaultID)
	if !ok {
		http.Error(w, errNoVaultNamed, http.StatusNotFound)
		return
	}
	if l.files.openPluginFile == nil {
		refuse(w, errNoVault)
		return
	}
	opened, err := l.files.openPluginFile.Execute(r.Context(), target, pluginID, path)
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		refuse(w, err)
		return
	}
	defer closeAndReport(l.files.state.reportTask, opened.Body)
	if opened.Size != size {
		refuse(w, file.ErrChanged)
		return
	}
	var frames []string
	if plugins := l.getPluginServices(); plugins != nil {
		frames = plugins.GetFrameOrigins(pluginID)
	}
	w.Header().Set("Cache-Control", immutable)
	w.Header().Set("Content-Security-Policy", pluginPagePolicy(frames))
	w.Header().Set("Referrer-Policy", pluginPageReferrer)
	if named := getMediaType(l.files.mediaTypes, path); named != "" {
		w.Header().Set("Content-Type", named)
	}
	http.ServeContent(w, r, path, time.Time{}, opened.Body)
}

// setPluginServices names the plugins whose frame origins the files of their stores are served with.
func (l *Loopback) setPluginServices(plugins PluginServices) {
	if l != nil && plugins != nil {
		l.plugins.Store(&plugins)
	}
}

func (l *Loopback) getPluginServices() PluginServices {
	if found := l.plugins.Load(); found != nil {
		return *found
	}
	return nil
}
