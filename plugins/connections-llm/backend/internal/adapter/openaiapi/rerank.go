package openaiapi

import (
	"context"
	"net/http"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/domain/servicetype"
)

// rerankBody is a request in the format of Cohere and Jina. TopN is the number of documents, so a
// service that returns fewer results by default still scores every document.
type rerankBody struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

type rerankAnswer struct {
	Results []struct {
		Index          *int     `json:"index"`
		RelevanceScore *float32 `json:"relevance_score"`
	} `json:"results"`
}

// Rerank asks the rerank endpoint of the service, in the format of Cohere and Jina, for one score for each
// document, in the order of the documents. The format has no field for an instruction, so the instruction
// and a line break go before the query. OpenAI has no rerank endpoint. An answer with another number of
// results than documents, an index that is missing, out of range or repeated, or a missing score is an
// unavailable service.
func (g *Gateway) Rerank(ctx context.Context, call connection.Call, limit time.Duration, request *llm.RerankRequest) ([]float32, error) {
	if call.Type == servicetype.OpenAI {
		return nil, connection.NewStopError(connection.ReasonInvalidRequest)
	}
	query := request.GetQuery()
	if instruction := request.GetInstruction(); instruction != "" {
		query = instruction + "\n" + query
	}
	body := rerankBody{Model: call.Options[llm.OptionModel], Query: query, Documents: request.GetDocuments(), TopN: len(request.GetDocuments())}
	var answer rerankAnswer
	if err := g.send(ctx, call, limit, endpoint{http.MethodPost, "/rerank"}, body, &answer); err != nil {
		return nil, err
	}
	return orderScores(answer, len(request.GetDocuments()))
}

func orderScores(answer rerankAnswer, count int) ([]float32, error) {
	if len(answer.Results) != count {
		return nil, connection.NewStopError(connection.ReasonUnavailable)
	}
	scores := make([]float32, count)
	isSet := make([]bool, count)
	for _, one := range answer.Results {
		if one.Index == nil || one.RelevanceScore == nil {
			return nil, connection.NewStopError(connection.ReasonUnavailable)
		}
		index := *one.Index
		if index < 0 || index >= count || isSet[index] {
			return nil, connection.NewStopError(connection.ReasonUnavailable)
		}
		scores[index], isSet[index] = *one.RelevanceScore, true
	}
	return scores, nil
}
