package pull

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/health"
	syncv1 "github.com/jiva-studio/numen/libs/shared/syncprotocol/gen/numen/sync/v1"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/healthkey"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
)

const (
	// KeyVersionJump is the prefix of the health item of a file whose revision has a version its history does not explain.
	KeyVersionJump = "version-jump"

	// maxVersionStep is the most a revision may rise above the highest version this device knows for its file. A restored
	// server loses edits and the author continues above them, so a gap is not an error, but no file has this many edits.
	maxVersionStep = 1 << 24
)

func isJump(rev *syncv1.Revision, floor uint64, record transfer.FileRecord, isKnown bool) bool {
	if isKnown {
		floor = max(floor, record.Version)
	}
	return rev.GetFileVersion() > floor && rev.GetFileVersion()-floor > maxVersionStep
}

// skipJump leaves a file whose revision jumps in version as it is: the position stays before the revision, so the file
// stays named in the warning while the server serves it, and the pull goes on with the other files.
func (r *run) skipJump(stored *syncv1.StoredRevision, meta note.Meta) error {
	if err := checkRoom(len(r.jumped)); err != nil {
		return err
	}
	r.hold(stored.GetSeq())
	text := meta.Path
	if stored.GetRevision().GetDeleted() {
		text = fmt.Sprintf("%x", stored.GetRevision().GetFileId())
	}
	if !slices.Contains(r.jumped, text) {
		r.jumped = append(r.jumped, text)
	}
	return nil
}

func (r *run) reportJumps(ctx context.Context) error {
	key := healthkey.Create(KeyVersionJump, string(r.in.VaultID))
	if len(r.jumped) == 0 {
		return r.deps.Health.Clear(ctx, key)
	}
	detail := strings.Join(r.jumped, ", ")
	if len(detail) > health.MaxDetailLength {
		detail = strings.ToValidUTF8(detail[:health.MaxDetailLength], "")
	}
	return r.deps.Health.Report(ctx, health.Item{
		Key: key, Severity: health.Error, Text: "The sync server sent a file with a version number its history does not explain, so it was not applied", Detail: detail, Action: nil,
	})
}
