package domain

// ArtifactKind is a text made from a file of the vault.
type ArtifactKind uint8

const (
	ArtifactKindRecognition ArtifactKind = iota + 1
	ArtifactKindRecognitionCorrected
	ArtifactKindTranscript
	ArtifactKindTranscriptCorrected
	ArtifactKindArticle
	ArtifactKindCopy
)

// AddressText is what the tool that reaches a URL's address finds there.
type AddressText uint8

const (
	// AddressTextNone is an address no tool reaches.
	AddressTextNone AddressText = iota
	// AddressTextCaptions is words published against a clock.
	AddressTextCaptions
	// AddressTextArticle is the prose a page is written around.
	AddressTextArticle
)

// GetArtifactKinds is every artifact a file of a kind can carry, in the order they are made.
//
// A book carries a recognition, a recording a transcript, and a URL carries what its address holds: a transcript and a copy where the site publishes captions, an article otherwise.
func GetArtifactKinds(kind SourceKind, text AddressText) []ArtifactKind {
	switch {
	case kind == KindBook:
		return []ArtifactKind{ArtifactKindRecognition, ArtifactKindRecognitionCorrected}
	case kind == KindRecording:
		return []ArtifactKind{ArtifactKindTranscript, ArtifactKindTranscriptCorrected}
	case kind != KindURL || text == AddressTextNone:
		return nil
	case text == AddressTextCaptions:
		return []ArtifactKind{ArtifactKindTranscript, ArtifactKindCopy}
	default:
		return []ArtifactKind{ArtifactKindArticle}
	}
}
