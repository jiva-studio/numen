// Package proofreading is what a person configured to put a reading right.
//
// Nothing here is reached unless a person named a model. An installation that
// named none uses a reading exactly as it was read, and asks for no key and no
// network.
package proofreading

import (
	"encoding/json"
	"os"
)

// Which proofreader an installation uses.
const UseService = "service"

// KeyEnvVar is where the service key is read from when the configuration file
// does not carry one.
const KeyEnvVar = "NUMEN_PROOFREADING_KEY"

// Config is the proofreading section of this installation's settings: whether
// anything proofreads at all, and what it is reached through.
type Config struct {
	Use     string  `json:"use"`
	Service Service `json:"service"`
}

// Service is a hosted model reached over HTTP.
type Service struct {
	BaseURL string `json:"base_url"`
	// BatchURL is a queue the pages are left in and collected from later, at
	// half the price. Empty asks a page at a time and waits.
	BatchURL string `json:"batch_url"`
	Name     string `json:"name"`
	// KeyEnv names the environment variable holding the key, for an
	// installation that keeps it out of the file.
	KeyEnv string `json:"key_env"`
	// PagesAtOnce is how many pages one request carries.
	PagesAtOnce int `json:"pages_at_once"`
	// LettersApart is how far a correction may move a line's letters and still
	// be a correction, as a share of the longer of the two. It is here because
	// it was measured on one book and the next book is not that one.
	LettersApart float64 `json:"letters_apart"`

	// key is unexported: no value a caller formats or serialises carries it.
	// Key reads it.
	key string
}

// Defaults proofread nothing. The models are there and a person asks, which is
// what recognition already does.
func Defaults() Config {
	return Config{
		Service: Service{
			BaseURL:      "https://openrouter.ai/api/v1",
			BatchURL:     "https://openrouter.ai/api/beta/batches",
			KeyEnv:       KeyEnvVar,
			PagesAtOnce:  40,
			LettersApart: 0.30,
		},
	}
}

// Named says whether this installation asked for anything to proofread with.
func (c Config) Named() bool {
	return c.Use == UseService && c.Service.Name != ""
}

// serviceFile is the shape on disk, with the key among the fields a person
// writes.
type serviceFile struct {
	BaseURL      *string  `json:"base_url"`
	BatchURL     *string  `json:"batch_url"`
	Name         *string  `json:"name"`
	KeyEnv       *string  `json:"key_env"`
	Key          *string  `json:"key"`
	PagesAtOnce  *int     `json:"pages_at_once"`
	LettersApart *float64 `json:"letters_apart"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (s *Service) UnmarshalJSON(raw []byte) error {
	var f serviceFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&s.BaseURL, f.BaseURL)
	assign(&s.BatchURL, f.BatchURL)
	assign(&s.Name, f.Name)
	assign(&s.KeyEnv, f.KeyEnv)
	assign(&s.key, f.Key)
	assign(&s.PagesAtOnce, f.PagesAtOnce)
	assign(&s.LettersApart, f.LettersApart)
	return nil
}

// MarshalJSON writes everything but the key. Rewriting the file is not how a
// key is set.
func (s Service) MarshalJSON() ([]byte, error) {
	return json.Marshal(serviceFile{
		BaseURL:      &s.BaseURL,
		BatchURL:     &s.BatchURL,
		Name:         &s.Name,
		KeyEnv:       &s.KeyEnv,
		PagesAtOnce:  &s.PagesAtOnce,
		LettersApart: &s.LettersApart,
	})
}

// String reports the configuration with the key replaced. A value that formats
// itself cannot be logged into a file by accident.
func (s Service) String() string {
	held := "absent"
	if s.Key() != "" {
		held = "present"
	}
	return "service " + s.Name + " at " + s.BaseURL + ", key " + held
}

// Key is the service key: from the file if it names one, otherwise from the
// environment. It is never taken as an argument, so there is no call site at
// which it could be written down.
func (s Service) Key() string {
	if s.key != "" {
		return s.key
	}
	name := s.KeyEnv
	if name == "" {
		name = KeyEnvVar
	}
	return os.Getenv(name)
}

func assign[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}
