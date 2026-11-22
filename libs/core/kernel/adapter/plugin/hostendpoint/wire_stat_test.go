package hostendpoint

import (
	"time"

	v1 "github.com/jiva-studio/numen/libs/shared/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func writeFileStat(f domain.FileStat) *v1.FileStat {
	return &v1.FileStat{Path: f.Path, Size: f.Size, Mtime: f.ModTime.UnixNano(), IsNote: f.Kind == domain.KindNote}
}

func readFileStat(f *v1.FileStat) domain.FileStat {
	kind := domain.SourceKind("")
	switch {
	case f.GetIsNote():
		kind = domain.KindNote
	case f.GetPath() != "":
		kind = domain.KindFile
	}
	return domain.FileStat{Path: f.GetPath(), Kind: kind, Size: f.GetSize(), ModTime: time.Unix(0, f.GetMtime())}
}
