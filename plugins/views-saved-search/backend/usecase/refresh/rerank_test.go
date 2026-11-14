package refresh_test

import (
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/reranker/rerankertest"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/resultcache"
)

func getDocuments(requests []rerankertest.Request) [][]string {
	documents := make([][]string, len(requests))
	for i, request := range requests {
		documents[i] = request.Documents
	}
	return documents
}

func isEverySentOnce(requests []rerankertest.Request, want ...string) bool {
	return len(requests) == 1 && slices.Equal(slices.Sorted(slices.Values(requests[0].Documents)), slices.Sorted(slices.Values(want)))
}

func TestRerunAfterOneChunkChangedReranksOnlyThatChunk(t *testing.T) {
	w := newWorld(t, 0)
	w.runDue(t)
	w.standIn.ClearRequests()

	w.changeMooringChunk()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)

	requests := w.standIn.GetRequests()
	if len(requests) != 1 || !slices.Equal(requests[0].Documents, []string{mooringsText}) {
		t.Fatalf("rerank requests = %v, want the changed passage alone", getDocuments(requests))
	}
	record := w.readRecord(t, stormsID)
	if got := getResultTexts(record); !slices.Equal(got, []string{galeText, mooringsText}) {
		t.Errorf("results = %v, want the kept score of the gale and the new score of the moorings", got)
	}
	_, hasOld := record.RerankScores[resultcache.HashText(mooringText)]
	if len(record.RerankScores) != 2 || hasOld || record.RerankScores[resultcache.HashText(mooringsText)] != float64(float32(0.7)) {
		t.Errorf("rerank scores = %v, want those of the gale and the changed passage alone", record.RerankScores)
	}
}

func TestChangeOfTheRerankerModelReranksEveryResult(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	w.standIn.ClearRequests()

	w.rows.SetOptions(harborConnection, map[string]string{"model": "harbor-rerank-b"})
	w.clock.Advance(time.Minute)
	w.runDue(t)

	requests := w.standIn.GetRequests()
	if !isEverySentOnce(requests, galeText, mooringText) || requests[0].Options["model"] != "harbor-rerank-b" {
		t.Fatalf("rerank requests = %+v, want every passage sent once to the new model", requests)
	}
	if record := w.readRecord(t, stormsID); record.Reranker == nil || record.Reranker.Options["model"] != "harbor-rerank-b" {
		t.Errorf("reranker = %+v, want the row with the new model", record.Reranker)
	}
}

func TestChangeOfTheConnectionReranksEveryResult(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	pier := rerankertest.NewStandIn(map[string]float32{galeText: 0.2, mooringText: 0.8})
	w.rows.Clear()
	w.rows.Add("conn-pier", "Pier reranker", map[string]string{"model": "harbor-rerank-a"}, pier)

	w.clock.Advance(time.Minute)
	w.runDue(t)

	if requests := pier.GetRequests(); !isEverySentOnce(requests, galeText, mooringText) {
		t.Fatalf("rerank requests = %v, want every passage sent to the new connection", getDocuments(requests))
	}
	record := w.readRecord(t, stormsID)
	if record.Reranker == nil || record.Reranker.ConnectionID != "conn-pier" || !slices.Equal(getResultTexts(record), []string{mooringText, galeText}) {
		t.Errorf("record = %+v, want the order and the row of conn-pier", record)
	}
}

func TestRemovalOfTheLastConnectionRunsInFusedOrder(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	w.rows.Clear()

	ranAt := w.clock.Advance(time.Minute)
	w.runDue(t)

	record := w.readRecord(t, stormsID)
	if !record.ComputedAt.Equal(ranAt) || record.Order != "fused" || record.Reranker != nil || len(record.RerankScores) != 0 {
		t.Errorf("record = %+v, want a run in fused order without reranker", record)
	}
}

func TestAddingAConnectionRunsAFusedSearchAtOnce(t *testing.T) {
	w := newWorld(t, 10)
	w.rows.Clear()
	w.runDue(t)
	w.rows.Add(harborConnection, "Harbor reranker", map[string]string{"model": "harbor-rerank-a"}, w.standIn)

	w.clock.Advance(time.Minute)
	w.runDue(t)

	if requests := w.standIn.GetRequests(); !isEverySentOnce(requests, galeText, mooringText) {
		t.Errorf("rerank requests = %v, want every passage sent once a row is set", getDocuments(requests))
	}
	if record := w.readRecord(t, stormsID); record.Order != "reranked" {
		t.Errorf("order = %q, want reranked", record.Order)
	}
}

func TestChangeOfTheFloorReranksEveryResult(t *testing.T) {
	w := newWorld(t, 0)
	w.runDue(t)
	w.standIn.ClearRequests()

	w.search.SetModel(hosttest.SearchModel{Name: "harbor-model", Recipe: "harbor-recipe", FloorHint: 0.5})
	w.clock.Advance(time.Minute)
	w.changeIndex(t)

	if requests := w.standIn.GetRequests(); !isEverySentOnce(requests, galeText, mooringText) {
		t.Fatalf("rerank requests = %v, want every passage sent again", getDocuments(requests))
	}
	if record := w.readRecord(t, stormsID); record.Floor != 0.5 {
		t.Errorf("floor = %v, want 0.5", record.Floor)
	}
}

