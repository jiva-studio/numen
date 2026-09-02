// Package proofreading is what a person configured to put a reading right.
//
// Nothing here is reached unless a person named a profile. An installation that
// named none uses a reading exactly as it was read, and asks for no key and no
// network.
package proofreading

import (
	"encoding/json"
	"os"
)

// What a profile is reached through. UseService is anything speaking the
// /v1/chat/completions request shape; UseAgent is the `claude` command line.
const (
	UseService = "service"
	UseAgent   = "agent"
)

// KeyEnvVar is where a service key is read from when the configuration file
// does not carry one.
const KeyEnvVar = "NUMEN_PROOFREADING_KEY"

// DefaultLettersApart is how far a correction may move a line's letters where
// nobody said. It was measured on one book and the next book is not that one.
const DefaultLettersApart = 0.30

// DefaultBatchSize is how many lines one request carries where nobody said.
const DefaultBatchSize = 40

// DefaultInFlight is how many batches are asked about at once where nobody
// said. A service is asked over four connections; the command line is a
// person's own, and is left most of itself while they are using it.
const (
	DefaultInFlight      = 4
	DefaultAgentInFlight = 2
)

// DefaultAgentBatchSize is how many lines one run of the command line carries.
// Starting it costs the same whatever it is asked, so it is asked a lot.
const DefaultAgentBatchSize = 60

// Config is the proofreading section of this installation's settings: the
// profiles a reading may be put right at, and the one threshold they are all
// held to.
type Config struct {
	// LettersApart is how far a correction may move a line's letters and still
	// be a correction, as a share of the longer of the two. It stands over
	// every profile: it is a property of the text, not of a transport.
	LettersApart float64 `json:"letters_apart"`

	// Profiles are the stations a reading is put right at, by the name a
	// consumer asks for one under. An installation naming none proofreads
	// nothing.
	Profiles map[string]Profile `json:"profiles"`
}

// Profile is one station. Every key stands at this level and Use says which of
// them apply; a key Use does not apply to is ignored.
type Profile struct {
	// Use is `service` or `agent`.
	Use string `json:"use"`

	// BaseURL points at anything speaking the /v1/chat/completions request
	// shape.
	BaseURL string `json:"base_url"`
	// BatchURL is a queue the batches are left in and collected from later, at
	// half the price. Empty asks a batch at a time and waits.
	BatchURL string `json:"batch_url"`
	// Name is which model corrects a reading at a service. It stands beside
	// every line it corrected.
	Name string `json:"name"`
	// KeyEnv names the environment variable holding the key, for an
	// installation that keeps it out of the file.
	KeyEnv string `json:"key_env"`

	// Model is which model corrects a reading at the command line, by the name
	// the command line knows it as.
	Model string `json:"model"`
	// Command starts the command line. Empty means `claude` from the path.
	Command []string `json:"command"`

	// BatchSize is how many lines one request carries.
	BatchSize int `json:"batch_size"`
	// Overlap is how many lines neighbouring batches share. A line two batches
	// both answered about is taken from the later one, which saw more of what
	// follows it.
	Overlap int `json:"overlap"`
	// InFlight is how many batches are being asked about at any moment. At the
	// command line this is how much of a person's own model is taken while
	// they are using it.
	InFlight int `json:"in_flight"`

	// key is unexported: no value a caller formats or serialises carries it.
	// Key reads it.
	key string
}

// Proofread is how one kind of reading is put right: the profile that does it,
// and whether that happens without anybody asking.
type Proofread struct {
	// With is the profile, by the name the profiles carry it under. Empty
	// names none, and nothing is put right.
	With string `json:"with"`
	// Automatically is whether a reading already written down is put right
	// without anybody asking for it.
	Automatically bool `json:"automatically"`
}

// Defaults proofread nothing. The models are there and a person asks, which is
// what recognition already does.
func Defaults() Config {
	return Config{LettersApart: DefaultLettersApart}
}

// ServiceDefaults are what a service profile keeps for the fields it leaves
// out.
func ServiceDefaults() Profile {
	return Profile{
		Use:       UseService,
		BaseURL:   "https://openrouter.ai/api/v1",
		BatchURL:  "https://openrouter.ai/api/beta/batches",
		KeyEnv:    KeyEnvVar,
		BatchSize: DefaultBatchSize,
		InFlight:  DefaultInFlight,
	}
}

