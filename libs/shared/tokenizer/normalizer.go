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
		var settings struct {
			Normalizers []step `json:"normalizers"`
		}
		if err := s.decode(&settings); err != nil {
			return nil, fmt.Errorf("Sequence: %w", err)
		}
		var out []normalizer
		for _, one := range settings.Normalizers {
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
			return nil, fmt.Errorf("Precompiled: %w", err)
		}
		return []normalizer{one}, nil
	case "Strip":
		one, err := buildStrip(s)
		if err != nil {
			return nil, fmt.Errorf("Strip: %w", err)
		}
		return []normalizer{one}, nil
	case "Replace":
		one, err := buildReplace(s)
		if err != nil {
			return nil, fmt.Errorf("Replace: %w", err)
		}
		return []normalizer{one}, nil
	default:
		return nil, fmt.Errorf("%q is not supported", s.Type)
	}
}

func buildPrecompiled(s step) (normalizer, error) {
	var settings struct {
		Charsmap string `json:"precompiled_charsmap"`
	}
	if err := s.decode(&settings); err != nil {
		return nil, err
	}
	charsmap, err := parseCharsmap(settings.Charsmap)
	if err != nil {
		return nil, err
	}
	return charsmap.normalize, nil
}

func buildStrip(s step) (normalizer, error) {
	var settings struct {
		Left  bool `json:"strip_left"`
		Right bool `json:"strip_right"`
	}
	if err := s.decode(&settings); err != nil {
		return nil, err
	}
	return func(text string) string {
		if settings.Left {
			text = strings.TrimLeftFunc(text, unicode.IsSpace)
		}
		if settings.Right {
			text = strings.TrimRightFunc(text, unicode.IsSpace)
		}
		return text
	}, nil
}

func buildReplace(s step) (normalizer, error) {
	var settings struct {
		Pattern struct {
			String *string `json:"String"`
			Regex  *string `json:"Regex"`
		} `json:"pattern"`
		Content string `json:"content"`
	}
	if err := s.decode(&settings); err != nil {
		return nil, err
	}
	content := settings.Content
	switch {
	case settings.Pattern.Regex != nil:
		expression, err := regexp.Compile(*settings.Pattern.Regex)
		if err != nil {
			return nil, err
		}
		return func(text string) string { return expression.ReplaceAllLiteralString(text, content) }, nil
	case settings.Pattern.String != nil:
		literal := *settings.Pattern.String
		if literal == "" {
			return nil, errors.New("the String pattern is empty")
		}
		return func(text string) string { return strings.ReplaceAll(text, literal, content) }, nil
	default:
		return nil, errors.New("the pattern is neither a String nor a Regex")
	}
}
