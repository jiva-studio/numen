package launch

import (
	"context"
	"errors"
)

// StartMissingFile starts the recognition of a file that has no text of its own, and returns
// StateSkipped for a file with a text layer under the mode whenNoTextLayer.
func (l *Launcher) StartMissingFile(_ context.Context, _ string) (string, error) {
	return "", errors.New("StartMissingFile is not implemented")
}
