package hostendpoint

import (
	"time"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func writeFileStat(f domain.FileStat) *v1.HostFileStat {
	return &v1.HostFileStat{Path: f.Path, Size: f.Size, ModTime: f.ModTime.UnixNano(), IsNote: f.Kind == domain.KindNote}
}

func readFileStat(f *v1.HostFileStat) domain.FileStat {
	kind := domain.SourceKind("")
	switch {
	case f.GetIsNote():
		kind = domain.KindNote
	case f.GetPath() != "":
		kind = domain.KindFile
	}
	return domain.FileStat{Path: f.GetPath(), Kind: kind, Size: f.GetSize(), ModTime: time.Unix(0, f.GetModTime())}
}
