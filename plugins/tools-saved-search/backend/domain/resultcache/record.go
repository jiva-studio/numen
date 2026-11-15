// Package resultcache is the file the plugin keeps for each saved search: the last good results, the hash
// of the query that produced them, and the marks of a run that is due or failed.
package resultcache

import (
	"bytes"
	"encoding/json"
	"time"
)

// Record is the cache of one saved search. ComputedAt is the time of the last good run, zero before the
// first; StaleSince the time the index changed after it; FailedAt and Failure the time and reason of a
// run that failed after it. Floor, Order, Reranker and IndexGeneration are what the core reported for the
// last good run: Reranker is the connection that reranked, empty in fused order. Errors are the problems
// of the last run, counted from the first line of the query; the results stay those of the last good run.
type Record struct {
	Path            string     `json:"path"`
	QueryHash       string     `json:"queryHash"`
	Floor           float64    `json:"floor"`
	ComputedAt      time.Time  `json:"computedAt"`
	StaleSince      *time.Time `json:"staleSince,omitempty"`
	FailedAt        *time.Time `json:"failedAt,omitempty"`
	Failure         string     `json:"failure,omitempty"`
	Order           Order      `json:"order"`
	Reranker        string     `json:"reranker,omitempty"`
	IndexGeneration string     `json:"indexGeneration,omitempty"`
	Results         []Result   `json:"results"`
	Warnings        []string   `json:"warnings"`
	Errors          []Problem  `json:"errors"`
}

// Encode returns the bytes of the record.
func Encode(record Record) ([]byte, error) {
	return json.Marshal(record)
}

// Decode reads a record. It returns false for bytes that are not a record: broken JSON, a value that is
// not an object, a field of another kind or a field a record does not have.
func Decode(raw []byte) (Record, bool) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return Record{}, false
	}
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.DisallowUnknownFields()
	var record Record
	if err := decoder.Decode(&record); err != nil {
		return Record{}, false
	}
	return record, true
}
