package resultcache_test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
)

func newRecord() resultcache.Record {
	computedAt := time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC)
	staleSince := time.Date(2026, 11, 14, 9, 41, 12, 0, time.UTC)
	failedAt := time.Date(2026, 11, 14, 9, 52, 0, 0, time.UTC)
	page := resultcache.Place{Page: 14}
	score := 0.913
	return resultcache.Record{
		Path:            "Harbor/Harbor master- winter storms.md",
		QueryHash:       resultcache.HashText(`meaning("the harbor master in a winter storm").top(30)`),
		Floor:           0.3,
		ComputedAt:      computedAt,
		StaleSince:      &staleSince,
		FailedAt:        &failedAt,
		Failure:         "the reranking service did not answer",
		Order:           resultcache.OrderReranked,
		Reranker:        "Harbor reranker",
		IndexGeneration: "gen-42",
		Results: []resultcache.Result{{
			Path: "Logbooks/1911.pdf", Title: "1911", Kind: "pdf", Place: &page,
			Text:  "…the harbor master rowed out to the moored boats while the storm…",
			Start: 48210, Length: 2410, Line: 380, URI: "numen:Logbooks%2F1911.pdf?start=48210&length=2410", Score: &score,
		}},
		Warnings: []string{},
		Errors:   []resultcache.Problem{},
	}
}

func TestDecodeReadsWhatEncodeWrote(t *testing.T) {
	record := newRecord()

	raw, err := resultcache.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, ok := resultcache.Decode(raw)

	if !ok {
		t.Fatalf("Decode(%s) is no record", raw)
	}
	if !reflect.DeepEqual(decoded, record) {
		t.Errorf("Decode(Encode(record)) = %+v, want %+v", decoded, record)
	}
}

func TestEncodeWritesTheKeysOfTheRecord(t *testing.T) {
	raw, err := resultcache.Encode(newRecord())
	if err != nil {
		t.Fatal(err)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{
		"path", "queryHash", "floor", "computedAt", "staleSince", "failedAt", "failure", "order", "reranker",
		"indexGeneration", "results", "warnings", "errors",
	} {
		if _, isThere := keys[key]; !isThere {
			t.Errorf("record %s has no key %q", raw, key)
		}
	}
	if string(keys["computedAt"]) != `"2026-11-14T09:30:00Z"` || string(keys["failedAt"]) != `"2026-11-14T09:52:00Z"` {
		t.Errorf("computedAt = %s, failedAt = %s, want UTC RFC 3339 with seconds", keys["computedAt"], keys["failedAt"])
	}
}

func TestEncodeOmitsUnsetMarks(t *testing.T) {
	record := newRecord()
	record.StaleSince, record.FailedAt, record.Failure, record.Reranker = nil, nil, "", ""

	raw, err := resultcache.Encode(record)
	if err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{`"staleSince"`, `"failedAt"`, `"failure"`, `"reranker"`} {
		if strings.Contains(string(raw), key) {
			t.Errorf("record %s holds %s, want it absent when not set", raw, key)
		}
	}
}

func TestDecodeOfBrokenBytesIsNoRecord(t *testing.T) {
	for name, raw := range map[string]string{
		"broken JSON":             `{"path": "Harbor/Gale.md", "results": [`,
		"a field of a wrong kind": `{"path": 12, "results": []}`,
		"not an object":           `["Harbor/Gale.md"]`,
		"empty":                   ``,
		"a record of the old shape": `{"path": "Harbor/Gale.md", "rerankHash": "ab", "limit": 20, "results": [], ` +
			`"rerankScores": {}}`,
	} {
		t.Run(name, func(t *testing.T) {
			if record, ok := resultcache.Decode([]byte(raw)); ok {
				t.Errorf("Decode(%q) = %+v, want no record", raw, record)
			}
		})
	}
}

func TestGetFileNameUsesTheNoteID(t *testing.T) {
	if got := resultcache.GetFileName("01JCX6Q4ZB3W8N2V7T5R9K1M0P", "Harbor/Gale.md"); got != "01JCX6Q4ZB3W8N2V7T5R9K1M0P.json" {
		t.Errorf("GetFileName = %q, want the id with .json", got)
	}
}

func TestGetFileNameOfANoteWithoutIDHashesThePath(t *testing.T) {
	sum := sha256.Sum256([]byte("Harbor/Gale.md"))
	want := "path-" + hex.EncodeToString(sum[:])[:16] + ".json"

	if got := resultcache.GetFileName("", "Harbor/Gale.md"); got != want {
		t.Errorf("GetFileName = %q, want %q", got, want)
	}
	if resultcache.GetFileName("", "Harbor/Calm.md") == want {
		t.Error("two paths give one file name")
	}
}

func TestHashTextIsTheSHA256OfTheText(t *testing.T) {
	sum := sha256.Sum256([]byte("winter storms"))

	if got := resultcache.HashText("winter storms"); got != hex.EncodeToString(sum[:]) {
		t.Errorf("HashText = %q, want the hex SHA-256", got)
	}
}
