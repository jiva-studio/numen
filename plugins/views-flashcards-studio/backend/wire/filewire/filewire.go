// Package filewire converts the file stat of a note, and the end of a write to a note, to what a
// response carries.
package filewire

import (
	"errors"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/wire/apierror"
)

func stamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

func instant(nanos int64) time.Time {
	if nanos == 0 {
		return time.Time{}
	}
	return time.Unix(0, nanos)
}

// NewStat is the file stat a client presents at its next save, and nil for a file that has none.
func NewStat(ref domain.FileStat) *api.FileStat {
	if ref.IsZero() {
		return nil
	}
	return &api.FileStat{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

// GetStat is the file stat a client presented.
func GetStat(pos *api.FileStat) domain.FileStat {
	if pos == nil {
		// gomutants:disable-next-line RETURN_ZERO reason="Explicit zero-field struct literal equals the zero value."
		return domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}
	}
	return domain.FileStat{Path: pos.Path, Kind: "", Size: pos.Size, ModTime: instant(pos.Mtime)}
}

// Reason is a failure particular to one write, and the ErrorCode it is answered with.
type Reason struct {
	Err  error
	Code api.ErrorCode
}

// Outcome is the fields every response to a write of a note carries.
type Outcome struct {
	Error           api.ErrorCode
	At              *api.FileStat
	IsIndexOutdated bool
}

// ConvertWrite reads the end of a write. A write that reached the vault has happened, so it carries
// the file stat the client presents at its next save and reports where the index did not follow. A
// failure on the note is an ErrorCode, the reasons given first; any other failure is returned as an API error.
func ConvertWrite(pos domain.FileStat, err error, reasons ...Reason) (Outcome, error) {
	outdated := apierror.IsIndexOutdated(err)
	if err == nil || outdated {
		return Outcome{Error: api.ErrorCodeUnspecified, At: NewStat(pos), IsIndexOutdated: outdated}, nil
	}
	for _, reason := range reasons {
		if errors.Is(err, reason.Err) {
			return Outcome{Error: reason.Code, At: nil, IsIndexOutdated: false}, nil
		}
	}
	code, ok := apierror.GetErrorCode(err)
	if !ok {
		return Outcome{Error: api.ErrorCodeUnspecified, At: nil, IsIndexOutdated: false}, apierror.NewError(apierror.GetCode(err), err)
	}
	return Outcome{Error: code, At: nil, IsIndexOutdated: false}, nil
}