func TestChangeOfTheQueryRunsAtOnceAndReranksEveryResult(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	w.standIn.ClearRequests()
	query := `meaning("storm") && kind in ["note"]`

	w.vault.File(stormsPath, formatSearchNote(stormsID, query, stormsRerank, 0))
	ranAt := w.clock.Advance(time.Minute)
	w.changeIndex(t)

	if requests := w.standIn.GetRequests(); !isEverySentOnce(requests, galeText, mooringText) {
		t.Fatalf("rerank requests = %v, want every passage sent within the cooldown", getDocuments(requests))
	}
	if record := w.readRecord(t, stormsID); !record.ComputedAt.Equal(ranAt) || record.QueryHash != resultcache.HashText(query) {
		t.Errorf("record = %+v, want the new query computed at once", record)
	}
}

func TestChangeOfTheRerankTextRunsAtOnceAndReranksEveryResult(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	w.standIn.ClearRequests()

	w.vault.File(stormsPath, formatSearchNote(stormsID, stormsQuery, "Moorings broken in storms", 0))
	w.clock.Advance(time.Minute)
	w.changeIndex(t)

	requests := w.standIn.GetRequests()
	if !isEverySentOnce(requests, galeText, mooringText) || requests[0].Query != "Moorings broken in storms" {
		t.Fatalf("rerank requests = %+v, want every passage sent with the new rerank text", requests)
	}
	if record := w.readRecord(t, stormsID); record.RerankHash != resultcache.HashText("Moorings broken in storms") {
		t.Errorf("rerank hash = %s, want the hash of the new rerank text", record.RerankHash)
	}
}

func TestAnswerFromAnotherRowReranksEveryDocumentWithThatRow(t *testing.T) {
	w := newWorld(t, 0)
	pier := rerankertest.NewStandIn(map[string]float32{galeText: 0.1, mooringsText: 0.95})
	w.rows.Add("conn-pier", "Pier reranker", map[string]string{"model": "pier-model"}, pier)
	w.runDue(t)

	w.standIn.Fail(connection.NewTryNextError(connection.ReasonUnavailable, 0))
	w.changeMooringChunk()
	w.clock.Advance(time.Minute)
	w.changeIndex(t)

	requests := pier.GetRequests()
	if len(requests) == 0 || !isEverySentOnce(requests[len(requests)-1:], galeText, mooringsText) {
		t.Fatalf("rerank requests of the second row = %v, want its last one to hold every passage", getDocuments(requests))
	}
	record := w.readRecord(t, stormsID)
	if record.Reranker == nil || record.Reranker.ConnectionID != "conn-pier" || !slices.Equal(getResultTexts(record), []string{mooringsText, galeText}) {
		t.Errorf("record = %+v, want the order and the row of conn-pier", record)
	}
	if record.RerankScores[resultcache.HashText(galeText)] != float64(float32(0.1)) {
		t.Errorf("rerank scores = %v, want the score of the second row for the gale", record.RerankScores)
	}
}

func TestChangeOfTheLimitRunsAtOnceAndReranksOnlyNewDocuments(t *testing.T) {
	w := newWorld(t, 10)
	texts := make([]string, 8)
	chunks := make([]host.SearchResult, 8)
	for i := range texts {
		texts[i] = fmt.Sprintf("Storm number %d at the quay.", i)
		chunks[i] = chunk(fmt.Sprintf("c%d", i), fmt.Sprintf("Logbooks/%d.md", i), 0, 400, texts[i])
	}
	w.index(chunks...)
	w.vault.File(stormsPath, formatSearchNote(stormsID, stormsQuery, stormsRerank, 1))
	w.runDue(t)
	if requests := w.standIn.GetRequests(); len(requests) != 1 || !slices.Equal(requests[0].Documents, texts[:3]) {
		t.Fatalf("rerank requests = %v, want the first three passages for the limit 1", getDocuments(requests))
	}
	w.standIn.ClearRequests()

	w.vault.File(stormsPath, formatSearchNote(stormsID, stormsQuery, stormsRerank, 2))
	w.clock.Advance(time.Minute)
	w.changeIndex(t)

	if requests := w.standIn.GetRequests(); len(requests) != 1 || !slices.Equal(requests[0].Documents, texts[3:6]) {
		t.Errorf("rerank requests = %v, want the three passages not scored before, in fused order", getDocuments(requests))
	}
	if record := w.readRecord(t, stormsID); record.Limit != 2 || len(record.Results) != 2 || len(record.RerankScores) != 6 {
		t.Errorf("record = %+v, want the limit 2, two results and six scores", record)
	}
}

func TestChangeOfTheSettingsRunsAtOnce(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	first := w.readRecord(t, stormsID)
	config := w.getSettings().Preview
	config.MergeGapBytes = 50

	w.setPreviewConfig(config)
	ranAt := w.clock.Advance(time.Minute)
	w.runDue(t)

	record := w.readRecord(t, stormsID)
	if !record.ComputedAt.Equal(ranAt) || record.SettingsHash == first.SettingsHash {
		t.Errorf("record = %+v, want a run with the new settings", record)
	}
}

func TestChangeOfTheRefreshDelayDoesNotRunTheSearch(t *testing.T) {
	w := newWorld(t, 10)
	w.runDue(t)
	searches := w.host.getCallCount()
	if searches == 0 {
		t.Fatal("the first run did not search")
	}

	w.mu.Lock()
	w.settings.RefreshMinutes = 30
	w.mu.Unlock()
	w.clock.Advance(time.Minute)
	w.runDue(t)

	if got := w.host.getCallCount(); got != searches {
		t.Errorf("%d searches after the delay changed, want none", got-searches)
	}
}
