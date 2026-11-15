package hostendpoint

import (
	"context"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

func (s *recordingSearch) ScoreChunks(_ context.Context, v domain.Vault, q plugin.ScoreQuery) ([]plugin.ChunkScore, float64, error) {
	s.vaults = append(s.vaults, v.ID)
	s.scoreQueries = append(s.scoreQueries, q)
	similarity := 0.8
	return []plugin.ChunkScore{{
		ChunkID:    "7",
		Conditions: []plugin.ConditionScore{{Similarity: &similarity, IsMatch: true}, {IsMatch: false}},
	}}, 0.42, s.err
}

func getScoreRequest() *v1.ScoreChunksRequest {
	return &v1.ScoreChunksRequest{
		ChunkIds: []string{"7", "9"},
		Conditions: []*v1.ScoreCondition{
			{Condition: &v1.ScoreCondition_Meaning{Meaning: "herons at dusk"}},
			{Condition: &v1.ScoreCondition_Words{Words: "heron NOT reed"}},
		},
	}
}

func TestHostScoreChunksReturnsTheScoresAndTheFloor(t *testing.T) {
	search := &recordingSearch{}
	fakeCallers := newSearchCallers(search)
	client := newSearchClient(t, fakeCallers, "token-a")

	got, err := client.ScoreChunks(t.Context(), connect.NewRequest(getScoreRequest()))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("returned %v (%v)", got.Msg, err)
	}
	if got.Msg.GetFloor() != 0.42 || len(got.Msg.GetChunks()) != 1 {
		t.Fatalf("the answer is %v, want the floor 0.42 and one chunk", got.Msg)
	}
	chunk := got.Msg.GetChunks()[0]
	if chunk.GetChunkId() != "7" || len(chunk.GetConditions()) != 2 {
		t.Fatalf("the chunk is %v, want chunk 7 with two scores", chunk)
	}
	meaning, words := chunk.GetConditions()[0], chunk.GetConditions()[1]
	if meaning.Similarity == nil || meaning.GetSimilarity() != 0.8 || !meaning.GetIsMatch() {
		t.Errorf("the meaning score is %v, want the similarity 0.8 and a match", meaning)
	}
	if words.Similarity != nil || words.GetIsMatch() {
		t.Errorf("the words score is %v, want no similarity and no match", words)
	}

	want := plugin.ScoreQuery{
		ChunkIDs:   []domain.ChunkID{"7", "9"},
		Conditions: []plugin.ScoreCondition{{Meaning: "herons at dusk"}, {Words: "heron NOT reed"}},
	}
	if len(search.scoreQueries) != 1 || !slices.Equal(search.scoreQueries[0].ChunkIDs, want.ChunkIDs) ||
		!slices.Equal(search.scoreQueries[0].Conditions, want.Conditions) {
		t.Errorf("the search received %+v, want %+v", search.scoreQueries, want)
	}
	if !slices.Equal(fakeCallers.checked, []string{"md.numen.a index:read"}) {
		t.Errorf("checked %v, want index:read once", fakeCallers.checked)
	}
}

func TestHostScoreChunksRejectsAPluginWithoutIndexRead(t *testing.T) {
	search := &recordingSearch{}
	client := newSearchClient(t, newSearchCallers(search), "token-b")

	_, err := client.ScoreChunks(t.Context(), connect.NewRequest(getScoreRequest()))
	if readCode(t, err) != connect.CodePermissionDenied {
		t.Errorf("a plugin without index:read got %v, want permission denied", err)
	}
	if len(search.scoreQueries) != 0 {
		t.Errorf("the search ran for a rejected call: %+v", search.scoreQueries)
	}
}

func TestHostScoreChunksMapsEachErrorToItsCode(t *testing.T) {
	for _, c := range []struct {
		err  error
		want v1.HostSearchErrorCode
	}{
		{plugin.ErrLimitExceeded, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED},
		{plugin.ErrInvalidCondition, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_CONDITION},
		{plugin.ErrInvalidWords, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_WORDS},
		{plugin.ErrNoModel, v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL},
	} {
		client := newSearchClient(t, newSearchCallers(&recordingSearch{err: c.err}), "token-a")

		got, err := client.ScoreChunks(t.Context(), connect.NewRequest(getScoreRequest()))
		if err != nil || got.Msg.GetError() != c.want || len(got.Msg.GetChunks()) != 0 {
			t.Errorf("%v answered %v (%v), want %v and no chunks", c.err, got.Msg, err, c.want)
		}
	}
}

func TestHostScoreChunksAnswersNoModelWithoutASearchIndex(t *testing.T) {
	client := newSearchClient(t, newSearchCallers(nil), "token-a")

	got, err := client.ScoreChunks(t.Context(), connect.NewRequest(getScoreRequest()))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL {
		t.Errorf("returned %v (%v), want NO_MODEL", got.Msg, err)
	}
}

func TestHostScoreChunksPassesAConditionWithNoTextToTheSearch(t *testing.T) {
	search := &recordingSearch{err: plugin.ErrInvalidCondition}
	client := newSearchClient(t, newSearchCallers(search), "token-a")

	got, err := client.ScoreChunks(t.Context(), connect.NewRequest(&v1.ScoreChunksRequest{
		ChunkIds: []string{"7"}, Conditions: []*v1.ScoreCondition{{}},
	}))
	if err != nil || got.Msg.GetError() != v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_CONDITION {
		t.Errorf("a condition with no text answered %v (%v), want INVALID_CONDITION", got.Msg, err)
	}
	if len(search.scoreQueries) == 1 && !slices.Equal(search.scoreQueries[0].Conditions, []plugin.ScoreCondition{{}}) {
		t.Errorf("the search received %+v, want one condition with no text", search.scoreQueries[0].Conditions)
	}
}
