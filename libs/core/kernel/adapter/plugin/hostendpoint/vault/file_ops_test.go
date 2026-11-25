package vault

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
)

// holder is a plugin that makes the calls.
type holder struct {
	id string
}

func (h holder) Authenticate(string) (string, error) { return h.id, nil }

func (holder) IsTakingFile(string, string, []domain.NoteType) bool { return true }

// recordedOps answers the file operations with the error it holds and keeps what it was asked.
type recordedOps struct {
	err      error
	from, to string
	pluginID string
	name     string
	path     string
	hash     string
	baseline *domain.FileStat
	count    int
}

func (o *recordedOps) MoveFile(_ context.Context, from, to string, baseline *domain.FileStat) (int, error) {
	o.from, o.to, o.baseline = from, to, baseline
	return o.count, o.err
}

func (o *recordedOps) RemoveFile(_ context.Context, path string, baseline *domain.FileStat) (int, error) {
	o.path, o.baseline = path, baseline
	return o.count, o.err
}

func (o *recordedOps) PlaceFile(
	_ context.Context, pluginID, name, path, expectedHash string, baseline *domain.FileStat,
) (domain.FileStat, error) {
	o.pluginID, o.name, o.path, o.hash, o.baseline = pluginID, name, path, expectedHash, baseline
	return domain.FileStat{Path: path, Size: 3, ModTime: time.Unix(7, 0), Hash: expectedHash, Kind: domain.KindFile}, o.err
}

// recordedRanges answers a ranged read with the bytes and the error it holds.
type recordedRanges struct {
	err           error
	content       []byte
	path          string
	start, length int64
}

func (r *recordedRanges) ReadFileRange(_ context.Context, path string, start, length int64) ([]byte, domain.FileStat, error) {
	r.path, r.start, r.length = path, start, length
	return r.content, domain.FileStat{Path: path, Size: 10, ModTime: time.Unix(5, 0), Hash: "abc", Kind: domain.KindFile}, r.err
}

func createFileCalls(caller holder, access *Access) *filesCalls {
	return &filesCalls{&calls{auth: caller, source: shownAccess{access: access}}}
}

func getCaller() holder {
	return holder{id: "md.numen.syncer"}
}

type failure struct {
	err  error
	code v1.ErrorCode
}

func getFailures() []failure {
	return []failure{
		{fmt.Errorf("a: %w", fs.ErrNotExist), v1.ErrorCode_ERROR_CODE_NOT_FOUND},
		{fmt.Errorf("a: %w", vaultdomain.ErrAlreadyExists), v1.ErrorCode_ERROR_CODE_ALREADY_EXISTS},
		{fmt.Errorf("a: %w", vaultdomain.ErrOutside), v1.ErrorCode_ERROR_CODE_OUTSIDE_VAULT},
		{fmt.Errorf("a: %w", edit.ErrStale), v1.ErrorCode_ERROR_CODE_CONFLICT},
		{fmt.Errorf("a: %w", vaultdomain.ErrLockedByAnother), v1.ErrorCode_ERROR_CODE_LOCKED},
	}
}

func getBaseline() *v1.FileStat {
	return &v1.FileStat{Path: "Maps/Delta.pdf", Size: 9, Mtime: time.Unix(3, 4).UnixNano(), Hash: "h1"}
}

func TestMoveFileMovesThroughTheOperationsWithTheBaselineAndCountsTheFiles(t *testing.T) {
	ops := &recordedOps{count: 4}
	calls := createFileCalls(getCaller(), &Access{Ops: ops})

	res, err := calls.MoveFile(t.Context(), connect.NewRequest(&v1.HostFileServiceMoveFileRequest{From: "Maps", To: "Atlas", Baseline: getBaseline()}))

	if err != nil || res.Msg.GetError() != 0 || res.Msg.GetMovedCount() != 4 {
		t.Fatalf("the move answered %v (%v), want 4 files moved", res.Msg, err)
	}
	if ops.from != "Maps" || ops.to != "Atlas" || ops.baseline == nil || ops.baseline.Hash != "h1" || ops.baseline.Path != "Maps/Delta.pdf" {
		t.Errorf("the operations were asked %q to %q over %+v", ops.from, ops.to, ops.baseline)
	}
}

func TestMoveFileWithoutABaselinePassesNone(t *testing.T) {
	ops := &recordedOps{count: 1}

	_, err := createFileCalls(getCaller(), &Access{Ops: ops}).MoveFile(t.Context(), connect.NewRequest(&v1.HostFileServiceMoveFileRequest{From: "a.pdf", To: "b.pdf"}))

	if err != nil || ops.baseline != nil {
		t.Errorf("the move without a baseline passed %+v (%v), want none", ops.baseline, err)
	}
}

func TestMoveFileAnswersTheFailuresOfTheOperationsWithTheirCodes(t *testing.T) {
	for _, one := range getFailures() {
		calls := createFileCalls(getCaller(), &Access{Ops: &recordedOps{err: one.err}})

		res, err := calls.MoveFile(t.Context(), connect.NewRequest(&v1.HostFileServiceMoveFileRequest{From: "a", To: "b"}))

		if err != nil || res.Msg.GetError() != one.code {
			t.Errorf("a move that failed with %v answered %v (%v), want %v", one.err, res.Msg.GetError(), err, one.code)
		}
	}
}

