package manifestfile

import (
	"encoding/json"
	"errors"
	"io"
	"math"

	"github.com/jiva-studio/numen/libs/notes/domain"

	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
)

func init() {
	pluginport.SetManifestDecoder(decodeManifest)
}

const (
	// PermissionVaultRead allows reading files from the active vault.
	PermissionVaultRead = pluginport.PermissionVaultRead

	// PermissionVaultWrite allows modifying files in the active vault.
	PermissionVaultWrite = pluginport.PermissionVaultWrite

	// PermissionNetworkOutbound allows outbound network requests.
	PermissionNetworkOutbound = pluginport.PermissionNetworkOutbound

	// PermissionAIComplete allows asking the host for a model completion.
	PermissionAIComplete = pluginport.PermissionAIComplete
)

// Lifecycle defines how the host activates and evicts a plugin from memory.
type Lifecycle = pluginport.PluginLifecycle

// Backend describes the server-side executable or wasm module of a plugin.
type Backend = pluginport.PluginBackend

// Frontend describes the client-side bundle and styles of a plugin.
type Frontend = pluginport.PluginFrontend

// SettingOptions is a slice of option values supporting string arrays and object arrays.
type SettingOptions = pluginport.PluginSettingOptions

// Setting declares one configuration field managed by the core host.
type Setting = pluginport.PluginManifestSetting

// Manifest is the metadata declaring a plugin's identity, capabilities, and lifecycle.
type Manifest = pluginport.PluginManifest

// ReadManifest decodes and validates a manifest from JSON input.
func ReadManifest(r io.Reader) (Manifest, error) {
	return pluginport.ReadManifest(r)
}

type rawDependencyRequirement struct {
	ID             string                     `json:"id"`
	Kind           pluginport.DependencyKind  `json:"kind"`
	Version        string                     `json:"version,omitempty"`
	SourceURL      string                     `json:"sourceUrl,omitempty"`
	ChecksumSHA256 string                     `json:"checksumSha256,omitempty"`
	Metadata       map[string]string          `json:"metadata,omitempty"`
}

type rawLifecycle struct {
	IdleTimeoutSeconds    int    `json:"idleTimeoutSeconds"`
	KeepAliveWhileVisible bool   `json:"keepAliveWhileVisible"`
	MemoryLimitBytes      uint64 `json:"memoryLimitBytes,omitempty"`
	MemoryLimitMB         uint64 `json:"memoryLimitMB,omitempty"`
}

func (l rawLifecycle) toPort() (pluginport.PluginLifecycle, error) {
	limit := l.MemoryLimitBytes
	if limit == 0 && l.MemoryLimitMB > 0 {
		if l.MemoryLimitMB > math.MaxUint64/(1024*1024) {
			return pluginport.PluginLifecycle{}, errors.New("memory limit in megabytes exceeds maximum value")
		}
		limit = l.MemoryLimitMB * 1024 * 1024
	}
	return pluginport.PluginLifecycle{
		IdleTimeoutSeconds:    l.IdleTimeoutSeconds,
		KeepAliveWhileVisible: l.KeepAliveWhileVisible,
		MemoryLimitBytes:      limit,
	}, nil
}

type rawBackend struct {
	Type        string `json:"type"`
	Entrypoint  string `json:"entrypoint"`
	ServiceName string `json:"serviceName"`
}

type rawFrontend struct {
	Entrypoint string `json:"entrypoint"`
	Styles     string `json:"styles"`
}

type rawSettingOption struct {
	Value        string   `json:"value"`
	Title        string   `json:"title,omitempty"`
	Detail       string   `json:"detail,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`
}

type rawSettingOptions []pluginport.PluginSettingOption

