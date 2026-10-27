package textwire

import (
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ReadSegments is the segments of a text as the protocol carries them.
func ReadSegments(segments []*v1.TextSegment) []domain.TextSegment {
	out := make([]domain.TextSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, domain.TextSegment{Text: s.GetText(), Heading: s.GetHeading(), Selector: domain.Selector(s.GetSelector())})
	}
	return out
}

// WriteSegments is the segments of a text as the protocol carries them.
func WriteSegments(segments []domain.TextSegment) []*v1.TextSegment {
	out := make([]*v1.TextSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, &v1.TextSegment{Text: s.Text, Heading: s.Heading, Selector: string(s.Selector)})
	}
	return out
}

// WriteVersion is a version as the protocol carries it.
func WriteVersion(v domain.TextVersion) *v1.TextVersion {
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
