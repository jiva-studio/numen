package download

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

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

func parseCues(raw []byte) ([]Cue, error) {
	var captions published
	if err := json.Unmarshal(raw, &captions); err != nil {
		return nil, fmt.Errorf("what the captions said: %w", err)
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
