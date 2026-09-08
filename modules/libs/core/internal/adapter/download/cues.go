package download

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/transcript"
)

// published is the format a site hands its captions over in: one event to a
// stretch of speech, its text broken into the segments a player highlights one
// at a time.
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

// cued is the captions as cues.
//
// An event whose segments say nothing is the blank a site puts between two
// stretches, and an event marked as appending is the tail of the one before it
// drawn again. Neither is anything anybody said, so neither becomes a cue.
//
// A cue ends where the next begins, so that a moment on the player belongs to
// one stretch of speech.
func cued(raw []byte) ([]transcript.Cue, error) {
	var held published
	if err := json.Unmarshal(raw, &held); err != nil {
		return nil, fmt.Errorf("what the captions said: %w", err)
	}
	cues := make([]transcript.Cue, 0, len(held.Events))
	for _, event := range held.Events {
		if event.Appended != 0 {
			continue
		}
		said := &strings.Builder{}
		for _, segment := range event.Segments {
			said.WriteString(segment.Text)
		}
		text := strings.Join(strings.Fields(said.String()), " ")
		if text == "" {
			continue
		}
		cues = append(cues, transcript.Cue{
			Text: text,
			From: event.Start,
			To:   event.Start + event.Duration,
		})
	}
	return trimmed(cues), nil
}

// trimmed ends each cue where the next begins. A site draws two stretches at
// once while one is still being said, and a chunk cut from words that stand
// twice is a passage read back over its neighbour.
//
// What was said first stands first, whatever order it arrived in: the words are
// read as one text, and a moment on the player is found in them by looking
// forward.
func trimmed(cues []transcript.Cue) []transcript.Cue {
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
