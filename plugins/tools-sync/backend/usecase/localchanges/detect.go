package localchanges

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// DetectState is the sightings and the absences of each vault.
type DetectState interface {
	ports.Sightings
	ports.Absences
}

// DetectDeps are what the use case needs. Delay is how long a removed file is watched before its removal is sent.
type DetectDeps struct {
	Host  ports.Host
	State DetectState
	Clock ports.Clock
	Delay func() time.Duration
}

// DetectMoves finds the synced files that have another path and the ones that are gone, from the sightings the scan
// of changes keeps.
type DetectMoves struct{ deps DetectDeps }

// NewDetectMoves returns the use case over the dependencies.
func NewDetectMoves(deps DetectDeps) *DetectMoves { return &DetectMoves{deps: deps} }

type key struct {
	hash string
	size int64
}

// Execute returns the moves and the removals of the records. A move is a record whose file is gone and a settled file
// without a record that has its hash and size, when both are the only ones with them. A removal is a record whose
// file stayed gone for the removal delay.
func (u *DetectMoves) Execute(ctx context.Context, in transfer.ScanInput) (transfer.Scan, error) {
	seen, err := u.deps.State.ReadSightings(ctx, in.VaultID)
	if err != nil {
		return transfer.Scan{}, err
	}
	absent, err := u.deps.State.ReadAbsences(ctx, in.VaultID)
	if err != nil {
		return transfer.Scan{}, err
	}
	gone, fresh := u.split(seen, in.Records)
	scan := transfer.Scan{}
	scan.Moves = pairMoves(gone, fresh)
	moved := map[string]bool{}
	for _, move := range scan.Moves {
		moved[move.Record.Path] = true
	}
	now := u.deps.Clock.Now().UnixNano()
	kept := map[string]int64{}
	for _, record := range gone {
		if moved[record.Path] {
			continue
		}
		first, isKnown := absent[record.Path]
		if !isKnown {
			first = now
		}
		kept[record.Path] = first
		if time.Duration(now-first) < u.deps.Delay() {
			continue
		}
		if isGone, err := u.isGone(ctx, in, record); err != nil {
			return transfer.Scan{}, err
		} else if isGone {
			scan.Removals = append(scan.Removals, record)
		}
	}
	return scan, u.deps.State.WriteAbsences(ctx, in.VaultID, kept)
}

func (u *DetectMoves) isGone(ctx context.Context, in transfer.ScanInput, record transfer.FileRecord) (bool, error) {
	_, err := u.deps.Host.Files(in.VaultID).StatFile(ctx, record.Path)
	if errors.Is(err, host.ErrNotExist) {
		return true, nil
	}
	return false, err
}

func (u *DetectMoves) split(seen map[string]note.Seen, records []transfer.FileRecord) (gone []transfer.FileRecord, fresh map[string]note.Seen) {
	byPath := map[string]bool{}
	for _, record := range records {
		byPath[record.Path] = true
		if _, isThere := seen[record.Path]; !isThere {
			gone = append(gone, record)
		}
	}
	fresh = map[string]note.Seen{}
	now := u.deps.Clock.Now()
	for path, one := range seen {
		if !byPath[path] && note.IsSyncable(path) && now.Sub(time.Unix(0, one.ModifiedAt)) >= settleTime {
			fresh[path] = one
		}
	}
	return gone, fresh
}

func pairMoves(gone []transfer.FileRecord, fresh map[string]note.Seen) []transfer.Move {
	goneCount, freshCount := map[key]int{}, map[key]int{}
	for _, record := range gone {
		goneCount[key{record.BaseHash, record.Size}]++
	}
	for _, one := range fresh {
		freshCount[key{one.Hash, one.Size}]++
	}
	var moves []transfer.Move
	for _, record := range gone {
		k := key{record.BaseHash, record.Size}
		if goneCount[k] != 1 || freshCount[k] != 1 {
			continue
		}
		for path, one := range fresh {
			if (key{one.Hash, one.Size}) == k {
				moves = append(moves, transfer.Move{Record: record, To: path})
			}
		}
	}
	slices.SortFunc(moves, func(a, b transfer.Move) int { return strings.Compare(a.To, b.To) })
	return moves
}
