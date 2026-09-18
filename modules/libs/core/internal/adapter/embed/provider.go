package embed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
)

// Where a vector is made.
const (
	UseLocal   = "local"
	UseService = "service"
)

// What runs a model on this machine. EngineRuntime is ONNX Runtime, the library
// fetched at run time and reached by name. EnginePureGo is the backend written
// in Go, which asks for nothing that is not in the binary.
const (
	EngineRuntime = "onnxruntime"
	EnginePureGo  = "go"
)

// KeyEnvVar is where the service key is read from when the configuration file
// does not carry one.
const KeyEnvVar = "NUMEN_EMBEDDING_KEY"

// The files a model is made of, and where a repository keeps them. ModelFile is
// the build that is run when the configuration names none, and TokenizerFile
// defines the tokeniser in full.
const (
	ModelFile     = "model.onnx"
	TokenizerFile = "tokenizer.json"
	ModelFolder   = "onnx"
)

// Provider is where a vector is made: on this machine, or by a service.
//
// The settings for both are carried, because a file keeps the one it is not on
// and a person moves between them by changing a word. Which of the two is in
// force is Use, and Local and Service are the only way to the settings behind
// it: the half that is not in force describes nothing this installation does,
// and is not read out of here by mistake.
type Provider struct {
	// Use is UseLocal or UseService. Empty is an installation that makes no
	// vector at all.
	Use string

	local   LocalModel
	service ServiceModel
}

// Local is the model this machine runs, and false where a vector is made
// anywhere else.
func (p Provider) Local() (LocalModel, bool) {
	if p.Use != UseLocal {
		return LocalModel{}, false
	}
	return p.local, true
}

// Service is the service a vector is asked of, and false where a vector is made
// anywhere else.
func (p Provider) Service() (ServiceModel, bool) {
	if p.Use != UseService {
		return ServiceModel{}, false
	}
	return p.service, true
}

// SetLocal is this provider making its vectors on this machine with the model
// given, and SetService is it making them at the service given. Each sets the
// word with the settings, so a half is never written without being put in
// force, and each keeps the half it is not on the way the file does.
func (p Provider) SetLocal(m LocalModel) Provider {
	p.Use, p.local = UseLocal, m
	return p
}

func (p Provider) SetService(m ServiceModel) Provider {
	p.Use, p.service = UseService, m
	return p
}

// GetAddress is this provider as the address its vectors are kept under. It
// leads with the word that says which of the two it is, because one name is
// both a repository and something a service answers to.
func (p Provider) GetAddress() string {
	if local, ok := p.Local(); ok {
		return local.GetAddress()
	}
	if service, ok := p.Service(); ok {
		return service.GetAddress()
	}
	return ""
}

// LocalModel is how this machine reaches a model it runs.
type LocalModel struct {
	// Name is a HuggingFace repository.
	Name string `json:"name"`
	// Dir holds the model and tokenizer.json. Empty means the download cache.
	Dir string `json:"dir"`
	// File is the model inside the repository or the directory. Empty is
	// model.onnx, and naming another is how a quantised build of one model is
	// run in place of the full one.
	File string `json:"file"`
	// BatchTexts is how many texts one forward pass carries.
	BatchTexts int `json:"batch_texts"`
	// Engine is EngineRuntime or EnginePureGo. Empty takes ONNX Runtime where
	// this platform has one published, and the Go backend where it has none.
	Engine string `json:"engine"`
	// Runtime is the ONNX Runtime shared library. Empty means the one beside the
	// application, and then the one the platform holds.
	Runtime string `json:"runtime"`
	// Threads is how many of this machine one forward pass may use.
	Threads int `json:"threads"`
	// Download allows fetching the model when it is not on this machine. Turned
	// off, and with no directory named, a vault is searched by its words.
	Download bool `json:"download"`
}

// GetEngine is what runs this model here. A platform ONNX Runtime is published
// for takes it, and one it is not published for takes the Go backend.
func (m LocalModel) GetEngine() string {
	if m.Engine != "" {
		return m.Engine
	}
	if onnxruntime.IsPublished() {
		return EngineRuntime
	}
	return EnginePureGo
}

// GetThreads is how much of this machine one forward pass may use. A
// recognition runs beside this one and is told the same.
func (m LocalModel) GetThreads() int {
	if m.Threads <= 0 {
		return defaultThreads
	}
	return m.Threads
}

// defaultThreads is what a forward pass takes where the settings say nothing.
const defaultThreads = 4

