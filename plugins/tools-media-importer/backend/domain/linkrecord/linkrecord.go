// Package linkrecord is the record kept beside the text of a link: what fetched the text and what
// kind it is.
package linkrecord

import (
	"encoding/json"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// Record is what fetched the text of a link.
type Record struct {
	Address    string          `json:"address"`
	Title      string          `json:"title,omitempty"`
	Producer   string          `json:"producer,omitempty"`
	Downloader string          `json:"downloader,omitempty"`
	Kind       domain.TextKind `json:"kind,omitempty"`
}

// Encode returns the record as it is kept.
func Encode(rec Record) ([]byte, error) {
	return json.Marshal(rec)
}

// Decode reads a record as it is kept.
func Decode(raw []byte) (Record, error) {
	rec := Record{Address: "", Title: "", Producer: "", Downloader: "", Kind: ""}
	err := json.Unmarshal(raw, &rec)
	return rec, err
}
