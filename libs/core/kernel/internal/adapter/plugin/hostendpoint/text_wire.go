package hostendpoint

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func writeSegments(segments []domain.TextSegment) []*v1.TextSegment {
	out := make([]*v1.TextSegment, 0, len(segments))
	for _, s := range segments {
		out = append(out, &v1.TextSegment{Text: s.Text, Heading: s.Heading, Selector: string(s.Selector)})
	}
	return out
}

func writeVersion(v domain.TextVersion) *v1.TextVersion {
	return &v1.TextVersion{
		Id:               string(v.GetID()),
		Path:             v.Path,
		PluginId:         v.PluginID,
		Key:              v.Key,
		SourceVersionId:  string(v.Source),
		Kind:             string(v.Kind),
		InputHash:        v.InputHash,
		TextHash:         v.TextHash,
		CreateTimeUnixMs: v.CreateTime.UnixMilli(),
		File:             v.File,
		SourceProperty:   writeSourceProperty(v.SourceProperty),
	}
}

func writeSourceProperty(p domain.TextSourceProperty) *v1.TextSourceProperty {
	if p.Key == "" {
		return nil
	}
	return &v1.TextSourceProperty{Key: p.Key, Value: p.Value}
}
