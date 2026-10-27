package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

var errNoKinds = errors.New("the manifest lists no text kinds under textInput.kinds")

// readKinds is the text kinds the manifest lists under textInput.kinds.
func readKinds(manifest []byte) (map[domain.TextKind]bool, error) {
	var parsed struct {
		TextInput struct {
			Kinds []domain.TextKind `json:"kinds"`
		} `json:"textInput"`
	}
	if err := json.Unmarshal(manifest, &parsed); err != nil {
		return nil, fmt.Errorf("read the manifest: %w", err)
	}
	if len(parsed.TextInput.Kinds) == 0 {
		return nil, errNoKinds
	}
	kinds := make(map[domain.TextKind]bool, len(parsed.TextInput.Kinds))
	for _, kind := range parsed.TextInput.Kinds {
		kinds[kind] = true
	}
	return kinds, nil
}
