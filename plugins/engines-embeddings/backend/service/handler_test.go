package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/rpcerror"
)

func TestAModelThatHasNoProfileIsAFailedPrecondition(t *testing.T) {
	t.Setenv("NUMEN_EMBEDDING_MODEL", "owner/model")
	h := handler{model: newServer(nil)}

	_, infoErr := h.GetModelInfo(t.Context())
	_, embedErr := h.EmbedTexts(t.Context(), embedding.RoleDocument, []string{"text"})

	if rpcerror.GetCode(infoErr) != rpcerror.FailedPrecondition || rpcerror.GetCode(embedErr) != rpcerror.FailedPrecondition {
		t.Fatalf("GetModelInfo error %v and EmbedTexts error %v, want FailedPrecondition", infoErr, embedErr)
	}
}

func TestTheHandlerAnswersWithTheModelTheEnvironmentNames(t *testing.T) {
	t.Setenv("NUMEN_TEST_MOCK_ML", "true")
	h := handler{model: newServer(nil)}

	info, err := h.GetModelInfo(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := h.EmbedTexts(t.Context(), embedding.RoleQuery, []string{"boat", "ferry"})
	if err != nil {
		t.Fatal(err)
	}

	if info.From != "mock" || len(vectors) != 2 || len(vectors[0]) != int(info.Dimensions) {
		t.Fatalf("info %+v with %d vectors", info, len(vectors))
	}
}
