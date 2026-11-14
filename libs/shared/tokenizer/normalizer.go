package tokenizer

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// normalizer rewrites the whole text before it is split into words.
type normalizer func(string) string

func buildNormalizers(s step) ([]normalizer, error) {
	switch s.Type {
	case "Sequence":
		var normalizers []step
		if err := s.fields.read("normalizers", &normalizers); err != nil {
			return nil, fmt.Errorf("step Sequence: %w", err)
		}
		var out []normalizer
		for _, one := range normalizers {
			inner, err := buildNormalizers(one)
			if err != nil {
				return nil, err
			}
			out = append(out, inner...)
		}
		return out, nil
	case "Precompiled":
		one, err := buildPrecompiled(s)
		if err != nil {
			return nil, fmt.Errorf("step Precompiled: %w", err)
		}
		return []normalizer{one}, nil
	case "Strip":
		one, err := buildStrip(s)
		if err != nil {
			return nil, fmt.Errorf("step Strip: %w", err)
		}
		return []normalizer{one}, nil
	case "Replace":
		one, err := buildReplace(s)
		if err != nil {
			return nil, fmt.Errorf("step Replace: %w", err)
		}
		return []normalizer{one}, nil
	default:
		return nil, fmt.Errorf("step %q is not supported", s.Type)
	}
}

func buildPrecompiled(s step) (normalizer, error) {
	var encoded string
	if err := s.fields.read("precompiled_charsmap", &encoded); err != nil {
		return nil, err
	}
	charsmap, err := parseCharsmap(encoded)
	if err != nil {
		return nil, err
	}
	return charsmap.normalize, nil
}

func buildStrip(s step) (normalizer, error) {
	var isLeft, isRight bool
	if err := s.fields.read("strip_left", &isLeft); err != nil {
		return nil, err
	}
	if err := s.fields.read("strip_right", &isRight); err != nil {
		return nil, err
	}
	return func(text string) string {
		if isLeft {
			text = strings.TrimLeftFunc(text, unicode.IsSpace)
		}
		if isRight {
			text = strings.TrimRightFunc(text, unicode.IsSpace)
		}
		return text
	}, nil
}

func buildReplace(s step) (normalizer, error) {
	var pattern jsonObject
	var content string
	var literal, expression *string
	if err := s.fields.read("pattern", &pattern); err != nil {
		return nil, err
	}
	if err := s.fields.read("content", &content); err != nil {
		return nil, err
	}
	if err := pattern.read("String", &literal); err != nil {
		return nil, err
	}
	if err := pattern.read("Regex", &expression); err != nil {
		return nil, err
	}
	switch {
	case expression != nil:
		compiled, err := regexp.Compile(*expression)
		if err != nil {
			return nil, err
		}
		return func(text string) string { return compiled.ReplaceAllLiteralString(text, content) }, nil
	case literal != nil:
		old := *literal
		if old == "" {
			return nil, errors.New("the String pattern is empty")
		}
		return func(text string) string { return strings.ReplaceAll(text, old, content) }, nil
	default:
		return nil, errors.New("the pattern is neither a String nor a Regex")
	}
}
