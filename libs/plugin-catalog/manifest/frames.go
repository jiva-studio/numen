package manifest

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrInvalidFrame is a frame origin that is not an https origin.
var ErrInvalidFrame = errors.New("invalid frame origin")

// listFrameIssues checks that each origin the plugin frames is an https origin: a scheme and a host,
// with no path, query or fragment, as a frame-src source of a Content-Security-Policy names it.
func (m Manifest) listFrameIssues() []error {
	var issues []error
	for _, origin := range m.Frames {
		if !domain.IsHTTPSOrigin(origin) {
			issues = append(issues, fmt.Errorf("%w: %q is not https://host", ErrInvalidFrame, origin))
		}
	}
	return issues
}
