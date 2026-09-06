package mcp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// What a note, a deck and the note a link is written in were when they were
// read. Three subjects hand it back and forth, so the form it takes on the wire
// is nobody's alone.

// fingerprintOf is what a note was when it was read, in a form an agent hands
// back without having to understand it.
func fingerprintOf(ref domain.Fingerprint) string {
	return strconv.FormatInt(ref.Size, 10) + "-" + strconv.FormatInt(stamp(ref.ModTime), 10)
}

// parseFingerprint is the fingerprint a caller presents. A tool that changes
// what somebody may have read since takes one, and a call carrying none is
// refused.
func parseFingerprint(s string) (domain.Fingerprint, error) {
	if s == "" {
		return domain.Fingerprint{}, errors.New(
			"present the fingerprint the read gave you: note_read for a note, card_read for a deck")
	}
	size, mtime, found := strings.Cut(s, "-")
	if !found {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a fingerprint note_read gave out", s)
	}
	ref := domain.Fingerprint{}
	var err error
	if ref.Size, err = strconv.ParseInt(size, 10, 64); err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a fingerprint note_read gave out", s)
	}
	nanos, err := strconv.ParseInt(mtime, 10, 64)
	if err != nil {
		return domain.Fingerprint{}, fmt.Errorf("%q is not a fingerprint note_read gave out", s)
	}
	ref.ModTime = instant(nanos)
	return ref, nil
}

// stamp and instant are a modification time as an agent hands it back and forth
// — nanoseconds since the epoch — and as the core holds one. Zero is a file
// nothing was said about, not the epoch.

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
