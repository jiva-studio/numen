// Package rerankproto encodes the request and decodes the result of the operation llm.rerank with the generated
// messages of the core protocol.
package rerankproto

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	numenv1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
)

var errRerankResult = errors.New("the answer of the reranking service is not a RerankResult")

// Codec is the wire format of llm.rerank.
type Codec struct{}

// EncodeRequest leaves out an empty instruction. A text that is not UTF-8 cannot be sent, because the message
// holds strings.
func (Codec) EncodeRequest(query, instruction string, documents []string) ([]byte, error) {
	payload, err := proto.Marshal(&numenv1.RerankRequest{Query: query, Instruction: instruction, Documents: documents})
	if err != nil {
		return nil, fmt.Errorf("the request of the reranking service cannot be written: %w", err)
	}
	return payload, nil
}

// DecodeResult reads the scores packed or one field each.
func (Codec) DecodeResult(payload []byte) ([]float64, error) {
	var result numenv1.RerankResult
	if err := proto.Unmarshal(payload, &result); err != nil {
		return nil, fmt.Errorf("%w: %w", errRerankResult, err)
	}
	scores := make([]float64, len(result.GetScores()))
	for i, score := range result.GetScores() {
		scores[i] = float64(score)
	}
	return scores, nil
}
