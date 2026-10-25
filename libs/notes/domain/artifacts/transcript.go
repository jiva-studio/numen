package artifacts

// A TranscriptCue is one stretch of speech: the transcript, when it was said,
// and at its current position in the words a transcript reads as. Offset is filled by
// reading, because only then is there a text for it to be an offset into.
type TranscriptCue struct {
	Text   string
	From   int
	To     int
	Offset int
}
