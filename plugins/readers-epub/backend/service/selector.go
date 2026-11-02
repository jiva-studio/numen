package service

import (
	"encoding/json"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// cfiSelector locates a segment in a book: CFI is an EPUB Canonical Fragment Identifier.
type cfiSelector struct {
	CFI string `json:"cfi,omitempty"`
}

// encode returns the selector as the JSON a segment carries.
func (s cfiSelector) encode() (domain.Selector, error) {
	raw, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("encode CFI selector: %w", err)
	}
	return domain.Selector(raw), nil
}
