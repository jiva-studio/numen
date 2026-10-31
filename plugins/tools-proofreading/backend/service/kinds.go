package service

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

var errNoKinds = errors.New("the manifest lists no text kinds in the conditions of the proofread command")

// proofreadCommand is the id of the command that proofreads a text version.
const proofreadCommand = "proofread"

// kindKey is the key of the context condition that lists the kinds of text version the command is
// offered over.
const kindKey = "version.kind"

// readKinds is the text kinds the manifest lists in the `version.kind` condition of the proofread command.
func readKinds(manifest []byte) (map[domain.TextKind]bool, error) {
	var parsed struct {
		Commands []struct {
			ID   string `json:"id"`
			When []struct {
				Key string            `json:"key"`
				Is  []domain.TextKind `json:"is"`
			} `json:"when"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(manifest, &parsed); err != nil {
		return nil, fmt.Errorf("read the manifest: %w", err)
	}
	for _, command := range parsed.Commands {
		if command.ID != proofreadCommand {
			continue
		}
		for _, condition := range command.When {
			if condition.Key != kindKey || len(condition.Is) == 0 {
				continue
			}
			kinds := make(map[domain.TextKind]bool, len(condition.Is))
			for _, kind := range condition.Is {
				kinds[kind] = true
			}
			return kinds, nil
		}
	}
	return nil, errNoKinds
}
