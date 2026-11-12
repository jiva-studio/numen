package plugin

import (
	"maps"
	pathpkg "path"
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// TargetVersion is one text version of a target as target.versions holds it. A valid version still
// matches the file, its source chain whole, and the live one is the newest valid one.
type TargetVersion struct {
	Kind     string
	PluginID string
	IsLive   bool
	IsValid  bool
}

// TargetValues are the values a when condition reads over one target, built from its path and from
// what was read of it: the note, the text versions, the settings and the one version an item stands
// for. A part that was not read is left out, so a condition that depends on it is Unknown.
type TargetValues struct {
	values map[string]any
	isNote bool
}

// NewTargetValues derives the target variables from the path, whether it is a folder, and its content
// type, as libs/core/protocol/when.json lists them under targets. An empty path that is no folder is no
// target, and every variable is false or empty. The note type and properties of a note are left out
// until WithNote gives them; for any other path they are empty.
func NewTargetValues(path string, isFolder bool, contentType domain.ContentType) TargetValues {
	isNote := !isFolder && strings.EqualFold(pathpkg.Ext(path), domain.NoteExtension)
	typeName := string(contentType)
	if isFolder || contentType == domain.UnknownContentType {
		typeName = ""
	}
	values := map[string]any{
		"target.isRoot":      isFolder && path == "",
		"target.isFolder":    isFolder,
		"target.isMultiple":  false,
		"target.isNote":      isNote,
		"target.isFile":      !isFolder && !isNote && path != "",
		"target.contentType": typeName,
	}
	if !isNote {
		values["target.noteType"] = ""
		values["target.properties"] = map[string]any{}
	}
	return TargetValues{values: values, isNote: isNote}
}

// WithNote gives the type and the frontmatter read of the note. A target that is no note keeps both
// empty.
func (t TargetValues) WithNote(noteType domain.NoteType, properties map[string]any) TargetValues {
	if !t.isNote {
		return t
	}
	if properties == nil {
		properties = map[string]any{}
	}
	return t.with(map[string]any{"target.noteType": string(noteType), "target.properties": properties})
}

// WithVersions gives the text versions of the target as target.versions.
func (t TargetValues) WithVersions(versions []TargetVersion) TargetValues {
	list := make([]any, 0, len(versions))
	for _, one := range versions {
		list = append(list, map[string]any{"kind": one.Kind, "pluginId": one.PluginID, "isLive": one.IsLive, "isValid": one.IsValid})
	}
	return t.with(map[string]any{"target.versions": list})
}

// WithConfig gives the settings as config, by the owner split at its dots and then the key.
func (t TargetValues) WithConfig(config map[string]any) TargetValues {
	return t.with(map[string]any{"config": config})
}

// WithVersion gives the one text version an item of a command stands for, as version.kind and
// version.isLive.
func (t TargetValues) WithVersion(kind string, isLive bool) TargetValues {
	return t.with(map[string]any{"version.kind": kind, "version.isLive": isLive})
}

// Get returns the values by the qualified names of the variables, as Condition.Evaluate takes them.
func (t TargetValues) Get() map[string]any {
	return maps.Clone(t.values)
}

// IsNote reports whether the target is a note, whose type and properties WithNote gives.
func (t TargetValues) IsNote() bool {
	return t.isNote
}

func (t TargetValues) with(added map[string]any) TargetValues {
	values := maps.Clone(t.values)
	maps.Copy(values, added)
	return TargetValues{values: values, isNote: t.isNote}
}
