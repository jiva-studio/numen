package searchquery

import (
	"errors"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

// The fields of the messages of llm.rerank: RerankRequest{query 1, instruction 2, repeated documents 3} and
// RerankResult{repeated float scores 1}. The core does not import the contract.
const (
	fieldQuery       protowire.Number = 1
	fieldInstruction protowire.Number = 2
	fieldDocuments   protowire.Number = 3
	fieldScores      protowire.Number = 1
)

var errRerankResult = errors.New("the answer of the reranking service is not a RerankResult")

func encodeRerankRequest(query, instruction string, documents []string) []byte {
	var out []byte
	out = protowire.AppendTag(out, fieldQuery, protowire.BytesType)
	out = protowire.AppendString(out, query)
	if instruction != "" {
		out = protowire.AppendTag(out, fieldInstruction, protowire.BytesType)
		out = protowire.AppendString(out, instruction)
	}
	for _, d := range documents {
		out = protowire.AppendTag(out, fieldDocuments, protowire.BytesType)
		out = protowire.AppendString(out, d)
	}
	return out
}

func decodeRerankResult(payload []byte) ([]float64, error) {
	var scores []float64
	for len(payload) > 0 {
		number, kind, n := protowire.ConsumeTag(payload)
		if n < 0 {
			return nil, errRerankResult
		}
		payload = payload[n:]
		var err error
		switch {
		case number == fieldScores && kind == protowire.BytesType:
			scores, payload, err = consumePackedScores(scores, payload)
		case number == fieldScores && kind == protowire.Fixed32Type:
			scores, payload, err = consumeScore(scores, payload)
		default:
			payload, err = skipField(number, kind, payload)
		}
		if err != nil {
			return nil, err
		}
	}
	return scores, nil
}

func consumePackedScores(scores []float64, payload []byte) ([]float64, []byte, error) {
	packed, n := protowire.ConsumeBytes(payload)
	if n < 0 || len(packed)%4 != 0 {
		return nil, nil, errRerankResult
	}
	for len(packed) > 0 {
		var err error
		if scores, packed, err = consumeScore(scores, packed); err != nil {
			return nil, nil, err
		}
	}
	return scores, payload[n:], nil
}

func consumeScore(scores []float64, payload []byte) ([]float64, []byte, error) {
	bits, n := protowire.ConsumeFixed32(payload)
	if n < 0 {
		return nil, nil, errRerankResult
	}
	return append(scores, float64(math.Float32frombits(bits))), payload[n:], nil
}

func skipField(number protowire.Number, kind protowire.Type, payload []byte) ([]byte, error) {
	n := protowire.ConsumeFieldValue(number, kind, payload)
	if n < 0 {
		return nil, errRerankResult
	}
	return payload[n:], nil
}