func (o *rawSettingOptions) UnmarshalJSON(data []byte) error {
	var values []string
	if err := json.Unmarshal(data, &values); err == nil {
		options := make(rawSettingOptions, 0, len(values))
		for _, value := range values {
			options = append(options, pluginport.PluginSettingOption{Value: value})
		}
		*o = options
		return nil
	}
	var raw []rawSettingOption
	if err := json.Unmarshal(data, &raw); err != nil {
		return errors.New("options must be a list of values or of objects with a value")
	}
	options := make(rawSettingOptions, 0, len(raw))
	for _, item := range raw {
		options = append(options, pluginport.PluginSettingOption(item))
	}
	*o = options
	return nil
}

type rawSettingOptionsBy struct {
	Setting string                       `json:"setting"`
	Options map[string]rawSettingOptions `json:"options"`
}

func (b *rawSettingOptionsBy) toPort() *pluginport.PluginSettingOptionsBy {
	if b == nil {
		return nil
	}
	opts := make(map[string]pluginport.PluginSettingOptions, len(b.Options))
	for k, v := range b.Options {
		opts[k] = pluginport.PluginSettingOptions(v)
	}
	return &pluginport.PluginSettingOptionsBy{
		Setting: b.Setting,
		Options: opts,
	}
}

type rawSettingCondition struct {
	Setting string   `json:"setting"`
	Is      []string `json:"is"`
}

func (c *rawSettingCondition) toPort() *pluginport.PluginSettingCondition {
	if c == nil {
		return nil
	}
	return &pluginport.PluginSettingCondition{
		Setting: c.Setting,
		Is:      c.Is,
	}
}

type rawSecret struct {
	Name        string               `json:"name"`
	Title       string               `json:"title"`
	Description string               `json:"description,omitempty"`
	When        *rawSettingCondition `json:"when,omitempty"`
}

type rawSetting struct {
	Key         string               `json:"key"`
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Default     any                  `json:"default"`
	Palette     bool                 `json:"palette,omitempty"`
	Icon        string               `json:"icon,omitempty"`
	Options     rawSettingOptions    `json:"options,omitempty"`
	Min         float64              `json:"min,omitempty"`
	Max         float64              `json:"max,omitempty"`
	Step        float64              `json:"step,omitempty"`
	ModelsOf    string               `json:"modelsOf,omitempty"`
	Replaces    []string             `json:"replaces,omitempty"`
	OptionsBy   *rawSettingOptionsBy `json:"optionsBy,omitempty"`
	When        *rawSettingCondition `json:"when,omitempty"`
	Accept      []string             `json:"accept,omitempty"`
}

type rawNoteType struct {
	Type            string `json:"type"`
	Chunks          *bool  `json:"chunks,omitempty"`
	Headings        *[]int `json:"headings,omitempty"`
	IDHeadingLevels []int  `json:"idHeadingLevels,omitempty"`
	IDPattern       string `json:"idPattern,omitempty"`
}

type rawCommandArgument struct {
	Name              string `json:"name"`
	Title             string `json:"title"`
	Type              string `json:"type"`
	IsRequired        bool   `json:"required,omitempty"`
	IsHiddenInPalette bool   `json:"hiddenInPalette,omitempty"`
}

type rawCommand struct {
	ID                string               `json:"id"`
	Title             string               `json:"title"`
	Icon              string               `json:"icon,omitempty"`
	ContentTypes      []domain.ContentType `json:"contentTypes,omitempty"`
	NoteTypes         []string             `json:"noteTypes,omitempty"`
	Arguments         []rawCommandArgument `json:"arguments,omitempty"`
	IsHiddenInPalette bool                 `json:"hiddenInPalette,omitempty"`
}

type rawMenuEntry struct {
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
}

type rawTextInput struct {
	ContentTypes []domain.ContentType `json:"contentTypes,omitempty"`
	Kinds        []domain.TextKind    `json:"kinds,omitempty"`
	NoteTypes    []string             `json:"noteTypes,omitempty"`
	IsAutoRun    bool                 `json:"autoRun,omitempty"`
	MenuEntry    *rawMenuEntry        `json:"menuEntry,omitempty"`
}

