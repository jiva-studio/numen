// Package settings holds the configuration of the agent: the chat API to call, the model, and the
// limits of a task.
package settings

import (
	"strconv"
	"sync"
)

// The keys of the settings Configure reads.
const (
	KeyBaseURL     = "baseUrl"
	KeyModel       = "model"
	KeyAPIKey      = "apiKey"
	KeyTemperature = "temperature"
	KeyMaxTokens   = "maxTokens"
	KeyMaxSteps    = "maxSteps"
)

// DefaultModel is the model used when neither the task nor the settings specify one.
const DefaultModel = "openai/gpt-4o-mini"

// Settings is the configuration at one moment. Key is empty for a chat API that requires none, and
// BaseURL is empty where the default chat API is meant.
type Settings struct {
	BaseURL     string
	Key         string
	Model       string
	Temperature *float64
	MaxTokens   *int
	MaxSteps    int
}

// GetModel returns the model a task names, else the configured one, else the default.
func (s Settings) GetModel(named string) string {
	if named != "" {
		return named
	}
	if s.Model != "" {
		return s.Model
	}
	return DefaultModel
}

// GetMaxSteps returns the limit a task names, else the configured one; zero means no limit is set.
func (s Settings) GetMaxSteps(requested int) int {
	if requested > 0 {
		return requested
	}
	return s.MaxSteps
}

// Store keeps the current settings for readers and writers on different goroutines. The zero value
// is not ready; NewStore returns one.
type Store struct {
	mu      sync.Mutex
	current Settings
}

// NewStore returns a store holding no setting.
func NewStore() *Store {
	return &Store{
		mu:      sync.Mutex{},
		current: Settings{BaseURL: "", Key: "", Model: "", Temperature: nil, MaxTokens: nil, MaxSteps: 0},
	}
}

// Get returns the current settings.
func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// SetKey replaces the key sent to the chat API.
func (s *Store) SetKey(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.current.Key = key
}

// Configure applies the values: the base URL of the chat API, the model used when a task names
// none, and the limits. A setting absent from the values keeps its current value.
func (s *Store) Configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if base, ok := values[KeyBaseURL]; ok {
		s.current.BaseURL = base
	}
	if name, ok := values[KeyModel]; ok {
		s.current.Model = name
	}
	if tempStr, ok := values[KeyTemperature]; ok && tempStr != "" {
		if val, err := strconv.ParseFloat(tempStr, 64); err == nil {
			s.current.Temperature = &val
		}
	}
	if maxStr, ok := values[KeyMaxTokens]; ok && maxStr != "" {
		if val, err := strconv.Atoi(maxStr); err == nil {
			s.current.MaxTokens = &val
		}
	}
	if val, ok := readPositiveInt(values[KeyMaxSteps]); ok {
		s.current.MaxSteps = val
	}
}

func readPositiveInt(text string) (int, bool) {
	val, err := strconv.Atoi(text)
	return val, err == nil && val > 0
}
