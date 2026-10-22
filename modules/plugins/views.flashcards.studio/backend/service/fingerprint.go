package service

import (
	"time"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// stamp is when a file last changed, as the schema carries it. Zero is a file nothing was said
// about, not the epoch.
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

// writeFingerprint is what a file was, as the schema carries it. A file nothing is known of is
// carried as nothing.
func writeFingerprint(ref domain.Fingerprint) *v1.Fingerprint {
	if ref.IsZero() {
		return nil
	}
	return &v1.Fingerprint{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

// readFingerprint is what a client says a file was when it read it. Nothing said is nothing
// compared.
func readFingerprint(pos *v1.Fingerprint) domain.Fingerprint {
	if pos == nil {
		return domain.Fingerprint{}
	}
	return domain.Fingerprint{Path: pos.GetPath(), Size: pos.GetSize(), ModTime: instant(pos.GetMtime())}
}
