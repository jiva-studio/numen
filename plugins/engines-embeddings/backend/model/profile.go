package model

import (
	"bytes"
	"crypto/sha256"
	_ "embed" // profiles.json ships inside the plugin
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// DefaultModel is the local model a plugin runs when the settings name none.
const DefaultModel = "Xenova/multilingual-e5-small"

// MaxChunkTokens is the most tokens of a text a local model reads, whatever its profile allows.
const MaxChunkTokens = 512

// ProfilesFolder is the folder, inside the data directory of the plugin, where the owner puts
// profile files of their own.
const ProfilesFolder = "profiles"

// fallbackRemoteID is the id of the profile an unlisted remote model takes.
const fallbackRemoteID = "*"

// InputPositionIDs is the extra input a profile asks the model to be given.
const InputPositionIDs = "position_ids"

//go:embed profiles.json
var embeddedProfiles []byte

// ErrNoProfile is returned for a model id that no profile lists.
var ErrNoProfile = errors.New("no profile for the model")

// Profile holds everything about an embedding model that a person, not the model file, decides:
// how its output becomes a vector, what is put in front of a text, and where the files are.
// The shipped table is profiles.json beside this file, and a file in the profiles folder of the
// plugin data directory adds a profile or changes one by its id.
type Profile struct {
	ID             string            `json:"id"`
	Aliases        []string          `json:"aliases,omitempty"`
	Title          string            `json:"title"`
	Tier           string            `json:"tier,omitempty"`
	SizeMB         int               `json:"sizeMB,omitempty"`
	Remote         bool              `json:"remote,omitempty"`
	Dimensions     int               `json:"dims"`
	MaxTokens      int               `json:"maxTokens"`
	Pooling        string            `json:"pooling"`
	Normalise      *bool             `json:"normalise,omitempty"`
	QueryPrefix    string            `json:"queryPrefix,omitempty"`
	DocumentPrefix string            `json:"documentPrefix,omitempty"`
	OnnxFile       string            `json:"onnxFile,omitempty"`
	Revision       string            `json:"revision,omitempty"`
	SHA256         map[string]string `json:"sha256,omitempty"`
	ExtraInputs    []string          `json:"extraInputs,omitempty"`
	FloorHint      float64           `json:"floorHint,omitempty"`
}

// IsNormalised reports whether the vectors of the model are scaled to unit length.
func (p Profile) IsNormalised() bool {
	return p.Normalise == nil || *p.Normalise
}

// GetTokens returns the most tokens of a text the plugin gives the model.
func (p Profile) GetTokens() int {
	if p.Remote {
		return p.MaxTokens
	}
	return min(p.MaxTokens, MaxChunkTokens)
}

// GetModelFile returns the name of the model file inside the onnx folder of the repository.
func (p Profile) GetModelFile() string {
	return strings.TrimPrefix(p.OnnxFile, ModelFolder+"/")
}

// GetOptionTitle returns the name the settings list the model under.
func (p Profile) GetOptionTitle() string {
	return fmt.Sprintf("%s (%s, %d MB)", p.Title, p.Tier, p.SizeMB)
}

// GetOptionDetail returns the line the settings show under the name.
func (p Profile) GetOptionDetail() string {
	return fmt.Sprintf("%d dims · Hugging Face", p.Dimensions)
}

// Handling is what is done to a text before the model reads it and to the output after. It decides
// which vector a text gets, so it is part of the recipe.
type Handling struct {
	QueryPrefix    string   `json:"queryPrefix"`
	DocumentPrefix string   `json:"documentPrefix"`
	File           string   `json:"file"`
	Normalise      bool     `json:"normalise"`
	ExtraInputs    []string `json:"extraInputs"`
}

// Role is what a text is embedded for.
type Role int

// The roles of a text.
const (
	RoleDocument Role = iota
	RoleQuery
)

// Apply puts the prefix of the role in front of each text.
func (h Handling) Apply(role Role, texts []string) []string {
	prefix := h.DocumentPrefix
	if role == RoleQuery {
		prefix = h.QueryPrefix
	}
	if prefix == "" {
		return texts
	}
	out := make([]string, len(texts))
	for i, text := range texts {
		out[i] = prefix + text
	}
	return out
}

// Hash returns a short value that changes with the prefixes, the model file, the token limit and the
// normalisation, and with nothing else.
func (h Handling) Hash(maxTokens int) string {
	parts := []string{h.QueryPrefix, h.DocumentPrefix, h.File, strconv.Itoa(maxTokens), strconv.FormatBool(h.Normalise)}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])[:handlingHashLength]
}

