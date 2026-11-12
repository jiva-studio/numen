package filewire

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

func TestStampAndInstantRoundTripZeroAndNanoseconds(t *testing.T) {
	if got := stamp(time.Time{}); got != 0 {
		t.Fatalf("expected 0 for zero time stamp, got %d", got)
	}
	if got := stamp(time.Unix(0, 789)); got != 789 {
		t.Fatalf("expected 789 for stamp, got %d", got)
	}
	if !instant(0).IsZero() {
		t.Fatalf("expected zero time for instant(0), got %v", instant(0))
	}
	if got := instant(-1).UnixNano(); got != -1 {
		t.Fatalf("expected -1 for instant(-1), got %d", got)
	}
	if got := instant(1).UnixNano(); got != 1 {
		t.Fatalf("expected 1 for instant(1), got %d", got)
	}
}

func TestStatOfAFileWithNoneIsAbsent(t *testing.T) {
	if got := NewStat(domain.FileStat{}); got != nil {
		t.Fatalf("expected nil for zero FileStat, got %+v", got)
	}
	if got := GetStat(nil); got != (domain.FileStat{}) {
		t.Fatalf("expected zero FileStat for nil, got %+v", got)
	}
}

func TestConvertWriteReportsALandedWriteWithItsStat(t *testing.T) {
	pos := domain.FileStat{Path: "D.md", Size: 3, ModTime: time.Unix(0, 5)}
	out, err := ConvertWrite(pos, nil)
	if err != nil || out.Error != api.ErrorCodeUnspecified || out.At == nil || out.At.Mtime != 5 || out.IsIndexOutdated {
		t.Fatalf("unexpected outcome of a landed write: %+v, %v", out, err)
	}
	out, err = ConvertWrite(pos, edit.ErrIndexOutdated)
	if err != nil || out.At == nil || !out.IsIndexOutdated {
		t.Fatalf("unexpected outcome of a write the index did not follow: %+v, %v", out, err)
	}
}

func TestConvertWriteAnswersAFailureOnTheNoteWithItsCode(t *testing.T) {
	out, err := ConvertWrite(domain.FileStat{}, ports.ErrStale)
	if err != nil || out.Error != api.ErrorCodeConflict || out.At != nil {
		t.Fatalf("unexpected outcome of a stale write: %+v, %v", out, err)
	}
	custom := errors.New("not a preset")
	out, err = ConvertWrite(domain.FileStat{}, custom, Reason{Err: custom, Code: api.ErrorCodeNotAPreset})
	if err != nil || out.Error != api.ErrorCodeNotAPreset {
		t.Fatalf("unexpected outcome of a given reason: %+v, %v", out, err)
	}
}

func TestConvertWriteReturnsAnyOtherFailureAsAnAPIError(t *testing.T) {
	_, err := ConvertWrite(domain.FileStat{}, errors.New("disk boom"))
	var apiErr *server.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != server.CodeInternal {
		t.Fatalf("expected an internal API error, got %v", err)
	}
}
