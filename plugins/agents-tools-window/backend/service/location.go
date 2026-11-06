package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

var (
	errStateNotObject = errors.New("the state is not a JSON object")
	errNotPositiveInt = errors.New("not a positive integer")
	errNotSelection   = errors.New("not a non-empty array of non-negative numbers")
	errNotString      = errors.New("not a string")
)

// readTabLocation is the place the tab is open at, read from the state its reader saved: page and
// pageCount of a PDF, t of a recording, cfi and chapter of an EPUB. A tab whose state holds none of
// these keys has no location.
func readTabLocation(tab host.OpenTab) (*TabLocation, error) {
	if tab.State == "" {
		return nil, nil
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal([]byte(tab.State), &keys); err != nil || keys == nil {
		return nil, fmt.Errorf("tab %q: %w", tab.ID, errStateNotObject)
	}
	var location TabLocation
	isFound := false
	for _, field := range []struct {
		key  string
		read func(json.RawMessage) error
	}{
		{"page", func(raw json.RawMessage) error { return readPositiveInt(raw, &location.Page) }},
		{"pageCount", func(raw json.RawMessage) error { return readPositiveInt(raw, &location.PageCount) }},
		{"t", func(raw json.RawMessage) error { return readSelectionStart(raw, &location.PositionSeconds) }},
		{"cfi", func(raw json.RawMessage) error { return readString(raw, &location.CFI) }},
		{"chapter", func(raw json.RawMessage) error { return readString(raw, &location.Chapter) }},
	} {
		raw, hasKey := keys[field.key]
		if !hasKey {
			continue
		}
		if err := field.read(raw); err != nil {
			return nil, fmt.Errorf("tab %q: %s is %w", tab.ID, field.key, err)
		}
		isFound = true
	}
	if !isFound {
		return nil, nil
	}
	return &location, nil
}

func readPositiveInt(raw json.RawMessage, target *int) error {
	var n int
	if err := json.Unmarshal(raw, &n); err != nil || n < 1 {
		return errNotPositiveInt
	}
	*target = n
	return nil
}

// readSelectionStart reads the start of a selection saved as [start, end] seconds.
func readSelectionStart(raw json.RawMessage, target **float64) error {
	var selection []*float64
	if err := json.Unmarshal(raw, &selection); err != nil || len(selection) == 0 {
		return errNotSelection
	}
	for _, seconds := range selection {
		if seconds == nil || *seconds < 0 {
			return errNotSelection
		}
	}
	*target = selection[0]
	return nil
}

func readString(raw json.RawMessage, target *string) error {
	var text *string
	if err := json.Unmarshal(raw, &text); err != nil || text == nil {
		return errNotString
	}
	*target = *text
	return nil
}
