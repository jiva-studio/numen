package profile

import (
	"bytes"
	_ "embed" // profiles.json ships inside the plugin
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

//go:embed profiles.json
var embeddedProfiles []byte

// Profiles is the table of profiles: the shipped ones and the person's.
type Profiles struct {
	order    []string
	byID     map[string]Profile
	byName   map[string]string
	userHome string
}

// UserFile is a profile file the person keeps, with the path it was read from.
type UserFile struct {
	Path string
	Data []byte
}

// ReadProfiles parses a profile file: an array of profiles, or one profile. A field the profile does
// not know is an error.
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

// WithUserProfiles returns the table with the profiles of the files merged over it by id or alias, in
// the order of the files. dir is the folder the files come from. A field a file leaves out keeps the
// value of the table. A file that is not JSON, names an unknown field or holds a profile that cannot run
// is passed to report and skipped whole. The table itself does not change.
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
		p, err := t.mergeEntry(one)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func (t *Profiles) mergeEntry(entry json.RawMessage) (Profile, error) {
	var head struct {
		ID string `json:"id"`
	}
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
	if p.OnnxFile == base.OnnxFile {
		return
	}
	if !bytes.Contains(entry, []byte(`"dependencies"`)) {
		p.Dependencies = nil
	}
	if bytes.Contains(entry, []byte(`"sha256"`)) {
		return
	}
	p.SHA256 = nil
	if !bytes.Contains(entry, []byte(`"revision"`)) {
		p.Revision = ""
	}
}

func cloneProfile(p Profile) Profile {
	p.Aliases = slices.Clone(p.Aliases)
	p.Dependencies = maps.Clone(p.Dependencies)
	if p.SHA256 != nil {
		sums := make(map[string]string, len(p.SHA256))
		maps.Copy(sums, p.SHA256)
		p.SHA256 = sums
	}
	return p
}

// GetLocal returns the profile of a model by its id or alias, ignoring case; an empty model is
// DefaultModel. A model that no profile lists is an error that wraps ErrNoProfile and names the folder
// of the profile files and the known models.
func (t *Profiles) GetLocal(id string) (Profile, error) {
	name := strings.TrimSpace(id)
	if name == "" {
		name = DefaultModel
	}
	p, ok := t.byID[t.byName[strings.ToLower(name)]]
	if !ok {
		return Profile{}, fmt.Errorf("%w %q: add a profile for it as a .json file in %s, or choose one of %s",
			ErrNoProfile, id, t.getUserFolder(), strings.Join(t.order, ", "))
	}
	return cloneProfile(p), nil
}

func (t *Profiles) getUserFolder() string {
	if t.userHome == "" {
		return "the " + ProfilesFolder + " folder of the plugin data directory"
	}
	return t.userHome
}

// GetCurated returns the profiles that have a tier, in table order.
func (t *Profiles) GetCurated() []Profile {
	var out []Profile
	for _, id := range t.order {
		if p := t.byID[id]; p.Tier != "" {
			out = append(out, cloneProfile(p))
		}
	}
	return out
}
