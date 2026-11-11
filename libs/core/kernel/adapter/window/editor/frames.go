package editor

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/csp"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (s pageSet) getPolicy(played []string) string {
	frames := append([]string(nil), played...)
	if s.plugins != nil {
		frames = append(frames, s.plugins.ListFrameOrigins()...)
	}
	return csp.Sources{Media: played, Frames: frames}.Policy()
}

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
	if l.files.open == nil {
		refuse(w, errNoVault)
		return
	}
	opened, err := l.files.open.ExecutePluginFile(r.Context(), l.files.state.GetCurrentVault(), vaultID, pluginID, path)
	if errors.Is(err, file.ErrNoVaultNamed) {
		http.Error(w, errNoVaultNamed, http.StatusNotFound)
		return
	}
	if errors.Is(err, fs.ErrNotExist) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, file.ErrNoPluginFiles) {
		refuse(w, errNoVault)
		return
	}
	if err != nil {
		refuse(w, err)
		return
	}
	defer closeAndReport(l.files.state.Window.AddTask, opened.Body)
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
	if opened.MediaType != "" {
		w.Header().Set("Content-Type", opened.MediaType)
	}
	http.ServeContent(w, r, path, time.Time{}, opened.Body)
}

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
