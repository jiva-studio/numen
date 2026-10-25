package textwire

import (
	"time"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ReadFormat is a format the protocol carries, and empty for one the core does not read.
func ReadFormat(f v1.TextFormat) domain.TextFormat {
	switch f {
	case v1.TextFormat_TEXT_FORMAT_PLAIN:
		return domain.TextFormatPlain
	case v1.TextFormat_TEXT_FORMAT_TIMED:
		return domain.TextFormatTimed
	case v1.TextFormat_TEXT_FORMAT_POSITIONED:
		return domain.TextFormatPositioned
	case v1.TextFormat_TEXT_FORMAT_UNSPECIFIED:
	}
	return ""
}

// WriteFormat is a format as the protocol carries it.
func WriteFormat(f domain.TextFormat) v1.TextFormat {
	switch f {
	case domain.TextFormatPlain:
		return v1.TextFormat_TEXT_FORMAT_PLAIN
	case domain.TextFormatTimed:
		return v1.TextFormat_TEXT_FORMAT_TIMED
	case domain.TextFormatPositioned:
		return v1.TextFormat_TEXT_FORMAT_POSITIONED
	}
	return v1.TextFormat_TEXT_FORMAT_UNSPECIFIED
}

// ReadParts is the parts of a text as the protocol carries them.
func ReadParts(parts []*v1.TextPart) []domain.TextPart {
	out := make([]domain.TextPart, 0, len(parts))
	for _, p := range parts {
		place := p.GetPlace()
		rects := make([]domain.Rect, 0, len(place.GetRects()))
		for _, r := range place.GetRects() {
			rects = append(rects, domain.Rect{MinX: r.GetMinX(), MinY: r.GetMinY(), MaxX: r.GetMaxX(), MaxY: r.GetMaxY()})
		}
		out = append(out, domain.TextPart{
			Text:    p.GetText(),
			Section: p.GetSection(),
			Place: domain.TextPlace{
				Page:  int(place.GetPage()),
				Rects: rects,
				Start: time.Duration(place.GetStartMs()) * time.Millisecond,
				End:   time.Duration(place.GetEndMs()) * time.Millisecond,
			},
		})
	}
	return out
}

// WriteParts is the parts of a text as the protocol carries them.
func WriteParts(parts []domain.TextPart) []*v1.TextPart {
	out := make([]*v1.TextPart, 0, len(parts))
	for _, p := range parts {
		rects := make([]*v1.TextRect, 0, len(p.Place.Rects))
		for _, r := range p.Place.Rects {
			rects = append(rects, &v1.TextRect{MinX: r.MinX, MinY: r.MinY, MaxX: r.MaxX, MaxY: r.MaxY})
		}
		out = append(out, &v1.TextPart{
			Text:    p.Text,
			Section: p.Section,
			Place: &v1.TextPlace{
				Page:    int32(p.Place.Page),
				Rects:   rects,
				StartMs: p.Place.Start.Milliseconds(),
				EndMs:   p.Place.End.Milliseconds(),
			},
		})
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
		InputVersionId:   string(v.Input),
		Format:           WriteFormat(v.Format),
		InputHash:        v.InputHash,
		Hash:             v.Hash,
		CreateTimeUnixMs: v.CreateTime.UnixMilli(),
	}
}