// AgentDefaults are what an agent profile keeps for the fields it leaves out.
func AgentDefaults() Profile {
	return Profile{
		Use:       UseAgent,
		BatchSize: DefaultAgentBatchSize,
		InFlight:  DefaultAgentInFlight,
	}
}

// Named says whether this installation asked for anything to proofread with.
func (c Config) Named() bool { return len(c.Profiles) > 0 }

// Apart is how far a correction may move a line's letters. A file naming
// nothing takes what was measured.
func (c Config) Apart() float64 {
	if c.LettersApart <= 0 {
		return DefaultLettersApart
	}
	return c.LettersApart
}

// Named says whether this profile carries a model to correct a reading with.
func (p Profile) Named() bool {
	switch p.Use {
	case UseService:
		return p.Name != ""
	case UseAgent:
		return p.Model != ""
	}
	return false
}

// configFile is the shape on disk, with every field a pointer so that what the
// file omits keeps its default.
type configFile struct {
	LettersApart *float64           `json:"letters_apart"`
	Profiles     map[string]Profile `json:"profiles"`
}

// profileFile is one profile on disk, with the key among the fields a person
// writes.
type profileFile struct {
	Use      *string `json:"use"`
	BaseURL  *string `json:"base_url,omitempty"`
	BatchURL *string `json:"batch_url,omitempty"`
	Name     *string `json:"name,omitempty"`
	KeyEnv   *string `json:"key_env,omitempty"`
	// A key is written by a person and never by us.
	Key       *string   `json:"key,omitempty"`
	Model     *string   `json:"model,omitempty"`
	Command   *[]string `json:"command,omitempty"`
	BatchSize *int      `json:"batch_size"`
	Overlap   *int      `json:"overlap"`
	InFlight  *int      `json:"in_flight"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (c *Config) UnmarshalJSON(raw []byte) error {
	var f configFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	*c = Defaults()
	assign(&c.LettersApart, f.LettersApart)
	c.Profiles = f.Profiles
	return nil
}

// UnmarshalJSON starts from the defaults of the kind the profile says it is.
func (p *Profile) UnmarshalJSON(raw []byte) error {
	var f profileFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}

	var use string
	assign(&use, f.Use)
	switch use {
	case UseService:
		*p = ServiceDefaults()
	case UseAgent:
		*p = AgentDefaults()
	default:
		*p = Profile{Use: use}
	}

	assign(&p.BaseURL, f.BaseURL)
	assign(&p.BatchURL, f.BatchURL)
	assign(&p.Name, f.Name)
	assign(&p.KeyEnv, f.KeyEnv)
	assign(&p.key, f.Key)
	assign(&p.Model, f.Model)
	assign(&p.Command, f.Command)
	assign(&p.BatchSize, f.BatchSize)
	assign(&p.Overlap, f.Overlap)
	assign(&p.InFlight, f.InFlight)
	return nil
}

// MarshalJSON writes what this profile's use applies to, and never the key.
// Rewriting the file is not how a key is set.
func (p Profile) MarshalJSON() ([]byte, error) {
	f := profileFile{Use: &p.Use, BatchSize: &p.BatchSize, Overlap: &p.Overlap, InFlight: &p.InFlight}
	switch p.Use {
	case UseService:
		f.BaseURL, f.BatchURL, f.Name, f.KeyEnv = &p.BaseURL, &p.BatchURL, &p.Name, &p.KeyEnv
	case UseAgent:
		f.Model = &p.Model
		if len(p.Command) > 0 {
			f.Command = &p.Command
		}
	}
	return json.Marshal(f)
}

// String reports the profile with the key replaced. A value that formats itself
// cannot be logged into a file by accident.
func (p Profile) String() string {
	if p.Use == UseAgent {
		return "the command line, model " + p.Model
	}
	held := "absent"
	if p.Key() != "" {
		held = "present"
	}
	return "service " + p.Name + " at " + p.BaseURL + ", key " + held
}

// Key is the service key: from the file if it names one, otherwise from the
// environment. It is never taken as an argument, so there is no call site at
// which it could be written down.
func (p Profile) Key() string {
	if p.key != "" {
		return p.key
	}
	name := p.KeyEnv
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