const handlingHashLength = 8

// GetHandling returns the handling of the texts of the model.
func (p Profile) GetHandling() Handling {
	if p.Remote {
		return Handling{}
	}
	return Handling{
		QueryPrefix:    p.QueryPrefix,
		DocumentPrefix: p.DocumentPrefix,
		File:           p.GetModelFile(),
		Normalise:      p.IsNormalised(),
		ExtraInputs:    p.ExtraInputs,
	}
}

// Validate returns an error when the profile cannot run a model.
func (p Profile) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("a profile needs an id")
	}
	if p.Dimensions <= 0 {
		return fmt.Errorf("%s: dims must be positive", p.ID)
	}
	if p.MaxTokens <= 0 {
		return fmt.Errorf("%s: maxTokens must be positive", p.ID)
	}
	if !isPooling(p.Pooling) {
		return fmt.Errorf("%s: pooling %q is none of mean, cls, last, output", p.ID, p.Pooling)
	}
	if p.FloorHint < 0 || p.FloorHint >= 1 {
		return fmt.Errorf("%s: floorHint %v is outside [0, 1)", p.ID, p.FloorHint)
	}
	for _, input := range p.ExtraInputs {
		if input != InputPositionIDs {
			return fmt.Errorf("%s: extraInputs %q is not supported, only %s", p.ID, input, InputPositionIDs)
		}
	}
	if !p.Remote && p.OnnxFile == "" {
		return fmt.Errorf("%s: a local model needs onnxFile", p.ID)
	}
	return nil
}

func isPooling(name string) bool {
	switch name {
	case PoolMean, PoolCLS, PoolLast, PoolOutput:
		return true
	}
	return false
}

// Profiles is the table of model profiles: the shipped ones and the owner's.
type Profiles struct {
	order    []string
	byID     map[string]Profile
	byName   map[string]string
	userHome string
}

