package host_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1/numenv1connect"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/hostcall"
)

type scoreHost struct {
	numenv1connect.UnimplementedHostSearchServiceHandler
	code        v1.HostSearchErrorCode
	status      *connect.Error
	vault       string
	textRequest *v1.HostSearchServiceSearchTextRequest
	request     *v1.ScoreChunksRequest
}

func (h *scoreHost) SearchText(_ context.Context, req *connect.Request[v1.HostSearchServiceSearchTextRequest]) (*connect.Response[v1.HostSearchServiceSearchTextResponse], error) {
	h.textRequest = req.Msg
	if h.status != nil {
		return nil, h.status
	}
	similarity, wordScore := 0.61, 4.25
	return connect.NewResponse(&v1.HostSearchServiceSearchTextResponse{
		Error: h.code,
		Results: []*v1.SearchResult{
			{Path: "notes/Heron.md", Text: "herons fish", ByteRange: &v1.Range{Start: 20, End: 60}, ChunkId: "41", Similarity: &similarity, WordScore: &wordScore},
			{Path: "notes/Reed.md", Text: "reeds sway", ByteRange: &v1.Range{Start: 0, End: 10}, ChunkId: "42"},
		},
	}), nil
}

func (h *scoreHost) ScoreChunks(_ context.Context, req *connect.Request[v1.ScoreChunksRequest]) (*connect.Response[v1.ScoreChunksResponse], error) {
	h.vault, h.request = req.Header().Get("Numen-Vault"), req.Msg
	if h.status != nil {
		return nil, h.status
	}
	similarity := 0.73
	return connect.NewResponse(&v1.ScoreChunksResponse{
		Error: h.code,
		Floor: 0.42,
		Chunks: []*v1.ChunkScore{{
			ChunkId: "41",
			Conditions: []*v1.ConditionScore{
				{Similarity: &similarity, IsMatch: true},
				{IsMatch: false},
			},
		}},
	}), nil
}

func dialScoreHost(t *testing.T, h *scoreHost) *host.SearchClient {
	t.Helper()
	mux := http.NewServeMux()
	mux.Handle(numenv1connect.NewHostSearchServiceHandler(h))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return host.NewSearchClient(hostcall.Endpoint{BaseURL: server.URL, Token: "tok"})
}

func TestASearchClientReadsTheChunkIDTheSimilarityAndTheWordScoreOfAResult(t *testing.T) {
	got, err := dialScoreHost(t, &scoreHost{}).SearchText(t.Context(), host.SearchResultQuery{Query: "heron"})
	if err != nil || len(got) != 2 {
		t.Fatalf("found %+v, %v, want two results", got, err)
	}
	scored, plain := got[0], got[1]
	if scored.ChunkID != "41" || scored.Similarity == nil || *scored.Similarity != 0.61 || scored.WordScore == nil || *scored.WordScore != 4.25 {
		t.Errorf("the scored result is %+v, want chunk 41, similarity 0.61 and word score 4.25", scored)
	}
	if plain.ChunkID != "42" || plain.Similarity != nil || plain.WordScore != nil {
		t.Errorf("the result with no scores is %+v, want chunk 42 and no scores", plain)
	}
}

