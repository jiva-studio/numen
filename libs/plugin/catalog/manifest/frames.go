package manifest

import (
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// ErrInvalidFrame is a frame origin that is not an https origin.
var ErrInvalidFrame = errors.New("invalid frame origin")

func (m Manifest) listFrameIssues() []error {
	var issues []error
	for _, origin := range m.Contributes.FrameOrigins {
		if !domain.IsHTTPSOrigin(origin) {
			issues = append(issues, fmt.Errorf("%w: %q is not https://host", ErrInvalidFrame, origin))
		}
	}
	return issues
}
