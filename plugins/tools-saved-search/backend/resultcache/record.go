// Package resultcache is the file the plugin keeps for each saved search: the last good results, the
// hashes of what produced them, the rerank scores to reuse, and the marks of a run that is due or failed.
package resultcache

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
)

// Record is the cache of one saved search. ComputedAt is the time of the last good run, zero before the
// first; StaleSince the time the index changed after it; FailedAt and Failure the time and reason of a
// run that failed after it. Limit is the limit the run used after the default and the cut, Floor the
// floor of its similarity, Reranker the row that reranked, nil in fused order. RerankScores holds a score
// for the hash of each text reranked, for RerankHash and Reranker. Errors are the problems of the note;
// a record with problems has no result.
type Record struct {
	Path           string             `json:"path"`
	QueryHash      string             `json:"queryHash"`
	RerankHash     string             `json:"rerankHash"`
	Limit          int                `json:"limit"`
	SettingsHash   string             `json:"settingsHash"`
	Reranker       *Reranker          `json:"reranker,omitempty"`
	Floor          float64            `json:"floor"`
	ComputedAt     time.Time          `json:"computedAt"`
	StaleSince     *time.Time         `json:"staleSince,omitempty"`
	FailedAt       *time.Time         `json:"failedAt,omitempty"`
	Failure        string             `json:"failure,omitempty"`
	Order          api.Order          `json:"order"`
	CandidateCount int                `json:"candidateCount"`
	Results        []Result           `json:"results"`
	RerankScores   map[string]float64 `json:"rerankScores"`
	Warnings       []string           `json:"warnings"`
	Errors         []api.Problem      `json:"errors"`
}

// Reranker is the row of the connections setting that reranked: its connection and the values of its
// row fields.
type Reranker struct {
	ConnectionID string            `json:"connectionId"`
	Options      map[string]string `json:"options"`
}

// Result is a result of the run and the chunk ids of its passages, in the order of their start.
type Result struct {
	api.PreviewResult

	ChunkIDs []string `json:"chunkIds"`
}

// Encode returns the bytes of the record.
func Encode(record Record) ([]byte, error) {
	return json.Marshal(record)
}

// Decode reads a record. It returns false for bytes that are not a record: broken JSON, a value that is
// not an object, or a field of another kind.
func Decode(raw []byte) (Record, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Record{}, false
	}
	var record Record
	if err := json.Unmarshal(trimmed, &record); err != nil {
		return Record{}, false
	}
	return record, true
}
