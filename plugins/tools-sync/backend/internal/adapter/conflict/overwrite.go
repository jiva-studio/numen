package conflict

import (
	"context"
	"log/slog"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// overwrite writes the head over a file that is not a note and keeps no copy.
type overwrite struct{ log *slog.Logger }

func (s overwrite) Resolve(ctx context.Context, in ports.ConflictInput) (ports.ConflictOutcome, error) {
	if err := in.Head.Replace(ctx, in.Local.Stat); err != nil {
		return ports.ConflictOutcome{Applied: nil, Copy: ""}, err
	}
	s.log.InfoContext(ctx, "the head replaced a file changed on two computers", "path", in.Path, "localHash", in.Local.Hash, "headHash", in.Head.Hash)
	return ports.ConflictOutcome{Applied: []string{in.Path}, Copy: ""}, nil
}
