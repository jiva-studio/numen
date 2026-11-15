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

func newTemplate(s step) (*template, error) {
	if s.Type != "TemplateProcessing" {
		return nil, fmt.Errorf("step %q is not supported", s.Type)
	}
	var singleEntries, pairEntries []jsonObject
	var tokens map[string]jsonObject
	for _, field := range []struct {
		key string
		out any
	}{
		{"single", &singleEntries},
		{"pair", &pairEntries},
		{"special_tokens", &tokens},
	} {
		if err := s.fields.read(field.key, field.out); err != nil {
			return nil, fmt.Errorf("step TemplateProcessing: %w", err)
		}
	}
	specials := make(map[string][]int, len(tokens))
	for name, token := range tokens {
		var ids []int
		if err := token.read("ids", &ids); err != nil {
			return nil, fmt.Errorf("step TemplateProcessing: special token %q: %w", name, err)
		}
		specials[name] = ids
	}
	single, err := parseTemplate(singleEntries, specials, "single")
	if err != nil {
		return nil, err
	}
	if len(single) == 0 {
		return nil, errors.New("step TemplateProcessing: the single template is empty")
	}
	var pair []templatePart
	if len(pairEntries) > 0 {
		if pair, err = parseTemplate(pairEntries, specials, "pair"); err != nil {
			return nil, err
		}
	}
	return &template{single: single, pair: pair}, nil
}

func parseTemplate(entries []jsonObject, specials map[string][]int, name string) ([]templatePart, error) {
	parts := make([]templatePart, 0, len(entries))
	for i, entry := range entries {
		part, err := parseTemplateEntry(entry, specials)
		if err != nil {
			return nil, fmt.Errorf("step TemplateProcessing: %s entry %d: %w", name, i, err)
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func parseTemplateEntry(entry jsonObject, specials map[string][]int) (templatePart, error) {
	var special, sequence jsonObject
	if err := entry.read("SpecialToken", &special); err != nil {
		return templatePart{}, err
	}
	if err := entry.read("Sequence", &sequence); err != nil {
		return templatePart{}, err
	}
	var id string
	var typeID int
	switch {
	case special != nil:
		if err := readEntryFields(special, &id, &typeID); err != nil {
			return templatePart{}, err
		}
		ids, ok := specials[id]
		if !ok {
			return templatePart{}, fmt.Errorf("the special token %q is not in special_tokens", id)
		}
		return templatePart{specialIDs: ids, typeID: typeID}, nil
	case sequence != nil:
		if err := readEntryFields(sequence, &id, &typeID); err != nil {
			return templatePart{}, err
		}
		if id != "A" && id != "B" {
			return templatePart{}, fmt.Errorf("the sequence %q is neither A nor B", id)
		}
		return templatePart{sequence: id, typeID: typeID}, nil
	default:
		return templatePart{}, errors.New("neither a SpecialToken nor a Sequence")
	}
}

func readEntryFields(entry jsonObject, id *string, typeID *int) error {
	if err := entry.read("id", id); err != nil {
		return err
	}
	return entry.read("type_id", typeID)
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
