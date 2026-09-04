package webui

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	derived "github.com/jiva-studio/numen/modules/libs/core/text"
)

// A source carrying no text a person typed is put through a run: a scan is
// read, a recording is heard. How far one has got is read off the store: what
// a run writes and what it holds while it writes are both names in it.
//
// A source a person named is never turned away for want of a turn. It waits in
// line and is taken up before anything the vault set itself.

// A Run reads a source a model has to produce the text of. Reading a scan and
// hearing a recording are one shape here, and one at a time of each.
type Run interface {
	// Start takes on one source a person named, and says whether it began now
	// or waits its turn. It runs under the application, so whoever asked is
	// answered at once.
	Start(v domain.Vault, path string) port.Taking
}

// errNoReading is a build with nothing to read a scan with, and errNoListening
// one with nothing to hear a recording with.
var (
	errNoReading   = errors.New("this build cannot read a scan")
	errNoListening = errors.New("this build cannot hear a recording")
)

// reached is how far a run over one source has got: done is the whole of the
// text a model produced already standing, under is a run holding this very
// source now, and stopped is part of the text on disk with no run behind it.
//
// answer is what a run got out of a source it got no words out of, and why is
// what it wrote about it. size is how many bytes stand under whichever name
// the run has reached, and is what tells a caller that what it read has moved
// on.
type reached struct {
	done    bool
	under   bool
	stopped bool
	answer  string
	why     string
	size    int
}

// far says how far a run over the source at a path has got.
//
// A build that cannot say which model produced a text answers nothing, and the
// run itself then decides what is left to do.
func (a *API) far(
	ctx context.Context,
	v domain.Vault,
	path string,
	kind domain.SourceKind,
) (reached, error) {
	said, store, produced, err := a.heard(ctx, v, path)
	if err != nil {
		return reached{}, err
	}
	if produced {
		return farUnder(ctx, store, said.Producer, said.Hash)
	}
	return a.byBytes(ctx, v, path, unnamed(kind))
}

// farUnder says how far the run keeping its files under a name has got.
//
// What a run reaches is written down as it goes and the whole of it is written
// under its own name at the end, so a source stands on the text once that name
// is there. It holds the name it is still writing under for as long as it
// takes, so a claim on that name that is refused is a run over this source: the
// claim is taken and given straight back, and whether it was refused is the
// answer.
func farUnder(ctx context.Context, store port.DerivedStore, from, hash string) (reached, error) {
	var got reached
	whole, err := store.Read(ctx, derived.Artifact(from, hash))
	switch {
	case err == nil:
		got.done, got.size = true, len(whole)
		return got, nil
	case !errors.Is(err, fs.ErrNotExist):
		return got, err
	}
	// A run that got no words out of a source wrote down what it got instead,
	// and asking again gets the same. Taking that record away is how a person
	// asks for the source to be tried afresh.
	switch held, err := store.Read(ctx, derived.Answer(from, hash)); {
	case err == nil:
		got.answer, got.why = derived.Answered(held)
		return got, nil
	case !errors.Is(err, fs.ErrNotExist):
		return got, err
	}

	// How far a run got is on disk under a name of its own, and the run holds
	// that name while it writes. Read before the claim is tried, so what is
	// answered is a run's own file and not one the claim itself made.
	part, err := store.Read(ctx, derived.Partial(from, hash))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return got, err
	}
	got.size = len(part)
	got.under = !free(ctx, store, derived.Partial(from, hash))
	// A run that stopped left what it reached behind it. It is not a source
	// nothing has touched, and asking again begins afresh.
	got.stopped = !got.under && got.size > 0
	return got, nil
}

// byBytes says how far a run over a source the index names no producer for has
// got. A source a run got no words out of is one of those, and what the run
// wrote is kept under the fingerprint of the bytes.
//
// The file is read and fingerprinted here, which is what a run does before
// anything else.
func (a *API) byBytes(ctx context.Context, v domain.Vault, path, from string) (reached, error) {
	_, stores, ok := a.hearing()
	if !ok || from == "" {
		return reached{}, nil
	}
	reader, err := a.Readers.Open(v)
	if err != nil {
		return reached{}, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return reached{}, err
	}
	store, err := stores.Open(v)
	if err != nil {
		return reached{}, err
	}
	return farUnder(ctx, store, from, derived.Fingerprint(raw))
}

// unnamed is the producer whose files stand for a source the index names none
// for: a recording is listened to. Nothing produces a scan's text without the
// index saying what did.
func unnamed(kind domain.SourceKind) string {
	if kind == domain.KindRecording {
		return derived.ASR
	}
	return ""
}
