package service

import (
	"context"
	"net/http"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/embedding"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/engine"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/environment"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
)

func newLocalModelOpener(library *onnx.Library) openEmbedder {
	// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
	return func(ctx context.Context, info engine.ModelInfo, local engine.LocalModel) (loading.Embedder, error) {
		emb, err := engine.Open(ctx, library, info, local, nil)
		// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="production opener wraps engine.Open with native ONNX runtime"
		if err != nil {
			// gomutants:disable-next-line RETURN_ERROR_NIL reason="production opener wraps engine.Open with native ONNX runtime"
			return nil, err
		}
		// gomutants:disable-next-line RETURN_ZERO reason="production opener wraps engine.Open with native ONNX runtime"
		return emb, nil
	}
}

func newEmbeddingServer() *embeddingServer {
	var none engine.ModelInfo
	var noLocal engine.LocalModel
	s := &embeddingServer{
		modelInfo: none, localModel: noLocal, localInfo: none, localErr: nil,
		profilesErr: nil, localKey: "", profiles: nil, mockLog: "", localName: "",
		profilesDir: getProfilesDir(),
		open:        newLocalModelOpener(onnx.NewLibrary()),
		isMock:      false,
		chunkTokens: chunking.DefaultTokens,
		mu:          sync.Mutex{},
		loader:      nil,
		apiKey:      "",
		baseURL:     "",
		provider:    "",
		remoteModel: "",
		profile:     performance.Balanced, isProfileSet: false, lowerNice: performance.SetNice,
	}
	s.loader = s.newLoader()
	s.loadProfilesLocked()
	if environment.IsMockModel() {
		s.useMockModel()
		return s
	}
	s.updateLocalModelLocked("")
	s.modelInfo = s.localInfo
	s.modelInfo.MinChunkLength = chunking.DefaultMinLength
	s.modelInfo.ChunkTokens = chunking.ClampTokens(chunking.DefaultTokens, s.modelInfo.MaxTokens)
	return s
}

func (s *embeddingServer) useMockModel() {
	s.modelInfo = engine.GetMockModel()
	s.modelInfo.MinChunkLength = chunking.DefaultMinLength
	s.modelInfo.ChunkTokens = chunking.ClampTokens(chunking.DefaultTokens, s.modelInfo.MaxTokens)
	s.isMock = true
	s.mockLog = environment.GetMockLogPath()
	s.localName = s.modelInfo.Name
	s.open = s.openMockModel
}

// Mount registers the embedding service on s, with the model the environment configures. The
// embedder closes with s.
func Mount(s *server.Server) error {
	embedSrv := newEmbeddingServer()
	s.SetUpdateConfig(embedSrv.updateConfig)
	embedding.Mount(s, embedSrv)
	// gomutants:disable-next-line STATEMENT_REMOVE reason="server shutdown callback closes internal embedSrv state"
	s.OnClose(embedSrv.Close)
	return nil
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given.
func NewHandler(vars map[string]string) (http.Handler, error) {
	return server.NewLinkedHandler(vars, Mount)
}
