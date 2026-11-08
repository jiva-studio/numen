// Package segmenter finds the speech segments in a recording.
package segmenter

// Model finds the speech segments in a recording. Its cuts are the transcriber's
// input, so where it cuts is part of the resulting words.
type Model struct {
	Name string `json:"name"`
	Repo string `json:"from"`
	Path string `json:"path"`

	Threshold float32 `json:"threshold"`
	Silence   int     `json:"silence"`
	Pad       int     `json:"pad"`
	Longest   int     `json:"longest"`
	Shortest  int     `json:"shortest"`
	Least     int     `json:"least"`
}

func (s Model) getThreshold() float32 {
	if s.Threshold <= 0 {
		return 0.5
	}
	return s.Threshold
}

func (s Model) getSilenceLength() int {
	if s.Silence <= 0 {
		return 500
	}
	return s.Silence
}

func (s Model) getPadding() int {
	if s.Pad <= 0 {
		return 200
	}
	return s.Pad
}

func (s Model) getLongestSegment() int {
	if s.Longest <= 0 {
		return 30000
	}
	return s.Longest
}

func (s Model) getShortestSegment() int {
	if s.Shortest <= 0 {
		return 100
	}
	return s.Shortest
}

func (s Model) getShortestPause() int {
	if s.Least <= 0 {
		return 2500
	}
	return s.Least
}
