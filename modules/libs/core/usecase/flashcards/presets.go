package flashcards

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"unicode/utf8"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	history "github.com/jiva-studio/numen/modules/libs/core/flashcards"
	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// LinkType is what a deck's link to its preset carries under `type`.
const LinkType = "preset"

// Preset is one preset as a read hands it over.
type Preset struct {
	// Path is the note the settings were read from, and is empty for a deck
	// naming no preset.
	Path    string
	Outcome note.Outcome
	// Type is what the note at the path says it is, so a caller handed an
	// ordinary note is told so.
	Type domain.NoteType
	// Preset is how the decks pointing here are scheduled. It stands at the
	// defaults for every outcome but Ok.
	Preset history.Preset
	// Problems are what was wrong in the file and was not guessed at. They are
	// shown against the preset, and the editor is where they are settled.
	Problems []string
	Ref      domain.FileRef
}

// Presets is how each deck of a vault is scheduled.
//
// A deck names its preset with an entry of its `links:` block carrying
// `type: preset`. A deck naming none is scheduled by the defaults, which is
// what a vault holding no preset at all gets.
type Presets struct {
	Readers port.VaultReaders
	// Links answers where a deck's link to its preset lands.
	Links port.LinkQueries
}

// Default is a deck scheduled by no preset.
func Default() Preset {
	return Preset{Outcome: note.Ok, Preset: history.Defaults()}
}

// Of is the preset the deck at path is scheduled by.
//
// A link reaching nothing, and a link reaching a note that is not a preset,
// leave the deck on the defaults and say so against it. A deck naming two
// presets is scheduled by the first and carries a problem: two presets are two
// answers to one question.
func (u Presets) Of(ctx context.Context, v domain.Vault, deck string) (Preset, error) {
	if u.Links == nil {
		return Default(), nil
	}
	links, err := u.Links.Links(ctx, v.ID, deck)
	if err != nil {
		return Preset{}, fmt.Errorf("the links of %s: %w", deck, err)
	}

	var at []string
	for _, link := range links {
		if link.Type == LinkType && link.To != "" {
			at = append(at, link.To)
		}
	}
	if len(at) == 0 {
		return Default(), nil
	}

	out, err := u.Read(ctx, v, at[0])
	if err != nil {
		return Preset{}, err
	}
	if len(at) > 1 {
		out.Problems = append(out.Problems, "this deck names more than one preset, and is scheduled by "+at[0])
	}
	return out, nil
}

// Read reads the preset at path.
//
// An error is the vault being out of reach. What is wrong with the note itself
// is an outcome or a problem, and the preset stands at the defaults.
func (u Presets) Read(ctx context.Context, v domain.Vault, path string) (Preset, error) {
	out := Preset{Path: path, Preset: history.Defaults()}

	reader, err := u.Readers.Open(v)
	if err != nil {
		return Preset{}, err
	}

	// The vault says what is at a path without opening it: a note, a file it
	// leaves alone, or nothing.
	ref, err := reader.Stat(ctx, path)
	held := err == nil
	switch {
	case held:
		out.Ref = ref
		if ref.Kind != domain.KindNote {
			out.Outcome = note.NotANote
			return out, nil
		}
		if ref.Size > note.MaxBytes {
			out.Outcome = note.TooLarge
			return out, nil
		}
	case errors.Is(err, port.ErrNotANote):
		out.Outcome = note.NotANote
		return out, nil
	case !errors.Is(err, fs.ErrNotExist):
		return Preset{}, fmt.Errorf("look at %s: %w", path, err)
	}

	raw, err := reader.Read(ctx, path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		out.Outcome = note.Missing
		return out, nil
	case !held && err == nil:
		out.Outcome = note.NotANote
		return out, nil
	case err != nil:
		return Preset{}, fmt.Errorf("read %s: %w", path, err)
	}

	if !utf8.Valid(raw) {
		out.Outcome = note.NotText
		return out, nil
	}
	if _, err := markdown.Open(raw); err != nil {
		out.Outcome = note.Unreadable
		return out, nil
	}

	n := markdown.Parse(ref, raw)
	out.Outcome, out.Type = note.Ok, n.Type
	if n.Type != domain.TypePreset {
		out.Problems = append(out.Problems, path+" is not a preset, and the defaults stand")
		return out, nil
	}
	out.Preset, out.Problems = history.ReadPreset(n.Frontmatter)
	return out, nil
}
