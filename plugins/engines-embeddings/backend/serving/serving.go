// Package serving decides which embedding model the plugin serves from its settings and the
// environment, loads it, and embeds texts with it.
package serving

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/performance"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/chunking"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/limits"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/loading"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/model"
)

// Machine is the process the server runs in.
type Machine interface {
	CountCPUs() int
	SetMaxProcs(count int)
	LowerNice(nice int) error
}

// ProfileSource reads the model profiles. It returns the profiles it could read together with the
// error that kept it from reading the shipped ones.
type ProfileSource interface {
	Load() (*model.Profiles, error)
}

// Environment is what the process environment sets for the model.
type Environment interface {
	GetOverrides() model.Overrides
	GetServiceURL() string
	IsMockModel() bool
	GetMockLogPath() string
	WaitMockOpenHold(ctx context.Context) error
}

// OpenEmbedder loads the model a server embeds with.
type OpenEmbedder func(ctx context.Context, info model.ModelInfo, local model.LocalModel) (loading.Embedder, error)

// NewMockEmbedder returns the stand-in for the model that appends the texts it is given to the file at logPath, when logPath is not empty.
type NewMockEmbedder func(logPath string) loading.Embedder

// FetchVectors asks a remote service for one vector per text.
type FetchVectors func(ctx context.Context, key, baseURL, model string, texts []string) ([][]float32, error)

// Ports is what a server asks of the machine. Report writes a fault nothing else handles.
type Ports struct {
	Machine      Machine
	Profiles     ProfileSource
	Environment  Environment
	OpenLocal    OpenEmbedder
	NewMock      NewMockEmbedder
	FetchVectors FetchVectors
	Report       func(what string, err error)
}

// Server serves one embedding model, local or remote, and follows the settings it is given.
type Server struct {
	machine      Machine
	profileSrc   ProfileSource
	env          Environment
	fetchVectors FetchVectors
	report       func(what string, err error)

	modelInfo   model.ModelInfo
	localModel  model.LocalModel
	localInfo   model.ModelInfo
	localErr    error
	profilesErr error
	localKey    string

	profiles *model.Profiles

	open    OpenEmbedder
	newMock NewMockEmbedder
	isMock  bool
	mockLog string

	chunkTokens int

	mu          sync.Mutex
	loader      *loading.Loader[modelSpec]
	apiKey      string
	baseURL     string
	provider    string
	localName   string
	remoteModel string

	profile      performance.Profile
	isProfileSet bool
}

// modelSpec is the model one load opens, with the function that opens it.
type modelSpec struct {
	open  OpenEmbedder
	info  model.ModelInfo
	local model.LocalModel
}

// New returns a server with the model the environment names, or the default model.
func New(ports Ports) *Server {
	var none model.ModelInfo
	var noLocal model.LocalModel
	s := &Server{
		machine: ports.Machine, profileSrc: ports.Profiles, env: ports.Environment,
		fetchVectors: ports.FetchVectors, report: ports.Report,
		modelInfo: none, localModel: noLocal, localInfo: none, localErr: nil,
		profilesErr: nil, localKey: "", profiles: nil, mockLog: "", localName: "",
		open:        ports.OpenLocal,
		newMock:     ports.NewMock,
		isMock:      false,
		chunkTokens: chunking.DefaultTokens,
		mu:          sync.Mutex{},
		loader:      nil,
		apiKey:      "",
		baseURL:     "",
		provider:    "",
		remoteModel: "",
		profile:     performance.Balanced, isProfileSet: false,
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

func (s *Server) newLoader() *loading.Loader[modelSpec] {
	return loading.New(&s.mu, loading.Ports[modelSpec]{
		Open: func(ctx context.Context, spec modelSpec) (loading.Embedder, error) {
			return spec.open(ctx, spec.info, spec.local)
		},
		LowerNice: func(nice int) error { return s.machine.LowerNice(nice) },
		Report:    s.report,
		Opened:    func(emb loading.Embedder) { s.modelInfo.Concurrency = emb.Concurrency() },
	})
}

func (s *Server) getEmbedder(ctx context.Context) (loading.Embedder, error) {
	return s.loader.Get(ctx, s.getStartLocked)
}

func (s *Server) getStartLocked() loading.Start[modelSpec] {
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
	return loading.Start[modelSpec]{Spec: modelSpec{open: s.open, info: info, local: local}, Nice: run.Nice}
}

func (s *Server) getLimitsLocked() limits.Run {
	return limits.GetRun(s.profile, s.machine.CountCPUs(), model.GetDefaultBatchTexts())
}

func (s *Server) applyProfileLocked(profile performance.Profile) {
	if s.isProfileSet && profile == s.profile {
		return
	}
	before := s.getLimitsLocked()
	s.profile, s.isProfileSet = profile, true
	run := s.getLimitsLocked()
	// gomutants:disable-next-line CONDITIONALS_NEGATION,BRANCH_IF reason="stderr log on priority error has no side effect"
	if err := s.machine.LowerNice(run.Nice); err != nil {
		s.report("set the priority of the embeddings plugin", err)
	}
	if s.isMock {
		return
	}
	s.machine.SetMaxProcs(run.Threads)
	if before != run {
		s.loader.ResetLocked()
	}
}

func (s *Server) openMockModel(ctx context.Context, _ model.ModelInfo, _ model.LocalModel) (loading.Embedder, error) {
	if err := s.env.WaitMockOpenHold(ctx); err != nil {
		return nil, err
	}
	return s.newMock(s.mockLog), nil
}

// Close ends the load of the model and closes the loaded model.
func (s *Server) Close() error {
	return s.loader.Close()
}