// GetAddress is where this machine reads the weights: the directory when one is
// named, and the repository otherwise, with the file that is run inside it. A
// quantised build is a file of its own and answers with numbers of its own.
//
// The file and the directory are settled to one form, so one set of weights has
// one address whichever way the configuration writes it.
func (m LocalModel) GetAddress() string {
	at := m.Name
	if m.Dir != "" {
		at = filepath.ToSlash(filepath.Clean(m.Dir))
	}
	file := m.File
	if file == "" {
		file = ModelFile
	}
	return UseLocal + ":" + at + "/" + file
}

// ServiceModel is how a hosted model is reached over HTTP.
type ServiceModel struct {
	// BaseURL points at anything speaking the /v1/embeddings request shape.
	BaseURL string `json:"base_url"`
	// Name is what that service calls the model.
	Name string `json:"name"`
	// BatchCharacters bounds one request by the characters of everything in it.
	BatchCharacters int `json:"batch_characters"`
	// KeyEnv names the environment variable holding the key, for an
	// installation that keeps it out of the file.
	KeyEnv string `json:"key_env"`

	// key is unexported: no value a caller formats or serialises carries it.
	// Key reads it.
	key string
}

// GetAddress is the service and the name it is asked for there. Two services
// answering to one name are two models, and the key is no part of this.
func (s ServiceModel) GetAddress() string {
	return UseService + ":" + strings.TrimSuffix(s.BaseURL, "/") + "/" + s.Name
}

// serviceModelFile is the shape on disk, with the key among the fields a person
// writes.
type serviceModelFile struct {
	BaseURL         *string `json:"base_url"`
	Name            *string `json:"name"`
	BatchCharacters *int    `json:"batch_characters"`
	KeyEnv          *string `json:"key_env"`
	// A key is written by a person and never by us.
	Key *string `json:"key,omitempty"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (s *ServiceModel) UnmarshalJSON(raw []byte) error {
	var f serviceModelFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&s.BaseURL, f.BaseURL)
	assign(&s.Name, f.Name)
	assign(&s.BatchCharacters, f.BatchCharacters)
	assign(&s.KeyEnv, f.KeyEnv)
	assign(&s.key, f.Key)
	return nil
}

// MarshalJSON writes everything but the key. Rewriting the file is not how a
// key is set.
func (s ServiceModel) MarshalJSON() ([]byte, error) {
	return json.Marshal(serviceModelFile{
		BaseURL:         &s.BaseURL,
		Name:            &s.Name,
		BatchCharacters: &s.BatchCharacters,
		KeyEnv:          &s.KeyEnv,
	})
}

// String reports the configuration with the key replaced.
func (s ServiceModel) String() string {
	held := "absent"
	if s.Key() != "" {
		held = "present"
	}
	return "service " + s.Name + " at " + s.BaseURL + ", key " + held
}

// Key is the service key: from the file if it names one, otherwise from the
// environment. It is never taken as an argument.
func (s ServiceModel) Key() string {
	if s.key != "" {
		return s.key
	}
	name := s.KeyEnv
	if name == "" {
		name = KeyEnvVar
	}
	return os.Getenv(name)
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (m *LocalModel) UnmarshalJSON(raw []byte) error {
	var f struct {
		Name       *string `json:"name"`
		Dir        *string `json:"dir"`
		File       *string `json:"file"`
		BatchTexts *int    `json:"batch_texts"`
		Engine     *string `json:"engine"`
		Runtime    *string `json:"runtime"`
		Threads    *int    `json:"threads"`
		Download   *bool   `json:"download"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&m.Name, f.Name)
	assign(&m.Dir, f.Dir)
	assign(&m.File, f.File)
	assign(&m.BatchTexts, f.BatchTexts)
	assign(&m.Engine, f.Engine)
	assign(&m.Runtime, f.Runtime)
	assign(&m.Threads, f.Threads)
	assign(&m.Download, f.Download)
	return nil
}

// providerFile is the shape on disk. The keys are written down here because the
// settings behind them are unexported, and a file anybody already has is read
// and written by these three names whatever the fields come to be called.
type providerFile struct {
	// Use is `local` or `service`, and names which of the two sections below is
	// the one in force. Empty makes no vector at all.
	Use *string `json:"use"`

	Local   *LocalModel   `json:"local"`
	Service *ServiceModel `json:"service"`
}

// UnmarshalJSON keeps whatever the defaults set for the fields the file omits.
func (p *Provider) UnmarshalJSON(raw []byte) error {
	f := providerFile{Local: &p.local, Service: &p.service}
	if err := json.Unmarshal(raw, &f); err != nil {
		return err
	}
	assign(&p.Use, f.Use)
	return nil
}

// MarshalJSON writes both halves, the one in force and the one kept: a person
// moving between them by a word finds the settings they left still there.
func (p Provider) MarshalJSON() ([]byte, error) {
	return json.Marshal(providerFile{Use: &p.Use, Local: &p.local, Service: &p.service})
}
