package pluginport

import (
	"errors"
	"fmt"
	"net/url"
)

// ErrInvalidFrame is a frame origin that is not an https origin.
var ErrInvalidFrame = errors.New("invalid frame origin")

// listFrameIssues checks that each origin the plugin frames is an https origin: a scheme and a host,
// with no path, query or fragment, as a frame-src source of a Content-Security-Policy names it.
func (m PluginManifest) listFrameIssues() []error {
	var issues []error
	for _, origin := range m.Frames {
		if !isHTTPSOrigin(origin) {
			issues = append(issues, fmt.Errorf("%w: %q is not https://host", ErrInvalidFrame, origin))
		}
	}
	return issues
}

func isHTTPSOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.Path == "" &&
		parsed.RawQuery == "" && parsed.Fragment == "" && parsed.Opaque == "" && !parsed.ForceQuery
}
