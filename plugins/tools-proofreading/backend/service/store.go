package service

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/domain"
)

// keyPrefix starts the key of every version the plugin makes.
const keyPrefix = "proofread-"

// getVersionKey is the key of the proofreading of a version. One version has one proofreading,
// rewritten when it is proofread again.
func getVersionKey(source domain.TextVersionID) string {
	sum := sha256.Sum256([]byte(source))
	return keyPrefix + hex.EncodeToString(sum[:8])
}

// getFileName is the file of the plugin's folder the corrected text kept under the key is written
// to, in the common format of text versions.
func getFileName(key string) string { return key + ".jsonl" }

// applyCorrections is the parts with the corrections in place of the lines they answer for, a
// line numbered as readLines numbers the parts. A correction keeps the space around the words of
// the parts it replaces. A correction answering for a run of lines is one part: the place of a
// timed run spans from the first part's start to the last part's end, and a positioned run keeps
// the rectangles of its parts on the first part's page.
func applyCorrections(parts []domain.TextSegment, corrections []Line) []domain.TextSegment {
	byLine := make(map[int]Line, len(corrections))
	for _, line := range corrections {
		byLine[line.Number] = line
	}
	out := make([]domain.TextSegment, 0, len(parts))
	for i := 0; i < len(parts); i++ {
		line, ok := byLine[i+1]
		if !ok {
			out = append(out, parts[i])
			continue
		}
		last := min(getLastLine(line), len(parts)) - 1
		first, end := parts[i], parts[last]
		part := domain.TextSegment{
			Text:     getLeadingSpace(first.Text) + line.Text + getTrailingSpace(end.Text),
			Heading:  first.Heading,
			Selector: first.Selector,
		}
		if last > i {
			part.Selector = joinSelectors(parts[i : last+1])
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

// joinSelectors is the selector of a run of segments made one: the span of a recording from the
// first segment's start to the last one's end, and the boxes of the segments on the first one's page.
func joinSelectors(run []domain.TextSegment) domain.Selector {
	first, err := readSegmentSelector(run[0].Selector)
	if err != nil {
		return run[0].Selector
	}
	joined := segmentSelector{T: nil, Page: first.Page, CFI: first.CFI, Boxes: nil}
	for _, one := range run {
		selector, err := readSegmentSelector(one.Selector)
		if err != nil {
			continue
		}
		if len(selector.T) == 2 {
			if len(joined.T) == 0 {
				joined.T = []float64{selector.T[0], selector.T[1]}
			}
			joined.T[1] = selector.T[1]
		}
		if selector.Page == first.Page {
			joined.Boxes = append(joined.Boxes, selector.Boxes...)
		}
	}
	return joined.encode()
}