// ReadProfiles parses a profile file: an array of profiles, or one profile.
func ReadProfiles(data []byte) ([]Profile, error) {
	raw, err := splitProfiles(data)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(raw))
	for _, one := range raw {
		var p Profile
		if err := decodeStrict(one, &p); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func splitProfiles(data []byte) ([]json.RawMessage, error) {
	text := bytes.TrimSpace(data)
	if !bytes.HasPrefix(text, []byte("[")) {
		return []json.RawMessage{text}, nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(text, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func decodeStrict(data []byte, into any) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(into)
}

// GetShippedProfiles returns the table that ships with the plugin.
func GetShippedProfiles() (*Profiles, error) {
	table := &Profiles{order: nil, byID: map[string]Profile{}, byName: map[string]string{}, userHome: ""}
	list, err := ReadProfiles(embeddedProfiles)
	if err != nil {
		return table, fmt.Errorf("profiles.json: %w", err)
	}
	for _, p := range list {
		if err := p.Validate(); err != nil {
			return table, fmt.Errorf("profiles.json: %w", err)
		}
		table.put(p)
	}
	return table, nil
}

func (t *Profiles) put(p Profile) {
	if _, ok := t.byID[p.ID]; !ok {
		t.order = append(t.order, p.ID)
	}
	t.byID[p.ID] = p
	t.byName[strings.ToLower(p.ID)] = p.ID
	for _, alias := range p.Aliases {
		t.byName[strings.ToLower(alias)] = p.ID
	}
}

func (t *Profiles) clone() *Profiles {
	out := &Profiles{order: slices.Clone(t.order), byID: map[string]Profile{}, byName: map[string]string{}, userHome: t.userHome}
	maps.Copy(out.byID, t.byID)
	maps.Copy(out.byName, t.byName)
	return out
}

// UserFile is a profile file the owner keeps, with the path it was read from.
type UserFile struct {
	Path string
	Data []byte
}

// WithUserProfiles returns the table with the profiles of the files merged over it by id, in the
// order of the files. dir is the folder the files come from. A field a user profile leaves out keeps
// the shipped value. A file that holds a profile that cannot run is passed to report and skipped
// whole.
func (t *Profiles) WithUserProfiles(dir string, files []UserFile, report func(file string, err error)) *Profiles {
	out := t.clone()
	out.userHome = dir
	for _, file := range files {
		merged, err := out.readUserFile(file.Data)
		if err != nil {
			report(file.Path, err)
			continue
		}
		for _, p := range merged {
			out.put(p)
		}
	}
	return out
}

func (t *Profiles) readUserFile(data []byte) ([]Profile, error) {
	raw, err := splitProfiles(data)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(raw))
	for _, one := range raw {
		p, err := t.layer(one)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (t *Profiles) layer(entry json.RawMessage) (Profile, error) {
	var head struct {
		ID string `json:"id"`
	}
	// gomutants:disable-next-line BRANCH_IF reason="decodeStrict rejects any entry that fails json.Unmarshal"
	if err := json.Unmarshal(entry, &head); err != nil {
		return Profile{}, err
	}
	base := t.byID[t.byName[strings.ToLower(head.ID)]]
	p := cloneProfile(base)
	if err := decodeStrict(entry, &p); err != nil {
		return Profile{}, err
	}
	if base.ID != "" {
		p.ID = base.ID
		dropStalePins(&p, base, entry)
	}
	return p, p.Validate()
}

func dropStalePins(p *Profile, base Profile, entry json.RawMessage) {
	if p.OnnxFile == base.OnnxFile || bytes.Contains(entry, []byte(`"sha256"`)) {
		return
	}
	p.SHA256 = nil
	if !bytes.Contains(entry, []byte(`"revision"`)) {
		p.Revision = ""
	}
}

func cloneProfile(p Profile) Profile {
	p.Aliases = slices.Clone(p.Aliases)
	p.ExtraInputs = slices.Clone(p.ExtraInputs)
	if p.SHA256 != nil {
		sums := make(map[string]string, len(p.SHA256))
		maps.Copy(sums, p.SHA256)
		p.SHA256 = sums
	}
	return p
}

// GetLocal returns the profile of a model run on this machine. A model that no profile lists is an
// error that says where to add one.
func (t *Profiles) GetLocal(id string) (Profile, error) {
	p, ok := t.byID[t.byName[strings.ToLower(strings.TrimSpace(id))]]
	if !ok || p.Remote {
		return Profile{}, fmt.Errorf("%w %q: add a profile for it as a .json file in %s, or choose one of %s",
			ErrNoProfile, id, t.getUserFolder(), strings.Join(t.getLocalIDs(), ", "))
	}
	return cloneProfile(p), nil
}

// GetRemote returns the profile of a model that a service runs. A model that no profile lists takes
// the profile of any other remote model.
func (t *Profiles) GetRemote(id string) Profile {
	if p, ok := t.byID[t.byName[strings.ToLower(strings.TrimSpace(id))]]; ok && p.Remote {
		return cloneProfile(p)
	}
	p := cloneProfile(t.byID[fallbackRemoteID])
	p.ID = id
	return p
}

// IsLocal reports whether the table lists the id as a model run on this machine.
func (t *Profiles) IsLocal(id string) bool {
	p, ok := t.byID[t.byName[strings.ToLower(strings.TrimSpace(id))]]
	return ok && !p.Remote
}

func (t *Profiles) getUserFolder() string {
	if t.userHome == "" {
		return "the " + ProfilesFolder + " folder of the plugin data directory"
	}
	return t.userHome
}

func (t *Profiles) getLocalIDs() []string {
	var ids []string
	for _, id := range t.order {
		if !t.byID[id].Remote {
			ids = append(ids, id)
		}
	}
	return ids
}

// GetCurated returns the profiles the settings list, in table order.
func (t *Profiles) GetCurated() []Profile {
	var out []Profile
	for _, id := range t.order {
		if p := t.byID[id]; !p.Remote && p.Tier != "" {
			out = append(out, cloneProfile(p))
		}
	}
	return out
}

// IsRemote reports whether the table lists the id as a model that a service runs.
func (t *Profiles) IsRemote(id string) bool {
	p, ok := t.byID[t.byName[strings.ToLower(strings.TrimSpace(id))]]
	return ok && p.Remote
}

// IsZero reports whether the handling changes nothing.
func (h Handling) IsZero() bool {
	return h.QueryPrefix == "" && h.DocumentPrefix == "" && h.File == "" && !h.Normalise && len(h.ExtraInputs) == 0
}
