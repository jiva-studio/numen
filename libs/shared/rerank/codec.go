package rerank

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

var errResult = errors.New("the answer of the reranking service is not a RerankResult")

// EncodeRequest writes the payload of llm.rerank and leaves out an empty instruction. A text that is not UTF-8
// cannot be sent, because the message holds strings.
func EncodeRequest(query, instruction string, documents []string) ([]byte, error) {
	payload, err := proto.Marshal(&RerankRequest{Query: query, Instruction: instruction, Documents: documents})
	if err != nil {
		return nil, fmt.Errorf("the request of the reranking service cannot be written: %w", err)
	}
	return payload, nil
}

// DecodeResult reads the scores of the done frame of llm.rerank, packed or one field each.
func DecodeResult(payload []byte) ([]float64, error) {
	var result RerankResult
	if err := proto.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("%w: %w", errResult, err)
	}
	scores := make([]float64, len(result.GetScores()))
	for i, score := range result.GetScores() {
		scores[i] = float64(score)
	}
	return scores, nil
}
