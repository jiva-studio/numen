package conflict

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// markdown keeps the local text of a note in a copy next to it, then writes the head over the note.
type markdown struct{ host ports.Host }

func (s markdown) Resolve(ctx context.Context, in ports.ConflictInput) (ports.ConflictOutcome, error) {
	none := ports.ConflictOutcome{Applied: nil, Copy: ""}
	files := s.host.Files(in.Vault)
	text, _, err := files.Read(ctx, in.Path)
	if err != nil {
		return none, fmt.Errorf("read the local note: %w", err)
	}
	copyPath, isMade, err := findCopy(ctx, files, in.Path, in.Day, note.Hash(text))
	if err != nil {
		return none, err
	}
	if !isMade {
		if err := files.Create(ctx, copyPath, text); err != nil {
			return none, fmt.Errorf("write the copy: %w", err)
		}
	}
	if err := in.Head.Replace(ctx, in.Local.Stat); err != nil {
		return none, err
	}
	return ports.ConflictOutcome{Applied: []string{in.Path, copyPath}, Copy: copyPath}, nil
}

// findCopy returns the first name for the copy that is free or already holds the text, and whether the copy with the
// text is there.
func findCopy(ctx context.Context, files ports.Files, notePath string, day time.Time, hash string) (string, bool, error) {
	for number := 1; ; number++ {
		name := note.GetConflictCopyPath(notePath, day, number)
		stat, err := files.StatFile(ctx, name)
		if errors.Is(err, host.ErrNotExist) {
			return name, false, nil
		}
		if err != nil {
			return "", false, fmt.Errorf("look for the copy: %w", err)
		}
		if stat.Hash == hash {
			return name, true, nil
		}
	}
}
