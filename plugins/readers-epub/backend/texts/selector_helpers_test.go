package texts

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

type cfiSelector struct {
	CFI string `json:"cfi,omitempty"`
}

func readCFISelector(raw domain.Selector) (cfiSelector, error) {
	var out cfiSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
