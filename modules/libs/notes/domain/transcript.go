package domain

// A TranscriptCue is one stretch of speech: what was said, when it was said,
// and where it stands in the words a transcript reads as. Offset is filled by
// reading, because only then is there a text for it to be an offset into.
type TranscriptCue struct {
	Text   string
	From   int
	To     int
	Offset int
}
