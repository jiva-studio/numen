package artifacts

import "github.com/jiva-studio/numen/modules/libs/notes/domain"

// The kinds an artifact is asked for by, which are the kinds the store files
// them under.
//
// A kind says what a thing is, and a producer says what made it. Words with the
// times they were said at are a transcript whether a model heard them or a site
// published them. An ocr carries where on which page each word stands, and the
// prose a page is written around carries no such thing, so they are two kinds.
const (
	KindRecognition = domain.OCR
	KindTranscript  = "transcript"
	KindArticle     = domain.Article
	KindCopy        = "copy"
)

// Artifact is one thing made from a file of the vault. It is addressed by the
// file it was made from and the kind it is.
type Artifact struct {
	Path     string
	Kind     string
	Producer string
	Bytes    int64
	// Format is the media type the artifact is read back as.
	Format string
}

// IsReadable says whether the artifact is words to read. A copy is bytes to play.
func (a Artifact) IsReadable() bool { return a.Kind != KindCopy }

// getArtifactFormat is the media type a producer's artifact is written as.
func getArtifactFormat(producer string) string {
	switch producer {
	case domain.ASR, domain.Captions:
		return "text/vtt"
	case domain.OCR:
		return "application/json"
	default:
		return "text/plain"
	}
}

// getArtifactKind is the kind a producer's text is.
func getArtifactKind(producer string) string {
	switch producer {
	case domain.ASR, domain.Captions:
		return KindTranscript
	case domain.Article:
		return KindArticle
	default:
		return KindRecognition
	}
}
