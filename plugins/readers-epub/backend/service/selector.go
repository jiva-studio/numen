package service

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// cfiSelector locates a segment in a book: CFI is an EPUB Canonical Fragment Identifier.
type cfiSelector struct {
	CFI string `json:"cfi,omitempty"`
}

// encode returns the selector as the JSON a segment carries.
func (s cfiSelector) encode() domain.Selector {
	// A struct of strings always marshals.
	raw, _ := json.Marshal(s)
	return domain.Selector(raw)
}

// readCFISelector reads the CFI of a segment's selector. An empty selector has none.
func readCFISelector(raw domain.Selector) (cfiSelector, error) {
	var out cfiSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
