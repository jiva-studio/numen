package tokenizer

import (
	"errors"
	"fmt"
)

// templatePart is one entry of a TemplateProcessing template: special token ids, or the place of
// sequence A or B, with the type id its ids get.
type templatePart struct {
	specialIDs []int
	sequence   string
	typeID     int
}

// template is a TemplateProcessing post-processor. A nil pair means the file has no pair template.
type template struct {
	single []templatePart
	pair   []templatePart
}

type templateEntry struct {
	SpecialToken *struct {
		ID     string `json:"id"`
		TypeID int    `json:"type_id"`
	} `json:"SpecialToken"`
	Sequence *struct {
		ID     string `json:"id"`
		TypeID int    `json:"type_id"`
	} `json:"Sequence"`
}

func newTemplate(s step) (*template, error) {
	if s.Type != "TemplateProcessing" {
		return nil, fmt.Errorf("%q is not supported", s.Type)
	}
	var settings struct {
		Single        []templateEntry `json:"single"`
		Pair          []templateEntry `json:"pair"`
		SpecialTokens map[string]struct {
			IDs []int `json:"ids"`
		} `json:"special_tokens"`
	}
	if err := s.decode(&settings); err != nil {
		return nil, fmt.Errorf("TemplateProcessing: %w", err)
	}
	specials := make(map[string][]int, len(settings.SpecialTokens))
	for name, token := range settings.SpecialTokens {
		specials[name] = token.IDs
	}
	single, err := parseTemplate(settings.Single, specials, "single")
	if err != nil {
		return nil, err
	}
	if len(single) == 0 {
		return nil, errors.New("TemplateProcessing: the single template is empty")
	}
	var pair []templatePart
	if len(settings.Pair) > 0 {
		if pair, err = parseTemplate(settings.Pair, specials, "pair"); err != nil {
			return nil, err
		}
	}
	return &template{single: single, pair: pair}, nil
}

func parseTemplate(entries []templateEntry, specials map[string][]int, name string) ([]templatePart, error) {
	parts := make([]templatePart, 0, len(entries))
	for i, entry := range entries {
		switch {
		case entry.SpecialToken != nil:
			ids, ok := specials[entry.SpecialToken.ID]
			if !ok {
				return nil, fmt.Errorf("TemplateProcessing: %s entry %d names the special token %q, which special_tokens does not hold", name, i, entry.SpecialToken.ID)
			}
			parts = append(parts, templatePart{specialIDs: ids, typeID: entry.SpecialToken.TypeID})
		case entry.Sequence != nil:
			if entry.Sequence.ID != "A" && entry.Sequence.ID != "B" {
				return nil, fmt.Errorf("TemplateProcessing: %s entry %d names the sequence %q, want A or B", name, i, entry.Sequence.ID)
			}
			parts = append(parts, templatePart{sequence: entry.Sequence.ID, typeID: entry.Sequence.TypeID})
		default:
			return nil, fmt.Errorf("TemplateProcessing: %s entry %d is neither a SpecialToken nor a Sequence", name, i)
		}
	}
	return parts, nil
}

func (t *template) hasPair() bool {
	return t.pair != nil
}

func (t *template) countPairSpecials() int {
	count := 0
	for _, part := range t.pair {
		count += len(part.specialIDs)
	}
	return count
}

func (t *template) applySingle(a []int) []int {
	return fillTemplate(t.single, a, nil).IDs
}

func (t *template) applyPair(a, b []int) Encoding {
	return fillTemplate(t.pair, a, b)
}

func fillTemplate(parts []templatePart, a, b []int) Encoding {
	var out Encoding
	for _, part := range parts {
		ids := part.specialIDs
		switch part.sequence {
		case "A":
			ids = a
		case "B":
			ids = b
		}
		out.IDs = append(out.IDs, ids...)
		for range ids {
			out.TypeIDs = append(out.TypeIDs, part.typeID)
		}
	}
	return out
}
