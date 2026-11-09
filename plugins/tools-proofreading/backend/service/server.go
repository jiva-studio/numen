// Package service is the backend of the proofreading plugin: it reads a text
// version of a file, sends its lines to a model for correction, stores the corrected
// text in the plugin's folder of the vault's store, and reports it to the host
// as a version derived from the one it read.
package service

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/hostcall"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/server"
)

// PluginID is the identity the manifest gives the plugin.
const PluginID = "md.numen.tools.proofreading"

// EnvMockML names the variable that has the plugin answer with the mock proofreader where its settings hold
// no connection; with a connection the plugin asks it.
const EnvMockML = "NUMEN_MOCK_ML"

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// completer asks the connections of the plugin to complete a request.
type completer func(ctx context.Context, request connection.CompleteRequest) (connection.CompleteResult, error)

// errNoConnection means the settings of the plugin hold no connection that can complete.
var errNoConnection = errors.New("proofreading is off: add a connection on the Connections page, then choose it in the settings of LLM Proofreading")

// Host provides the plugin's host services: the store folder for corrected
// texts, the text versions it reads and reports, and the queue it runs in.
type Host struct {
	Store host.Store
	Texts host.Texts
	Queue host.Queue
}

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

// mockEnabled is the value of EnvMockML that selects the mock model.
const mockEnabled = "true"

// settings is the configuration the host last passed.
type settings struct {
	mu  sync.RWMutex
	cfg proofreadingConfig
}

func newSettings() *settings {
	return &settings{mu: sync.RWMutex{}, cfg: defaultProofreadingConfig()}
}

func (s *settings) updateConfig(values map[string]string, _ server.Secrets) {
	s.mu.Lock()
	defer s.mu.Unlock()
	applyIntConfig(values, "batchSize", &s.cfg.batchSize, 1)
	applyIntConfig(values, "overlap", &s.cfg.overlap, 0)
	applyIntConfig(values, "inFlight", &s.cfg.inFlight, 1)
	if v, ok := values["maxEditDistance"]; ok {
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

func newAskModel(complete completer) askModel {
	return func(ctx context.Context, instruction, prompt string) (string, error) {
		result, err := complete(ctx, connection.CompleteRequest{
			Messages: []connection.Message{
				{Role: "system", Text: instruction},
				{Role: "user", Text: prompt},
			},
			Temperature: nil,
			MaxTokens:   0,
			Priority:    connection.PriorityInteractive,
		})
		if errors.Is(err, connection.ErrNoConnection) {
			return "", errNoConnection
		}
		return result.Text, err
	}
}

func completeOrMock(complete completer) completer {
	return func(ctx context.Context, request connection.CompleteRequest) (connection.CompleteResult, error) {
		result, err := complete(ctx, request)
		if errors.Is(err, connection.ErrNoConnection) {
			return completeMock(ctx, request)
		}
		return result, err
	}
}

// Mount registers the proofread and delete commands on s, using the
// host that the environment names. The model is the mock proofreader when the
// environment enables mock models.
func Mount(s *server.Server) error {
	endpoint, err := hostcall.FromEnv()
	if err != nil {
		return err
	}
	client, err := connection.NewClientFromEnv()
	if err != nil {
		return err
	}
	complete := completer(client.Complete)
	if os.Getenv(EnvMockML) == mockEnabled {
		complete = completeOrMock(complete)
	}
	return mount(s, openHostAt(endpoint), complete, nil)
}

// NewHandler returns the handler of the plugin's backend for a host that links
// the plugin in, configured from the variables given to a launched plugin
// process. It uses the host that those variables name.
func NewHandler(vars map[string]string) (http.Handler, error) {
	endpoint, err := hostcall.FromVars(vars)
	if err != nil {
		return nil, err
	}
	client, err := connection.NewClientFromVars(vars)
	if err != nil {
		return nil, err
	}
	complete := completer(client.Complete)
	if vars[EnvMockML] == mockEnabled {
		complete = completeOrMock(complete)
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
		config.updateConfig(vars, nil)
	}
	run := newProofreading(h, getProofreadKinds(), newAskModel(complete), config.getConfig)
	commands, err := newCommands(run)
	if err != nil {
		return err
	}
	s.SetUpdateConfig(config.updateConfig)
	command.Mount(s, commands)
	return nil
}

func newCommands(run *Proofreading) (*command.Registry, error) {
	commands := command.New()
	if err := commands.Add("proofreading.create", newProofreadCommand(run)); err != nil {
		return nil, err
	}
	return commands, nil
}
