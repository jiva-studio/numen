package hostendpoint

import (
	"context"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func (e *searchEndpoint) ScoreChunks(
	ctx context.Context, req *connect.Request[v1.ScoreChunksRequest],
) (*connect.Response[v1.ScoreChunksResponse], error) {
	call, err := e.begin(req.Header(), manifest.PermissionIndexRead, true)
	if err != nil {
		return nil, err
	}
	if code := call.getRefusal(true); code != 0 {
		return connect.NewResponse(&v1.ScoreChunksResponse{Error: code}), nil
	}
	scored, floor, err := call.search.ScoreChunks(ctx, call.vault, readScoreQuery(req.Msg))
	code, err := answerSearch(ctx, err)
	if err != nil || code != 0 {
		return connect.NewResponse(&v1.ScoreChunksResponse{Error: code}), err
	}
	chunks := make([]*v1.ChunkScore, len(scored))
	for i, one := range scored {
		conditions := make([]*v1.ConditionScore, len(one.Conditions))
		for j, c := range one.Conditions {
			conditions[j] = &v1.ConditionScore{Similarity: c.Similarity, IsMatch: c.IsMatch}
		}
		chunks[i] = &v1.ChunkScore{ChunkId: string(one.ChunkID), Conditions: conditions}
	}
	return connect.NewResponse(&v1.ScoreChunksResponse{Chunks: chunks, Floor: floor}), nil
}

func readScoreQuery(msg *v1.ScoreChunksRequest) plugin.ScoreQuery {
	ids := make([]domain.ChunkID, len(msg.GetChunkIds()))
	for i, id := range msg.GetChunkIds() {
		ids[i] = domain.ChunkID(id)
	}
	conditions := make([]plugin.ScoreCondition, len(msg.GetConditions()))
	for i, c := range msg.GetConditions() {
		conditions[i] = plugin.ScoreCondition{Meaning: c.GetMeaning(), Words: c.GetWords()}
	}
	return plugin.ScoreQuery{ChunkIDs: ids, Conditions: conditions}
}
