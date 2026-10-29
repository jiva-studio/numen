package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var errNoKinds = errors.New("the manifest lists no text kinds under the condition of the proofread command")

// proofreadCommand is the id of the command that proofreads a text version.
const proofreadCommand = "proofread"

// readKinds is the text kinds the manifest lists under `when.liveTextKinds` of the proofread command.
func readKinds(manifest []byte) (map[domain.TextKind]bool, error) {
	var parsed struct {
		Commands []struct {
			ID   string `json:"id"`
			When struct {
				LiveTextKinds []domain.TextKind `json:"liveTextKinds"`
			} `json:"when"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(manifest, &parsed); err != nil {
		return nil, fmt.Errorf("read the manifest: %w", err)
	}
	for _, command := range parsed.Commands {
		if command.ID != proofreadCommand || len(command.When.LiveTextKinds) == 0 {
			continue
		}
		kinds := make(map[domain.TextKind]bool, len(command.When.LiveTextKinds))
		for _, kind := range command.When.LiveTextKinds {
			kinds[kind] = true
		}
		return kinds, nil
	}
	return nil, errNoKinds
}
