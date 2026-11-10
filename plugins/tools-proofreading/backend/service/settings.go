package service

import (
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/internal/adapter/model"
	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/proofread"
)

// config is what the host last passed: how the model is reached and how a text is proofread.
type config struct {
	endpoint model.Endpoint
	run      proofread.Config
}

// settings is the configuration the host last passed.
type settings struct {
	mu  sync.RWMutex
	cfg config
}

func newSettings() *settings {
	return &settings{mu: sync.RWMutex{}, cfg: config{endpoint: model.DefaultEndpoint(), run: proofread.DefaultConfig()}}
}

func (s *settings) updateConfig(values map[string]string, secrets server.Secrets) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if provider := values["provider"]; provider != "" {
		s.cfg.endpoint.Provider = provider
	}
	if baseURL, ok := values["baseUrl"]; ok {
		s.cfg.endpoint.BaseURL = baseURL
	}
	if name := values["model"]; name != "" {
		s.cfg.endpoint.Model = name
	}
	if list, carried := server.ReadSecrets(secrets, server.SecretAPIKey); carried {
		s.cfg.endpoint.APIKey = ""
		if len(list) > 0 {
			s.cfg.endpoint.APIKey = list[0].Value
		}
	}
	applyIntConfig(values, "batchSize", &s.cfg.run.BatchSize, 1)
	applyIntConfig(values, "overlap", &s.cfg.run.Overlap, 0)
	applyIntConfig(values, "inFlight", &s.cfg.run.InFlight, 1)
	if v, ok := values["maxEditDistance"]; ok {
		if d, err := strconv.ParseFloat(v, 64); err == nil && d >= 0 {
			s.cfg.run.MaxEditDistance = d
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

func (s *settings) getConfig() config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg
}

func (s *settings) getEndpoint() model.Endpoint { return s.getConfig().endpoint }

func (s *settings) getRunConfig() proofread.Config { return s.getConfig().run }
