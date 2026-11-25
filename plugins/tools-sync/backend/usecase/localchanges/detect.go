package localchanges

import (
	"context"
	"errors"
	"maps"
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
	since, err := u.deps.State.ReadAbsences(ctx, in.VaultID)
	if err != nil {
		return transfer.Scan{}, err
	}
	gone, fresh := u.split(seen, in.Records)
	candidates, listed, err := u.separateListed(ctx, in, gone)
	if err != nil {
		return transfer.Scan{}, err
	}
	scan := transfer.Scan{}
	moves, err := u.keepGone(ctx, in, matchMoves(candidates, fresh, listed))
	if err != nil {
		return transfer.Scan{}, err
	}
	scan.Moves = moves
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
		first, isKnown := since[record.Path]
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
	if maps.Equal(since, kept) {
		return scan, nil
	}
	return scan, u.deps.State.WriteAbsences(ctx, in.VaultID, kept)
}

func (u *DetectMoves) isGone(ctx context.Context, in transfer.ScanInput, record transfer.FileRecord) (bool, error) {
	_, err := u.deps.Host.GetFiles(in.VaultID).StatFile(ctx, record.Path)
	if errors.Is(err, host.ErrFileAbsent) {
		return false, nil
	}
	if errors.Is(err, host.ErrNotExist) {
		return true, nil
	}
	return false, err
}

func (u *DetectMoves) split(seen map[string]note.FileSnapshot, records []transfer.FileRecord) (gone []transfer.FileRecord, fresh map[string]note.FileSnapshot) {
	byPath := map[string]bool{}
	for _, record := range records {
		byPath[record.Path] = true
		if _, isThere := seen[record.Path]; !isThere {
			gone = append(gone, record)
		}
	}
	fresh = map[string]note.FileSnapshot{}
	now := u.deps.Clock.Now()
	for path, one := range seen {
		if !byPath[path] && note.IsSyncable(path) && now.Sub(time.Unix(0, one.ModifiedAt)) >= settleTime {
			fresh[path] = one
		}
	}
	return gone, fresh
}

// separateListed returns the gone records without the records of files the vault lists as absent, which have no
// sighting and are not renamed, and the listed files that no record names: they are what the files of the
// gone records were renamed to.
func (u *DetectMoves) separateListed(ctx context.Context, in transfer.ScanInput, gone []transfer.FileRecord) ([]transfer.FileRecord, []host.AbsentFile, error) {
	if len(gone) == 0 {
		return nil, nil, nil
	}
	listed, err := u.deps.Host.GetAbsent(in.VaultID).ListAbsentFiles(ctx)
	if err != nil {
		return nil, nil, err
	}
	isListed := map[string]bool{}
	for _, one := range listed {
		isListed[one.Path] = true
	}
	isRecorded := map[string]bool{}
	for _, record := range in.Records {
		isRecorded[record.Path] = true
	}
	gone = slices.DeleteFunc(slices.Clone(gone), func(record transfer.FileRecord) bool { return isListed[record.Path] })
	unclaimed := slices.DeleteFunc(slices.Clone(listed), func(one host.AbsentFile) bool {
		return isRecorded[one.Path] || !note.IsSyncable(one.Path)
	})
	return gone, unclaimed, nil
}

func matchMoves(gone []transfer.FileRecord, fresh map[string]note.FileSnapshot, listed []host.AbsentFile) []transfer.Move {
	goneCount, freshCount, listedCount := map[key]int{}, map[key]int{}, map[key]int{}
	for _, record := range gone {
		goneCount[key{record.BaseHash, record.Size}]++
	}
	for _, one := range fresh {
		freshCount[key{one.Hash, one.Size}]++
	}
	for _, one := range listed {
		listedCount[key{strings.ToLower(one.Hash), one.Size}]++
	}
	var moves []transfer.Move
	for _, record := range gone {
		k := key{record.BaseHash, record.Size}
		if goneCount[k] != 1 || freshCount[k] != 1 || listedCount[k] != 0 {
			continue
		}
		for path, one := range fresh {
			if (key{one.Hash, one.Size}) == k {
				moves = append(moves, transfer.Move{Record: record, To: path})
			}
		}
	}
	moves = append(moves, matchListed(gone, listed, goneCount, freshCount)...)
	slices.SortFunc(moves, func(a, b transfer.Move) int { return strings.Compare(a.To, b.To) })
	return moves
}

// matchListed pairs the gone records with the listed files of the same hash and size when there are as many of the
// one as of the other and no present file has them. Files of one content are interchangeable, so the order of their
// paths pairs them.
func matchListed(gone []transfer.FileRecord, listed []host.AbsentFile, goneCount, freshCount map[key]int) []transfer.Move {
	targets := map[key][]string{}
	for _, one := range listed {
		k := key{strings.ToLower(one.Hash), one.Size}
		targets[k] = append(targets[k], one.Path)
	}
	sorted := slices.Clone(gone)
	slices.SortFunc(sorted, func(a, b transfer.FileRecord) int { return strings.Compare(a.Path, b.Path) })
	var moves []transfer.Move
	taken := map[key]int{}
	for _, record := range sorted {
		k := key{record.BaseHash, record.Size}
		if freshCount[k] != 0 || len(targets[k]) != goneCount[k] {
			continue
		}
		paths := slices.Sorted(slices.Values(targets[k]))
		moves = append(moves, transfer.Move{Record: record, To: paths[taken[k]]})
		taken[k]++
	}
	return moves
}

func (u *DetectMoves) keepGone(ctx context.Context, in transfer.ScanInput, moves []transfer.Move) ([]transfer.Move, error) {
	var kept []transfer.Move
	for _, move := range moves {
		isGone, err := u.isGone(ctx, in, move.Record)
		if err != nil {
			return nil, err
		}
		if isGone {
			kept = append(kept, move)
		}
	}
	return kept, nil
}
