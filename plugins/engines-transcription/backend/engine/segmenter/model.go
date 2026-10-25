// Package segmenter finds where in a recording somebody is speaking.
package segmenter

// Model is how the segmenter finds where in a recording somebody is speaking. What it cuts is
// what the transcriber is given, so where it cuts is part of what the words
// are.
type Model struct {
	// Name is what this segmenter is called in the record kept beside a text.
	Name string `json:"name"`
	// Repo is where the model is fetched from, and Path is a file on this
	// machine. A path is used as given; Repo is looked for in Dir first.
	Repo string `json:"from"`
	Path string `json:"path"`

	// Threshold is how sure the model has to be that a window carries speech.
	Threshold float32 `json:"threshold"`
	// Silence is how much quiet, in milliseconds, closes a segment.
	Silence int `json:"silence"`
	// Pad is how many milliseconds are kept on each side of a segment, so that
	// the first and last sound of a word are inside it.
	Pad int `json:"pad"`
	// Longest is how many milliseconds one segment may run to. Speech that goes
	// on longer is cut at the quietest window this side of the limit.
	Longest int `json:"longest"`
	// Shortest is how many milliseconds a segment carries to be a segment at
	// all.
	Shortest int `json:"shortest"`
	// Least is how many milliseconds a segment runs to before it stands as a
	// line of its own. A shorter one is put together with the segment after it.
	Least int `json:"least"`
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

// The model answers on the window a sound begins in, and the sound before that
// window is what the first letter of the word is made of.
func (s Model) getPadding() int {
	if s.Pad <= 0 {
		return 200
	}
	return s.Pad
}

// One segment is one run of the encoder, and its cost grows with its length.
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

// A line of a transcript is read, so it holds a phrase and not a breath. A
// speaker hesitating in the middle of a sentence stops for about this long.
func (s Model) getShortestPause() int {
	if s.Least <= 0 {
		return 2500
	}
	return s.Least
}
