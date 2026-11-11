// Package settings holds what the plugin transcribes with: the speech model, the segmenter and the numbers of each.
package settings

// SegmenterModel finds the speech segments in a recording. Its cuts are the transcriber's
// input, so where it cuts is part of the resulting words.
type SegmenterModel struct {
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

func (s SegmenterModel) GetThreshold() float32 {
	if s.Threshold <= 0 {
		return 0.5
	}
	return s.Threshold
}

func (s SegmenterModel) GetSilenceLength() int {
	if s.Silence <= 0 {
		return 500
	}
	return s.Silence
}

func (s SegmenterModel) GetPadding() int {
	if s.Pad <= 0 {
		return 200
	}
	return s.Pad
}

func (s SegmenterModel) GetLongestSegment() int {
	if s.Longest <= 0 {
		return 30000
	}
	return s.Longest
}

func (s SegmenterModel) GetShortestSegment() int {
	if s.Shortest <= 0 {
		return 100
	}
	return s.Shortest
}

func (s SegmenterModel) GetShortestPause() int {
	if s.Least <= 0 {
		return 2500
	}
	return s.Least
}
