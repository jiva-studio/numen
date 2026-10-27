// Package service is the backend of the proofreading plugin: it takes a text version of a file,
// asks a model to put its lines right, keeps the corrected text in the plugin's own folder of the
// vault's store, and reports it to the host as a version made from the one it read.
package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/ai"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
	numenv1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
	proofreading "github.com/jiva-studio/numen/plugins/tools-proofreading"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.proofreading"

// EnvMockML names the variable that has the plugin answer every question with the mock proofreader.
const EnvMockML = "NUMEN_MOCK_ML"

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// purpose is what every question to the model is for. It routes nothing.
const purpose = "proofreading"

// completer asks the host for the answer to a question.
type completer func(ctx context.Context, question *numenv1.CompletionQuestion) (string, error)

// Host is what the plugin works through the host: the plugin's folder the corrected texts are kept
// in, the text versions it reads and reports, and the queue it runs in.
type Host struct {
	Store host.Store
	Texts host.Texts
	Queue host.Queue
}

// openHostAt reaches the host at the endpoint.
func openHostAt(endpoint hostcall.Endpoint) Host {
	return Host{
		Store: host.NewStoreClient(endpoint),
		Texts: host.NewTextClient(endpoint),
		Queue: host.NewQueueClient(endpoint),
	}
}

type proofreadingConfig struct {
	batchSize       int
	overlap         int
	inFlight        int
	maxEditDistance float64
}

func defaultProofreadingConfig() proofreadingConfig {
	return proofreadingConfig{
		batchSize:       DefaultBatchSize,
		overlap:         DefaultOverlap,
		inFlight:        DefaultInFlight,
		maxEditDistance: MaxEditDistance,
	}
}

// settings is the configuration the host last handed over.
type settings struct {
	mu  sync.RWMutex
	cfg proofreadingConfig
}

func newSettings() *settings {
	return &settings{cfg: defaultProofreadingConfig()}
}

func (s *settings) updateConfig(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	applyIntConfig(values, "batch_size", &s.cfg.batchSize, 1)
	applyIntConfig(values, "overlap", &s.cfg.overlap, 0)
	applyIntConfig(values, "in_flight", &s.cfg.inFlight, 1)
	if v, ok := values["max_edit_distance"]; ok {
		if d, err := strconv.ParseFloat(v, 64); err == nil && d >= 0 {
			s.cfg.maxEditDistance = d
		}
	}
}

func applyIntConfig(values map[string]string, key string, target *int, minVal int) {
	if v, ok := values[key]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= minVal {
			*target = n
		}
	}
}

func (s *settings) getConfig() proofreadingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

// newAskModel returns the model as a system instruction and a prompt are put to it.
func newAskModel(complete completer) askModel {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		return complete(ctx, &numenv1.CompletionQuestion{
			Purpose: purpose,
			Messages: []*numenv1.CompletionMessage{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
		})
	}
}

// Mount registers the text provider and the proofread command on s, working the host the
// environment names. The model is the mock proofreader where the environment asks for mock models.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	complete := completer(ai.Complete)
	if os.Getenv(EnvMockML) == "true" {
		complete = completeMock
	}
	return mount(s, openHostAt(endpoint), complete, nil)
}

// NewHandler returns the handler of the plugin's backend for a host that links the plugin in, made
// from the variables a launched process of the plugin is given. It asks the host those name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	complete := func(ctx context.Context, question *numenv1.CompletionQuestion) (string, error) {
		client, err := ai.NewClientFromVars(vars)
		if err != nil {
			return "", err
		}
		return client.Complete(ctx, question)
	}
	if vars[EnvMockML] == "true" {
		complete = completeMock
	}
	hosted := openHostAt(endpoint)
	return server.NewLinkedHandler(vars, func(s *server.Server) error {
		return mount(s, hosted, complete, vars)
	})
}

var errIncompleteHost = errors.New("proofreading needs the store, the texts and the queue of the host")

func mount(s *server.Server, h Host, complete completer, vars map[string]string) error {
	if h.Store == nil || h.Texts == nil || h.Queue == nil {
		return errIncompleteHost
	}
	config := newSettings()
	if len(vars) > 0 {
		config.updateConfig(vars)
	}
	kinds, err := readKinds(proofreading.Manifest)
	if err != nil {
		return err
	}
	run := newProofreading(h, kinds, newAskModel(complete), config.getConfig)
	provider := newProvider(run)
	commands := command.New()
	if err := commands.Add("proofread", newProofreadCommand(run)); err != nil {
		return err
	}
	s.SetUpdateConfig(config.updateConfig)
	s.Register(host.NewTextProviderHandler(provider, s.HandlerOptions()...))
	command.Mount(s, commands)
	s.OnClose(provider.Close)
	return nil
}
