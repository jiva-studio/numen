// Package settings holds the configuration of the agent: the connection it chats through and the
// limits of a task.
package settings

import (
	"encoding/json"
	"strconv"
	"sync"
)

// The keys of the settings Configure reads.
const (
	KeyConnections = "connections"
	KeyTemperature = "temperature"
	KeyMaxTokens   = "maxTokens"
	KeyMaxSteps    = "maxSteps"
)

// Row is the connection the agent chats through, with the model and the reasoning level to ask it for.
type Row struct {
	Connection string `json:"connection"`
	Model      string `json:"model"`
	Reasoning  string `json:"reasoning"`
}

// Settings is the configuration at one moment. HasRow is false while no row names a connection.
type Settings struct {
	Row         Row
	HasRow      bool
	Temperature *float64
	MaxTokens   *int
	MaxSteps    int
}

// GetMaxTokens returns the answer length limit, zero where none is set.
func (s Settings) GetMaxTokens() int {
	if s.MaxTokens == nil {
		return 0
	}
	return *s.MaxTokens
}

// defaultMaxSteps is how many requests to the model one task may make while no limit is configured.
const defaultMaxSteps = 25

// GetMaxSteps returns how many requests to the model one task may make.
func (s Settings) GetMaxSteps() int {
	if s.MaxSteps <= 0 {
		return defaultMaxSteps
	}
	return s.MaxSteps
}

// Store keeps the current settings for readers and writers on different goroutines. The zero value
// is ready.
type Store struct {
	mu      sync.Mutex
	current Settings
}

// NewStore returns a store holding no setting.
func NewStore() *Store {
	return &Store{
		mu:      sync.Mutex{},
		current: Settings{Row: Row{Connection: "", Model: "", Reasoning: ""}, HasRow: false, Temperature: nil, MaxTokens: nil, MaxSteps: 0},
	}
}

// Get returns the current settings.
func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

// Configure applies the values: the row that names the connection, the model and its reasoning level,
// and the limits of a task. A setting absent from the values keeps its current value, and a row
// setting that is no list of rows is left out.
func (s *Store) Configure(values map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if row, hasRow, isRows := readRow(values[KeyConnections]); isRows {
		s.current.Row, s.current.HasRow = row, hasRow
	}
	if text, ok := values[KeyTemperature]; ok && text != "" {
		if val, err := strconv.ParseFloat(text, 64); err == nil {
			s.current.Temperature = &val
		}
	}
	if text, ok := values[KeyMaxTokens]; ok && text != "" {
		if val, err := strconv.Atoi(text); err == nil {
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

func readRow(text string) (row Row, hasRow, isRows bool) {
	if text == "" {
		return Row{Connection: "", Model: "", Reasoning: ""}, false, false
	}
	var rows []Row
	if err := json.Unmarshal([]byte(text), &rows); err != nil {
		return Row{Connection: "", Model: "", Reasoning: ""}, false, false
	}
	if len(rows) == 0 {
		return Row{Connection: "", Model: "", Reasoning: ""}, false, true
	}
	return rows[0], true, true
}
