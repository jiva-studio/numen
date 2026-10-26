package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/notes/domain"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// keyPrefix starts the key of every version the plugin makes.
const keyPrefix = "proofread-"

// getVersionKey is the key of the proofreading of a version. One version has one proofreading,
// rewritten when it is proofread again.
func getVersionKey(input domain.TextVersionID) string {
	sum := sha256.Sum256([]byte(input))
	return keyPrefix + hex.EncodeToString(sum[:8])
}

// textFiles are the corrected texts kept in the plugin's folder of a vault's store, `<key>.json`
// each.
type textFiles struct {
	store host.Store
}

func getFileName(key string) string { return key + ".json" }

// fileRect is a rectangle as [min x, min y, max x, max y].
type fileRect [4]float32

// filePart is one part of a kept text.
type filePart struct {
	Text    string     `json:"text"`
	Section string     `json:"section,omitempty"`
	Page    int        `json:"page,omitempty"`
	Rects   []fileRect `json:"rects,omitempty"`
	StartMS int64      `json:"startMs,omitempty"`
	EndMS   int64      `json:"endMs,omitempty"`
}

type textFile struct {
	Parts []filePart `json:"parts"`
}

// read is the parts kept under the key, or host.ErrNotExist.
func (k textFiles) read(ctx context.Context, key string) ([]domain.TextPart, error) {
	raw, err := k.store.Read(ctx, getFileName(key))
	if err != nil {
		return nil, err
	}
	var file textFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return nil, fmt.Errorf("read %s: %w", getFileName(key), err)
	}
	parts := make([]domain.TextPart, len(file.Parts))
	for i, one := range file.Parts {
		rects := make([]domain.Rect, 0, len(one.Rects))
		for _, rect := range one.Rects {
			rects = append(rects, domain.Rect{MinX: rect[0], MinY: rect[1], MaxX: rect[2], MaxY: rect[3]})
		}
		parts[i] = domain.TextPart{Text: one.Text, Section: one.Section, Place: domain.TextPlace{
			Page: one.Page, Rects: rects,
			Start: time.Duration(one.StartMS) * time.Millisecond, End: time.Duration(one.EndMS) * time.Millisecond,
		}}
	}
	return parts, nil
}

// write keeps the parts under the key.
func (k textFiles) write(ctx context.Context, key string, parts []domain.TextPart) error {
	file := textFile{Parts: make([]filePart, len(parts))}
	for i, part := range parts {
		one := filePart{
			Text: part.Text, Section: part.Section, Page: part.Place.Page,
			StartMS: part.Place.Start.Milliseconds(), EndMS: part.Place.End.Milliseconds(),
		}
		for _, rect := range part.Place.Rects {
			one.Rects = append(one.Rects, fileRect{rect.MinX, rect.MinY, rect.MaxX, rect.MaxY})
		}
		file.Parts[i] = one
	}
	raw, err := json.Marshal(file)
	if err != nil {
		return err
	}
	return k.store.Write(ctx, getFileName(key), raw)
}

// remove takes away what is kept under the key.
func (k textFiles) remove(ctx context.Context, key string) error {
	return k.store.Delete(ctx, getFileName(key))
}

// applyCorrections is the parts with the corrections in place of the lines they answer for, a
// line numbered as readLines numbers the parts. A correction keeps the space around the words of
// the parts it replaces. A correction answering for a run of lines is one part: the place of a
// timed run spans from the first part's start to the last part's end, and a positioned run keeps
// the rectangles of its parts on the first part's page.
func applyCorrections(parts []domain.TextPart, corrections []Line) []domain.TextPart {
	byLine := make(map[int]Line, len(corrections))
	for _, line := range corrections {
		byLine[line.Number] = line
	}
	out := make([]domain.TextPart, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		line, ok := byLine[i+1]
		if !ok {
			out = append(out, parts[i])
			continue
		}
		last := min(getLastLine(line), len(parts)) - 1
		first, end := parts[i], parts[last]
		part := domain.TextPart{
			Text:    getLeadingSpace(first.Text) + line.Text + getTrailingSpace(end.Text),
			Section: first.Section,
			Place:   first.Place,
		}
		if last > i {
			part.Place.End = end.Place.End
			part.Place.Rects = nil
			for _, joined := range parts[i : last+1] {
				if joined.Place.Page == first.Place.Page {
					part.Place.Rects = append(part.Place.Rects, joined.Place.Rects...)
				}
			}
		}
		out = append(out, part)
		i = last
	}
	return out
}

func getLeadingSpace(text string) string {
	return text[:len(text)-len(strings.TrimLeft(text, " \t\r\n"))]
}

func getTrailingSpace(text string) string {
	return text[len(strings.TrimRight(text, " \t\r\n")):]
}

// hashParts is the hash of the text the parts join to.
func hashParts(parts []domain.TextPart) string {
	var b strings.Builder
	for _, part := range parts {
		b.WriteString(part.Text)
	}
	return domain.HashContent([]byte(b.String()))
}