func (t *rawTextInput) toPort() *pluginport.PluginTextInput {
	if t == nil {
		return nil
	}
	var entry *pluginport.PluginMenuEntry
	if t.MenuEntry != nil {
		e := pluginport.PluginMenuEntry(*t.MenuEntry)
		entry = &e
	}
	return &pluginport.PluginTextInput{
		ContentTypes: t.ContentTypes,
		Kinds:        t.Kinds,
		NoteTypes:    t.NoteTypes,
		IsAutoRun:    t.IsAutoRun,
		MenuEntry:    entry,
	}
}

type rawPlayback struct {
	ContentTypes []domain.ContentType `json:"contentTypes,omitempty"`
	NoteTypes    []string             `json:"noteTypes,omitempty"`
}

type rawContentType struct {
	Type  domain.ContentType `json:"type"`
	Globs []string           `json:"globs"`
}

type rawReader struct {
	ID           string               `json:"id"`
	ContentTypes []domain.ContentType `json:"contentTypes"`
	Rank         domain.HandlerRank   `json:"rank"`
}

type rawManifest struct {
	ID               string                     `json:"id"`
	Name             string                     `json:"name"`
	Version          string                     `json:"version"`
	Protocol         int                        `json:"protocol,omitempty"`
	Description      string                     `json:"description"`
	Capabilities     []string                   `json:"capabilities"`
	Targets          []string                   `json:"targets"`
	ActivationEvents []string                   `json:"activationEvents"`
	Permissions      []string                   `json:"permissions"`
	Runtimes         []string                   `json:"runtimes,omitempty"`
	Dependencies     []rawDependencyRequirement `json:"dependencies,omitempty"`
	Lifecycle        rawLifecycle               `json:"lifecycle"`
	Backend          rawBackend                 `json:"backend,omitzero"`
	Frontend         rawFrontend                `json:"frontend,omitzero"`
	Settings         []rawSetting               `json:"settings"`
	Secrets          []rawSecret                `json:"secrets,omitempty"`
	Services         []string                   `json:"services,omitempty"`
	NoteTypes        []rawNoteType              `json:"noteTypes,omitempty"`
	Commands         []rawCommand               `json:"commands,omitempty"`
	TextInput        *rawTextInput              `json:"textInput,omitempty"`
	Playback         *rawPlayback               `json:"playback,omitempty"`
	ContentTypes     []rawContentType           `json:"contentTypes,omitempty"`
	Readers          []rawReader                `json:"readers,omitempty"`
	Frames           []string                   `json:"frames,omitempty"`
}

func decodeManifest(data []byte) (pluginport.PluginManifest, error) {
	var raw rawManifest
	if err := json.Unmarshal(data, &raw); err != nil {
		return pluginport.PluginManifest{}, err
	}
	lifecycle, err := raw.Lifecycle.toPort()
	if err != nil {
		return pluginport.PluginManifest{}, err
	}
	return raw.toPort(lifecycle), nil
}

func (r rawManifest) toPort(lifecycle pluginport.PluginLifecycle) pluginport.PluginManifest {
	var playback *pluginport.PluginPlayback
	if r.Playback != nil {
		p := pluginport.PluginPlayback(*r.Playback)
		playback = &p
	}
	return pluginport.PluginManifest{
		ID:               r.ID,
		Name:             r.Name,
		Version:          r.Version,
		Protocol:         r.Protocol,
		Description:      r.Description,
		Capabilities:     r.Capabilities,
		Targets:          r.Targets,
		ActivationEvents: r.ActivationEvents,
		Permissions:      r.Permissions,
		Runtimes:         r.Runtimes,
		Dependencies:     convertDependencies(r.Dependencies),
		Lifecycle:        lifecycle,
		Backend:          pluginport.PluginBackend(r.Backend),
		Frontend:         pluginport.PluginFrontend(r.Frontend),
		Settings:         convertSettings(r.Settings),
		Secrets:          convertSecrets(r.Secrets),
		Services:         r.Services,
		NoteTypes:        convertNoteTypes(r.NoteTypes),
		Commands:         convertCommands(r.Commands),
		TextInput:        r.TextInput.toPort(),
		Playback:         playback,
		ContentTypes:     convertContentTypes(r.ContentTypes),
		Readers:          convertReaders(r.Readers),
		Frames:           r.Frames,
	}
}

