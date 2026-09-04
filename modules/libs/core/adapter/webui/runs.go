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

// A Runner reads a source a model has to produce the text of. Reading a scan
// and hearing a recording are one shape here, and one at a time of each.
type Runner interface {
	// Start takes on one source a person named, and says whether it began now
	// or waits its turn. It runs under the application, so whoever asked is
	// answered at once.
	Start(v domain.Vault, path string) port.StartOutcome
}

// errNoReading is a build with nothing to read a scan with, and errNoListening
// one with nothing to hear a recording with.
var (
	errNoReading   = errors.New("this build cannot read a scan")
	errNoListening = errors.New("this build cannot hear a recording")
)

// reached is how far a run over one source has got: the one thing a source
// stands at, what the run wrote about a source it got no words out of, and how
// many bytes stand under whichever name it has reached, which is what tells a
// caller that what it read has moved on.
type reached struct {
	stands stand
	why    string
	size   int
}

// A stand is what a source stands at. One holds at a time: a source is not both
// being read and read through.
type stand int

const (
	// untouched is a source no run has left anything of.
	untouched stand = iota
	// done is the whole of the text a model produced already standing.
	done
	// under is a run holding this very source now.
	under
	// stopped is part of the text on disk with no run behind it.
	stopped
	// silent is a run that got no words out of it because it holds no speech,
	// and unopened one that got none because nothing here opens the bytes.
	silent
	unopened
)

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
		got.stands, got.size = done, len(whole)
		return got, nil
	case !errors.Is(err, fs.ErrNotExist):
		return got, err
	}
	// A run that got no words out of a source wrote down what it got instead,
	// and asking again gets the same. Taking that record away is how a person
	// asks for the source to be tried afresh.
	switch held, err := store.Read(ctx, derived.Answer(from, hash)); {
	case err == nil:
		switch answer, why := derived.Answered(held); answer {
		case derived.Silent:
			got.stands = silent
		case derived.Unopened:
			got.stands, got.why = unopened, why
		}
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
	switch {
	case !free(ctx, store, derived.Partial(from, hash)):
		got.stands = under
	// A run that stopped left what it reached behind it. It is not a source
	// nothing has touched, and asking again begins afresh.
	case got.size > 0:
		got.stands = stopped
	}
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
