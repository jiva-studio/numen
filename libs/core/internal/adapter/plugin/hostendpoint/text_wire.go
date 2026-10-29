package hostendpoint

import (
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// writeSegments is the segments of a text as the protocol carries them.
func writeSegments(segments []domain.TextSegment) []*v1.TextSegment {
	out := make([]*v1.TextSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, &v1.TextSegment{Text: s.Text, Heading: s.Heading, Selector: string(s.Selector)})
	}
	return out
}

// writeVersion is a version as the protocol carries it.
func writeVersion(v domain.TextVersion) *v1.TextVersion {
	return &v1.TextVersion{
		Id:               string(v.GetID()),
		Path:             v.Path,
		PluginId:         v.PluginID,
		Key:              v.Key,
		SourceVersionId:  string(v.Source),
		Kind:             string(v.Kind),
		SourceHash:       v.SourceHash,
		Hash:             v.Hash,
		CreateTimeUnixMs: v.CreateTime.UnixMilli(),
		File:             v.File,
	}
}
