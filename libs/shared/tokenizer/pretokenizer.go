package tokenizer

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// preTokenizer splits the normalized text into the pieces Viterbi runs over; no piece spans two of them.
type preTokenizer func(pieces []string) []string

func buildPreTokenizer(s step) (preTokenizer, error) {
	switch s.Type {
	case "Sequence":
		var settings struct {
			PreTokenizers []step `json:"pretokenizers"`
		}
		if err := s.decode(&settings); err != nil {
			return nil, fmt.Errorf("Sequence: %w", err)
		}
		chain := make([]preTokenizer, 0, len(settings.PreTokenizers))
		for _, one := range settings.PreTokenizers {
			inner, err := buildPreTokenizer(one)
			if err != nil {
				return nil, err
			}
			chain = append(chain, inner)
		}
		return func(pieces []string) []string {
			for _, split := range chain {
				pieces = split(pieces)
			}
			return pieces
		}, nil
	case "WhitespaceSplit":
		return splitWhitespace, nil
	case "Metaspace":
		split, err := buildMetaspace(s)
		if err != nil {
			return nil, fmt.Errorf("Metaspace: %w", err)
		}
		return split, nil
	default:
		return nil, fmt.Errorf("%q is not supported", s.Type)
	}
}

func splitWhitespace(pieces []string) []string {
	var out []string
	for _, piece := range pieces {
		out = append(out, strings.FieldsFunc(piece, unicode.IsSpace)...)
	}
	return out
}

func buildMetaspace(s step) (preTokenizer, error) {
	var settings struct {
		Replacement    string  `json:"replacement"`
		PrependScheme  *string `json:"prepend_scheme"`
		AddPrefixSpace *bool   `json:"add_prefix_space"`
		Split          *bool   `json:"split"`
	}
	if err := s.decode(&settings); err != nil {
		return nil, err
	}
	if utf8.RuneCountInString(settings.Replacement) != 1 {
		return nil, fmt.Errorf("the replacement %q is not one character", settings.Replacement)
	}
	switch {
	case settings.PrependScheme != nil:
		if *settings.PrependScheme != "always" {
			return nil, fmt.Errorf("prepend_scheme %q is not supported, only \"always\"", *settings.PrependScheme)
		}
	case settings.AddPrefixSpace == nil || !*settings.AddPrefixSpace:
		return nil, errors.New("only add_prefix_space true or prepend_scheme \"always\" is supported")
	}
	if settings.Split != nil && !*settings.Split {
		return nil, errors.New("split false is not supported")
	}
	replacement := settings.Replacement
	return func(pieces []string) []string {
		var out []string
		for _, piece := range pieces {
			out = splitMetaspace(out, piece, replacement)
		}
		return out
	}, nil
}

func splitMetaspace(out []string, piece, replacement string) []string {
	if piece == "" {
		return out
	}
	piece = strings.ReplaceAll(piece, " ", replacement)
	if !strings.HasPrefix(piece, replacement) {
		piece = replacement + piece
	}
	for len(piece) > 0 {
		next := strings.Index(piece[len(replacement):], replacement)
		if next < 0 {
			return append(out, piece)
		}
		cut := len(replacement) + next
		out = append(out, piece[:cut])
		piece = piece[cut:]
	}
	return out
}
