package document

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Fingerprint is which bytes a document is: where it sits in the vault, and
// what the vault says about the file there. The stamp is nanoseconds since the
// epoch, and zero is a file nothing was said about.
type Fingerprint struct {
	Path  string
	Size  int64
	Mtime int64
}

// ErrChanged is a file that is no longer the one an address was given out for.
var ErrChanged = errors.New("the file changed since this address was given out")

// getFingerprint is what the vault says about the file at a path, with the
// reader that says it.
func getFingerprint(
	ctx context.Context, readers port.VaultReaders, v domain.Vault, path string,
) (port.VaultReader, Fingerprint, error) {
	reader, err := readers.Open(v)
	if err != nil {
		return nil, Fingerprint{}, err
	}
	ref, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return nil, Fingerprint{}, err
	}
	return reader, Fingerprint{Path: ref.Path, Size: ref.Size, Mtime: getStamp(ref.ModTime)}, nil
}

func getStamp(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixNano()
}

// checkUnchanged refuses a file that is not the bytes the address named.
func checkUnchanged(named, found Fingerprint) error {
	if named.Size != found.Size || named.Mtime != found.Mtime {
		return ErrChanged
	}
	return nil
}

// getDomainFingerprint is the fingerprint as the core states it.
func getDomainFingerprint(mark Fingerprint) domain.Fingerprint {
	out := domain.Fingerprint{Path: mark.Path, Size: mark.Size}
	if mark.Mtime != 0 {
		out.ModTime = time.Unix(0, mark.Mtime)
	}
	return out
}