func TestASearchClientSendsTheResultsPerSourceTheModeAndTheRawWordsFlag(t *testing.T) {
	h := &scoreHost{}
	c := dialScoreHost(t, h)

	_, err := c.SearchText(t.Context(), host.SearchResultQuery{
		Query: "NEAR(heron reed, 3)", Limit: 300, ResultsPerSource: 10, Mode: host.SearchModeWords, IsRawWords: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	r := h.textRequest
	if r.GetLimit() != 300 || r.GetResultsPerSource() != 10 || r.GetMode() != v1.SearchMode_SEARCH_MODE_WORDS || !r.GetIsRawWords() {
		t.Errorf("the host received %v, want 300 results, 10 per file, words mode and raw words", r)
	}

	if _, err := c.SearchText(t.Context(), host.SearchResultQuery{Query: "heron", Mode: host.SearchModeMeaning}); err != nil {
		t.Fatal(err)
	}
	if got := h.textRequest.GetMode(); got != v1.SearchMode_SEARCH_MODE_MEANING || h.textRequest.GetIsRawWords() {
		t.Errorf("meaning mode was sent as %v, raw %v", got, h.textRequest.GetIsRawWords())
	}

	if _, err := c.SearchText(t.Context(), host.SearchResultQuery{Query: "heron"}); err != nil {
		t.Fatal(err)
	}
	if got := h.textRequest.GetMode(); got != v1.SearchMode_SEARCH_MODE_UNSPECIFIED && got != v1.SearchMode_SEARCH_MODE_HYBRID {
		t.Errorf("the zero mode was sent as %v, want a hybrid search", got)
	}
}

func TestASearchClientRefusesAnUnknownModeBeforeTheCall(t *testing.T) {
	h := &scoreHost{}

	_, err := dialScoreHost(t, h).SearchText(t.Context(), host.SearchResultQuery{Query: "heron", Mode: host.SearchMode(9)})
	if err == nil {
		t.Error("an unknown mode returned no error")
	}
	if h.textRequest != nil {
		t.Errorf("the host received %v for an unknown mode", h.textRequest)
	}
}

func TestASearchClientMapsInvalidWordsToErrInvalidWords(t *testing.T) {
	h := &scoreHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_WORDS}

	_, err := dialScoreHost(t, h).SearchText(t.Context(), host.SearchResultQuery{Query: `"heron`, Mode: host.SearchModeWords, IsRawWords: true})
	if !errors.Is(err, host.ErrInvalidWords) {
		t.Errorf("got %v, want ErrInvalidWords", err)
	}
}

func TestASearchClientReadsTheScoresAndTheFloorOfScoreChunks(t *testing.T) {
	h := &scoreHost{}

	got, err := dialScoreHost(t, h).With("v2").ScoreChunks(t.Context(), host.ScoreQuery{
		ChunkIDs:   []string{"41", "43"},
		Conditions: []host.ScoreCondition{{Meaning: "herons at dusk"}, {Words: "heron NOT reed"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Floor != 0.42 || len(got.Chunks) != 1 || got.Chunks[0].ChunkID != "41" || len(got.Chunks[0].Conditions) != 2 {
		t.Fatalf("the scores are %+v, want the floor 0.42 and chunk 41 with two scores", got)
	}
	meaning, words := got.Chunks[0].Conditions[0], got.Chunks[0].Conditions[1]
	if meaning.Similarity == nil || *meaning.Similarity != 0.73 || !meaning.IsMatch {
		t.Errorf("the meaning score is %+v, want the similarity 0.73 and a match", meaning)
	}
	if words.Similarity != nil || words.IsMatch {
		t.Errorf("the words score is %+v, want no similarity and no match", words)
	}

	r := h.request
	if !slices.Equal(r.GetChunkIds(), []string{"41", "43"}) || len(r.GetConditions()) != 2 ||
		r.GetConditions()[0].GetMeaning() != "herons at dusk" || r.GetConditions()[1].GetWords() != "heron NOT reed" || h.vault != "v2" {
		t.Errorf("the host received %v for vault %q", r, h.vault)
	}
}

func TestScoreChunksMapsTheErrorsOfTheHost(t *testing.T) {
	for name, tc := range map[string]struct {
		h    *scoreHost
		want error
	}{
		"invalid condition": {&scoreHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_CONDITION}, host.ErrInvalidCondition},
		"invalid words":     {&scoreHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_INVALID_WORDS}, host.ErrInvalidWords},
		"limit exceeded":    {&scoreHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_LIMIT_EXCEEDED}, host.ErrLimitExceeded},
		"no model":          {&scoreHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_MODEL}, host.ErrNoModel},
		"no vault":          {&scoreHost{code: v1.HostSearchErrorCode_HOST_SEARCH_ERROR_CODE_NO_VAULT}, host.ErrNoVault},
		"permission denied": {&scoreHost{status: connect.NewError(connect.CodePermissionDenied, errors.New("no"))}, host.ErrPermissionDenied},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := dialScoreHost(t, tc.h).ScoreChunks(t.Context(), host.ScoreQuery{
				ChunkIDs: []string{"41"}, Conditions: []host.ScoreCondition{{Words: "heron"}},
			})
			if !errors.Is(err, tc.want) {
				t.Errorf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestTheSearchLimitsOfTheSDKAreTheLimitsOfTheHost(t *testing.T) {
	if host.MaxSearchedResults != 500 || host.DefaultSearchedResultsPerSource != 3 {
		t.Errorf("a search returns at most %d results and %d per file by default, want 500 and 3", host.MaxSearchedResults, host.DefaultSearchedResultsPerSource)
	}
	if host.MaxScoredChunks != 500 || host.MaxScoredConditions != 16 {
		t.Errorf("ScoreChunks takes %d chunks and %d conditions, want 500 and 16", host.MaxScoredChunks, host.MaxScoredConditions)
	}
}
