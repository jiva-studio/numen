// Package ytdlp reads what the yt-dlp program prints about a video: its information and its captions.
package ytdlp

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Cue is one stretch of speech: the transcript, and the milliseconds of the recording it spans.
type Cue struct {
	Text string
	From int
	To   int
}

// Info is what yt-dlp reports about a video. Length is in milliseconds.
type Info struct {
	Title       string
	Language    string
	Length      int
	Bytes       int64
	Captions    []string
	IsAutomatic []string
}

// ParseInfo reads the information yt-dlp prints for a video as JSON.
func ParseInfo(raw []byte) (Info, error) {
	var metadata struct {
		Title             string                `json:"title"`
		Language          string                `json:"language"`
		Duration          float64               `json:"duration"`
		Filesize          int64                 `json:"filesize"`
		Approximate       int64                 `json:"filesize_approx"`
		Subtitles         map[string][]struct{} `json:"subtitles"`
		AutomaticCaptions map[string][]struct{} `json:"automatic_captions"`
	}
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return Info{}, err
	}
	return Info{
		Title:       strings.TrimSpace(metadata.Title),
		Language:    metadata.Language,
		Length:      int(metadata.Duration * 1000),
		Bytes:       max(metadata.Filesize, metadata.Approximate),
		Captions:    getLanguages(metadata.Subtitles),
		IsAutomatic: getLanguages(metadata.AutomaticCaptions),
	}, nil
}

func getLanguages(tracks map[string][]struct{}) []string {
	out := make([]string, 0, len(tracks))
	for language := range tracks {
		out = append(out, language)
	}
	sort.Strings(out)
	return out
}

type published struct {
	Events []struct {
		Start    int `json:"tStartMs"`
		Duration int `json:"dDurationMs"`
		Appended int `json:"aAppend"`
		Segments []struct {
			Text string `json:"utf8"`
		} `json:"segs"`
	} `json:"events"`
}

// ParseCues reads the captions yt-dlp writes as JSON into cues in order, none overlapping.
func ParseCues(raw []byte) ([]Cue, error) {
	var captions published
	if err := json.Unmarshal(raw, &captions); err != nil {
		return nil, fmt.Errorf("read captions: %w", err)
	}
	cues := make([]Cue, 0, len(captions.Events))
	for _, event := range captions.Events {
		if event.Appended != 0 {
			continue
		}
		textBuilder := &strings.Builder{}
		for _, segment := range event.Segments {
			textBuilder.WriteString(segment.Text)
		}
		text := strings.Join(strings.Fields(textBuilder.String()), " ")
		if text == "" {
			continue
		}
		cues = append(cues, Cue{
			Text: text,
			From: event.Start,
			To:   event.Start + event.Duration,
		})
	}
	return trimOverlaps(cues), nil
}

func trimOverlaps(cues []Cue) []Cue {
	sort.SliceStable(cues, func(i, j int) bool { return cues[i].From < cues[j].From })
	for i := range cues {
		if i+1 < len(cues) && cues[i].To > cues[i+1].From {
			cues[i].To = cues[i+1].From
		}
		if cues[i].To < cues[i].From {
			cues[i].To = cues[i].From
		}
	}
	return cues
}
