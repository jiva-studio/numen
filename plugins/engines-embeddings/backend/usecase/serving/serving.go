// Package serving decides which embedding model the plugin serves from its settings and the
// environment, loads it, and embeds texts with it.
package serving

import (
	"context"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/ports"
)

// Ports is what a server asks of the machine. Report writes a fault nothing else handles.
type Ports struct {
	Queue       host.InferenceQueue
	Machine     ports.Machine
	Profiles    ports.ProfileSource
	Environment ports.Environment
	OpenLocal   ports.OpenEmbedder
	NewMock     ports.NewMockEmbedder
	OpenModels  ports.OpenModels
	Report      func(what string, err error)
}

// Server serves one embedding model, a local one or the one a connection answers with, and follows the
// settings it is given.
type Server struct {
	queue      host.InferenceQueue
	machine    ports.Machine
	profileSrc ports.ProfileSource
	env        ports.Environment
	report     func(what string, err error)

	modelInfo   model.ModelInfo
	localModel  model.LocalModel
	localInfo   model.ModelInfo
	localErr    error
	profilesErr error
	localKey    string

	profiles *model.Profiles

	open    ports.OpenEmbedder
	newMock ports.NewMockEmbedder
	isMock  bool
	mockLog string

	chunkTokens int

	mu         sync.Mutex
	loader     *modelLoader[modelSpec]
	localName  string
	source     string
	connection *connectionSource

	limits       performance.Limits
	isProfileSet bool
}

// modelSpec is the model one load opens, with the function that opens it.
type modelSpec struct {
	open  ports.OpenEmbedder
	info  model.ModelInfo
	local model.LocalModel
}

// New returns a server with the model the environment names, or the default model.
func New(ports Ports) *Server {
	var none model.ModelInfo
	var noLocal model.LocalModel
	s := &Server{
		machine: ports.Machine, profileSrc: ports.Profiles, env: ports.Environment,
		report: ports.Report, queue: ports.Queue,
		modelInfo: none, localModel: noLocal, localInfo: none, localErr: nil,
		profilesErr: nil, localKey: "", profiles: nil, mockLog: "", localName: "",
		open:        ports.OpenLocal,
		newMock:     ports.NewMock,
		isMock:      false,
		chunkTokens: chunking.DefaultTokens,
		mu:          sync.Mutex{},
		loader:      nil,
		source:      SourceLocal,
		connection:  newConnectionSource(ports.OpenModels),
		limits:      performance.Read(nil), isProfileSet: false,
	}
	s.loader = s.newLoader()
	s.loadProfilesLocked()
	if s.env.IsMockModel() {
		s.useMockModel()
		return s
	}
	s.updateLocalModelLocked("")
	s.modelInfo = s.localInfo
	s.modelInfo.MinChunkLength = chunking.DefaultMinLength
	s.modelInfo.ChunkTokens = chunking.ClampTokens(chunking.DefaultTokens, s.modelInfo.MaxTokens)
	return s
}

func (s *Server) useMockModel() {
	s.modelInfo = model.GetMockModel()
	s.modelInfo.MinChunkLength = chunking.DefaultMinLength
	s.modelInfo.ChunkTokens = chunking.ClampTokens(chunking.DefaultTokens, s.modelInfo.MaxTokens)
	s.isMock = true
	s.mockLog = s.env.GetMockLogPath()
	s.localName = s.modelInfo.Name
	s.open = s.openMockModel
}

func (s *Server) newLoader() *modelLoader[modelSpec] {
	return newModelLoader(&s.mu, loaderPorts[modelSpec]{
		Open: func(ctx context.Context, spec modelSpec) (ports.Embedder, error) {
			return spec.open(ctx, spec.info, spec.local)
		},
		Report: s.report,
		NotifyOpened: func(emb ports.Embedder) { s.modelInfo.Concurrency = emb.Concurrency() },
	})
}

func (s *Server) getEmbedder(ctx context.Context) (ports.Embedder, error) {
	return s.loader.Get(ctx, s.getStartLocked)
}

func (s *Server) getStartLocked() modelSpec {
	info := s.modelInfo
	if !s.isMock {
		info = s.localInfo
	}
	local := s.localModel
	run := s.getLimitsLocked()
	if local.Threads == 0 {
		local.Threads = run.Threads
	}
	local.BatchTexts = run.BatchTexts
	return modelSpec{open: s.open, info: info, local: local}
}

func (s *Server) getLimitsLocked() limits.Run {
	return limits.GetRun(s.limits, model.GetDefaultBatchTexts())
}

func (s *Server) applyProfileLocked(next performance.Limits) {
	if s.isProfileSet && next == s.limits {
		return
	}
	before := s.getLimitsLocked()
	s.limits, s.isProfileSet = next, true
	run := s.getLimitsLocked()
	if s.isMock {
		return
	}
	s.machine.SetMaxProcs(run.Threads)
	if before != run {
		s.loader.ResetLocked()
	}
}

func (s *Server) openMockModel(ctx context.Context, _ model.ModelInfo, _ model.LocalModel) (ports.Embedder, error) {
	if err := s.env.WaitMockOpenHold(ctx); err != nil {
		return nil, err
	}
	return s.newMock(s.mockLog), nil
}

// Close ends the load of the model and closes the loaded model.
func (s *Server) Close() error {
	return s.loader.Close()
}