func TestRemoveFileTrashesThroughTheOperationsAndAnswersTheFailuresWithTheirCodes(t *testing.T) {
	ops := &recordedOps{count: 3}
	res, err := createFileCalls(getCaller(), &Access{Ops: ops}).RemoveFile(t.Context(), connect.NewRequest(&v1.HostFileServiceRemoveFileRequest{Path: "Maps", Baseline: getBaseline()}))
	if err != nil || res.Msg.GetRemovedCount() != 3 || ops.path != "Maps" || ops.baseline == nil || ops.baseline.Hash != "h1" {
		t.Fatalf("the removal answered %v (%v) after %q over %+v", res.Msg, err, ops.path, ops.baseline)
	}
	for _, one := range getFailures() {
		calls := createFileCalls(getCaller(), &Access{Ops: &recordedOps{err: one.err}})

		res, err := calls.RemoveFile(t.Context(), connect.NewRequest(&v1.HostFileServiceRemoveFileRequest{Path: "a"}))

		if err != nil || res.Msg.GetError() != one.code {
			t.Errorf("a removal that failed with %v answered %v (%v), want %v", one.err, res.Msg.GetError(), err, one.code)
		}
	}
}

func TestPlaceFileWorksAsThePluginTheTokenNamesAndReturnsTheStatOfTheFile(t *testing.T) {
	ops := &recordedOps{}
	calls := createFileCalls(getCaller(), &Access{Ops: ops})

	res, err := calls.PlaceFile(t.Context(), connect.NewRequest(&v1.PlaceFileRequest{Name: "ab.part", Path: "Scans/Delta.pdf", ExpectedHash: "feed", Baseline: getBaseline()}))

	if err != nil || res.Msg.GetError() != 0 {
		t.Fatalf("the placement answered %v (%v)", res.Msg, err)
	}
	if ops.pluginID != "md.numen.syncer" || ops.name != "ab.part" || ops.path != "Scans/Delta.pdf" || ops.hash != "feed" || ops.baseline == nil {
		t.Errorf("the operations were asked %+v", ops)
	}
	if got := res.Msg.GetStat(); got.GetPath() != "Scans/Delta.pdf" || got.GetHash() != "feed" || got.GetSize() != 3 {
		t.Errorf("the placement returned the stat %v", got)
	}
}

func TestPlaceFileAnswersAHashThatDiffersWithHashMismatchAndTheOtherFailuresWithTheirCodes(t *testing.T) {
	failures := append(getFailures(), failure{fmt.Errorf("a: %w", plugin.ErrHashMismatch), v1.ErrorCode_ERROR_CODE_HASH_MISMATCH})
	for _, one := range failures {
		calls := createFileCalls(getCaller(), &Access{Ops: &recordedOps{err: one.err}})

		res, err := calls.PlaceFile(t.Context(), connect.NewRequest(&v1.PlaceFileRequest{Name: "a.part", Path: "a"}))

		if err != nil || res.Msg.GetError() != one.code || res.Msg.GetStat() != nil {
			t.Errorf("a placement that failed with %v answered %v (%v), want %v and no stat", one.err, res.Msg, err, one.code)
		}
	}
}

func TestPlaceFileWithoutAVaultAnswersNoVault(t *testing.T) {
	res, err := createFileCalls(getCaller(), nil).PlaceFile(t.Context(), connect.NewRequest(&v1.PlaceFileRequest{Name: "a.part", Path: "a"}))

	if err != nil || res.Msg.GetError() != v1.ErrorCode_ERROR_CODE_NO_VAULT {
		t.Errorf("the placement with no vault answered %v (%v)", res.Msg, err)
	}
}

func TestReadFileRangeReturnsTheBytesAndTheStatAndPassesTheRange(t *testing.T) {
	ranges := &recordedRanges{content: []byte("DELTA")}

	res, err := createFileCalls(getCaller(), &Access{Ranges: ranges}).ReadFileRange(t.Context(), connect.NewRequest(&v1.ReadFileRangeRequest{Path: "Maps/Delta.pdf", Start: 5, Length: 5}))

	if err != nil || res.Msg.GetError() != 0 || string(res.Msg.GetContent()) != "DELTA" || res.Msg.GetStat().GetHash() != "abc" {
		t.Fatalf("the read answered %v (%v)", res.Msg, err)
	}
	if ranges.path != "Maps/Delta.pdf" || ranges.start != 5 || ranges.length != 5 {
		t.Errorf("the range was %q from %d for %d", ranges.path, ranges.start, ranges.length)
	}
}

func TestReadFileRangeAnswersAFolderAndARangeOutsideTheLimitsWithTheirCodes(t *testing.T) {
	for _, one := range []failure{
		{fmt.Errorf("a: %w", fs.ErrNotExist), v1.ErrorCode_ERROR_CODE_NOT_FOUND},
		{fmt.Errorf("a: %w", vaultdomain.ErrOutside), v1.ErrorCode_ERROR_CODE_OUTSIDE_VAULT},
		{fmt.Errorf("a: %w", plugin.ErrInvalidRange), v1.ErrorCode_ERROR_CODE_INVALID_RANGE},
	} {
		calls := createFileCalls(getCaller(), &Access{Ranges: &recordedRanges{err: one.err}})

		res, err := calls.ReadFileRange(t.Context(), connect.NewRequest(&v1.ReadFileRangeRequest{Path: "Maps", Length: 1}))

		if err != nil || res.Msg.GetError() != one.code || len(res.Msg.GetContent()) != 0 {
			t.Errorf("a read that failed with %v answered %v (%v), want %v", one.err, res.Msg, err, one.code)
		}
	}
}

func TestAFailureOfTheContextIsNotAnErrorCode(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	calls := createFileCalls(getCaller(), &Access{Ops: &recordedOps{err: errors.New("stopped")}})

	_, err := calls.MoveFile(ctx, connect.NewRequest(&v1.HostFileServiceMoveFileRequest{From: "a", To: "b"}))

	if connect.CodeOf(err) != connect.CodeCanceled {
		t.Errorf("a move with a canceled context answered %v, want canceled", err)
	}
}
