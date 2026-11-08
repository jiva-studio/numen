package service

import (
	"time"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/api"
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

func writeFileStat(ref domain.FileStat) *api.FileStat {
	if ref.IsZero() {
		return nil
	}
	return &api.FileStat{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

func readFileStat(pos *api.FileStat) domain.FileStat {
	if pos == nil {
		// gomutants:disable-next-line RETURN_ZERO reason="Explicit zero-field struct literal equals the zero value."
		return domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}}
	}
	return domain.FileStat{Path: pos.Path, Kind: "", Size: pos.Size, ModTime: instant(pos.Mtime)}
}
