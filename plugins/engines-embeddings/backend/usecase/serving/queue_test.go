package serving

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
)

func TestLocalEmbeddingQueriesAndIndexBatchesWaitAtTheirOwnPriority(t *testing.T) {
	srv, _ := newRecordedServer(t)
	stopped := errors.New("queue stopped")
	for _, role := range []model.Role{model.RoleQuery, model.RoleDocument} {
		srv.queue = func(_ context.Context, work host.Work) (host.Turn, error) {
			if work.IsUserStarted != (role == model.RoleQuery) {
				t.Errorf("role %d requested priority %v", role, work.IsUserStarted)
			}
			return nil, stopped
		}
		if _, err := srv.EmbedTexts(t.Context(), role, []string{"text"}); !errors.Is(err, stopped) {
			t.Fatalf("role %d: %v", role, err)
		}
	}
}