func convertDependencies(in []rawDependencyRequirement) []pluginport.DependencyRequirement {
	if in == nil {
		return nil
	}
	out := make([]pluginport.DependencyRequirement, 0, len(in))
	for _, d := range in {
		out = append(out, pluginport.DependencyRequirement{
			ID:             d.ID,
			Kind:           d.Kind,
			Version:        d.Version,
			SourceURL:      d.SourceURL,
			ChecksumSHA256: d.ChecksumSHA256,
			Metadata:       d.Metadata,
		})
	}
	return out
}

func convertSettings(in []rawSetting) []pluginport.PluginManifestSetting {
	if in == nil {
		return nil
	}
	out := make([]pluginport.PluginManifestSetting, 0, len(in))
	for _, s := range in {
		out = append(out, pluginport.PluginManifestSetting{
			Key:         s.Key,
			Type:        s.Type,
			Title:       s.Title,
			Description: s.Description,
			Default:     s.Default,
			Palette:     s.Palette,
			Icon:        s.Icon,
			Options:     pluginport.PluginSettingOptions(s.Options),
			Min:         s.Min,
			Max:         s.Max,
			Step:        s.Step,
			ModelsOf:    s.ModelsOf,
			Replaces:    s.Replaces,
			OptionsBy:   s.OptionsBy.toPort(),
			When:        s.When.toPort(),
			Accept:      s.Accept,
		})
	}
	return out
}

func convertSecrets(in []rawSecret) []pluginport.PluginSecret {
	if in == nil {
		return nil
	}
	out := make([]pluginport.PluginSecret, 0, len(in))
	for _, s := range in {
		out = append(out, pluginport.PluginSecret{
			Name:        s.Name,
			Title:       s.Title,
			Description: s.Description,
			When:        s.When.toPort(),
		})
	}
	return out
}

func convertNoteTypes(in []rawNoteType) []pluginport.PluginNoteType {
	if in == nil {
		return nil
	}
	out := make([]pluginport.PluginNoteType, 0, len(in))
	for _, n := range in {
		out = append(out, pluginport.PluginNoteType(n))
	}
	return out
}

func convertCommands(in []rawCommand) []pluginport.PluginCommand {
	if in == nil {
		return nil
	}
	out := make([]pluginport.PluginCommand, 0, len(in))
	for _, c := range in {
		var args []pluginport.PluginCommandArgument
		if c.Arguments != nil {
			args = make([]pluginport.PluginCommandArgument, 0, len(c.Arguments))
			for _, a := range c.Arguments {
				args = append(args, pluginport.PluginCommandArgument(a))
			}
		}
		out = append(out, pluginport.PluginCommand{
			ID:                c.ID,
			Title:             c.Title,
			Icon:              c.Icon,
			ContentTypes:      c.ContentTypes,
			NoteTypes:         c.NoteTypes,
			Arguments:         args,
			IsHiddenInPalette: c.IsHiddenInPalette,
		})
	}
	return out
}

func convertContentTypes(in []rawContentType) []pluginport.PluginContentType {
	if in == nil {
		return nil
	}
	out := make([]pluginport.PluginContentType, 0, len(in))
	for _, ct := range in {
		out = append(out, pluginport.PluginContentType(ct))
	}
	return out
}

func convertReaders(in []rawReader) []pluginport.PluginReader {
	if in == nil {
		return nil
	}
	out := make([]pluginport.PluginReader, 0, len(in))
	for _, r := range in {
		out = append(out, pluginport.PluginReader(r))
	}
	return out
}
