package source

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/text"
)

// DropTranscript takes away everything listening to a recording produced.
//
// The words a model heard, the words a person put right, the record of what
// listened and the answer a recording with no speech gave all go, and the
// chunks cut from those words go with them. The recording stands as it did
// before anything listened to it, and the next run hears it from the start.
type DropTranscript struct {
	Readers port.VaultReaders
	Sources port.SourceRepository
	Owing   port.SourceQueries
	Derived port.DerivedStores

	// Forgets takes a recording out of what a queue has already had an answer
	// about, so one that gave no words is offered again. A build without one
	// leaves the queue as it stands.
	Forgets func(v domain.Vault, path string)

	// Area is the store the artifact is kept in. Empty means the default.
	Area string
}

// Dropped reports what dropping a transcript did.
type Dropped struct {
	Path string // the recording whose transcript was dropped
	None bool   // nothing has listened to it, and nothing was done
	Busy bool   // somebody is listening to it, and nothing was done
}

// Execute drops the transcript of one recording.
func (u DropTranscript) Execute(ctx context.Context, v domain.Vault, path string) (Dropped, error) {
	res := Dropped{Path: path}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.Stat(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return res, err
	}

	from, hash, stood, err := u.produced(ctx, v, reader, path)
	if err != nil {
		return res, err
	}

	// A run holds the recording it is listening to for as long as it takes, by
	// the name it appends to. Nothing is taken away underneath it.
	partial := text.Partial(from, hash)
	release, err := store.Claim(ctx, partial)
	if errors.Is(err, port.ErrClaimed) {
		res.Busy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	// A store the person emptied leaves the index standing on a reading, and
	// those rows are this use case's to take away as well.
	names := text.Names(from, hash)
	held, err := u.kept(ctx, store, names)
	if err != nil {
		return res, err
	}
	if !held && !stood {
		res.None = true
		return res, nil
	}

	for _, name := range names {
		if name == partial {
			continue
		}
		if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return res, fmt.Errorf("remove %s: %w", name, err)
		}
	}

	// The source is recorded as it was walked: no fingerprint, no recipe, no
	// producer, and no chunks. What the index knew about the words goes in the
	// one write that says the recording owes its text again.
	if err := u.Sources.SaveExtraction(ctx, v.ID, port.Extraction{Source: port.Source{Ref: ref}}); err != nil {
		return res, fmt.Errorf("record %s: %w", path, err)
	}
	if u.Forgets != nil {
		u.Forgets(v, ref.Path)
	}

	// The claim is a lock on the file the name is claimed by, so that file goes
	// last: the name is free from the moment it does.
	return res, store.Remove(ctx, partial)
}

// produced is the producer and the fingerprint a recording's transcript is kept
// under, and whether the index says the recording stands on one.
//
// A recording that gave no words is recorded as standing on nothing, and what
// the run wrote is kept under the fingerprint of the bytes.
func (u DropTranscript) produced(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	path string,
) (from, hash string, stood bool, err error) {
	said, held, err := u.Owing.Reading(ctx, v.ID, path)
	if err != nil {
		return "", "", false, fmt.Errorf("read index: %w", err)
	}
	if held && said.From != "" {
		return said.From, said.Hash, true, nil
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", "", false, fmt.Errorf("read %s: %w", path, err)
	}
	return u.area(), text.Fingerprint(raw), false, nil
}

// kept says whether the store has something under any of these names.
func (u DropTranscript) kept(ctx context.Context, store port.DerivedStore, names []string) (bool, error) {
	for _, name := range names {
		switch _, err := store.Read(ctx, name); {
		case err == nil:
			return true, nil
		case !errors.Is(err, fs.ErrNotExist):
			return false, err
		}
	}
	return false, nil
}

func (u DropTranscript) area() string {
	if u.Area == "" {
		return text.ASR
	}
	return u.Area
}
