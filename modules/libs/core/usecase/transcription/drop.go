package transcription

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// Drop takes away everything listening to a recording produced.
//
// The words a model heard, the words a person put right, the record of what
// listened and the answer a recording with no speech gave all go, and the
// chunks cut from those words go with them. The recording stands as it did
// before anything listened to it, and the next run hears it from the start.
type Drop struct {
	Readers port.VaultReaders
	Sources port.SourceRepository
	Queries port.SourceQueries
	Derived port.DerivedStores

	// Forgets takes a recording out of what a queue has already had an answer
	// about, so one that gave no words is offered again. A build without one
	// leaves the queue as it stands.
	Forgets func(v domain.Vault, path string)

	// Area is the store the artifact is kept in. Empty means the default.
	Area string
}

// DropResult reports what dropping a transcript did.
type DropResult struct {
	Path   string // the recording whose transcript was dropped
	IsNone bool   // nothing has listened to it, and nothing was done
	IsBusy bool   // somebody is listening to it, and nothing was done
}

// Execute drops the transcript of one recording.
func (u Drop) Execute(ctx context.Context, v domain.Vault, path string) (DropResult, error) {
	res := DropResult{Path: path}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return res, err
	}
	ref, err := reader.GetFingerprint(ctx, path)
	if err != nil {
		return res, fmt.Errorf("stat %s: %w", path, err)
	}
	store, err := u.Derived.Open(v)
	if err != nil {
		return res, err
	}

	from, hash, stood, err := u.getProducer(ctx, v, reader, path)
	if err != nil {
		return res, err
	}

	// A run holds the recording it is listening to for as long as it takes, by
	// the name it appends to. Nothing is taken away underneath it.
	partial := text.Partial(from, hash)
	release, err := store.ClaimName(ctx, partial)
	if errors.Is(err, port.ErrClaimed) {
		res.IsBusy = true
		return res, nil
	}
	if err != nil {
		return res, err
	}
	defer release()

	// A store the person emptied leaves the index standing on a recognition, and
	// those rows are this use case's to take away as well.
	names := text.Names(from, hash)
	held, err := u.hasAnyName(ctx, store, names)
	if err != nil {
		return res, err
	}
	if !held && !stood {
		res.IsNone = true
		return res, nil
	}

	if err := removeNamesExcept(ctx, store, names, partial); err != nil {
		return res, err
	}

	// The source is recorded as it was walked: no fingerprint, no recipe, no
	// producer, and no chunks. What the index knew about the words goes in the
	// one write that says the recording owes its text again.
	if err := u.Sources.SaveExtraction(ctx, v.ID, domain.SourceChunks{Source: domain.Source{Fingerprint: ref}}); err != nil {
		return res, fmt.Errorf("record %s: %w", path, err)
	}
	if u.Forgets != nil {
		u.Forgets(v, ref.Path)
	}

	// The claim is a lock on the file the name is claimed by, so that file goes
	// last: the name is free from the moment it does.
	return res, store.Remove(ctx, partial)
}

// getProducer is the producer and the fingerprint a recording's transcript is
// kept under, and whether the index says the recording stands on one.
//
// A recording that gave no words is recorded as standing on nothing, and what
// the run wrote is kept under the fingerprint of the bytes.
func (u Drop) getProducer(
	ctx context.Context,
	v domain.Vault,
	reader port.VaultReader,
	path string,
) (from, hash string, stood bool, err error) {
	said, held, err := u.Queries.GetSourceText(ctx, v.ID, path)
	if err != nil {
		return "", "", false, fmt.Errorf("read index: %w", err)
	}
	if held && said.Producer != "" {
		return said.Producer, said.Hash, true, nil
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", "", false, fmt.Errorf("read %s: %w", path, err)
	}
	return u.area(), text.Fingerprint(raw), false, nil
}

// hasAnyName says whether the store has something under any of these names.
func (u Drop) hasAnyName(ctx context.Context, store port.DerivedStore, names []string) (bool, error) {
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

// removeNamesExcept removes every name from the store but keep, and passes over
// names that are already absent.
func removeNamesExcept(ctx context.Context, store port.DerivedStore, names []string, keep string) error {
	for _, name := range names {
		if name == keep {
			continue
		}
		if err := store.Remove(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", name, err)
		}
	}
	return nil
}

func (u Drop) area() string {
	if u.Area == "" {
		return text.ASR
	}
	return u.Area
}
