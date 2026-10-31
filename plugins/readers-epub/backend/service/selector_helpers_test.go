package service

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// readCFISelector reads the CFI of a segment's selector. An empty selector has none.
func readCFISelector(raw domain.Selector) (cfiSelector, error) {
	var out cfiSelector
	if raw == "" {
		return out, nil
	}
	err := json.Unmarshal([]byte(raw), &out)
	return out, err
}
