package domain

// The producers: what made a text. A recording is heard by a model here
// (ASR), a scan is read by one (OCR), and the address a link note points at
// brings back captions or an article.
const (
	ASR      = "asr"
	OCR      = "ocr"
	Captions = "captions"
	Article  = "article"
)

// The container a copy of a video is fetched and kept in, and what a player is
// told it is.
const (
	CopyExtension = ".mp4"
	CopyType      = "video/mp4"
)

// ArtifactFile is which file of a producer's work on one source a name is.
type ArtifactFile string

const (
	// ArtifactText is the recognised or heard text itself.
	ArtifactText ArtifactFile = "text"
	// ArtifactPartial is the text of a run still going.
	ArtifactPartial ArtifactFile = "partial"
	// ArtifactCorrections is the text as it was put right.
	ArtifactCorrections ArtifactFile = "corrections"
	// ArtifactParts is the headings a layout model named.
	ArtifactParts ArtifactFile = "parts"
	// ArtifactBoxes is the coordinates a model produced.
	ArtifactBoxes ArtifactFile = "boxes"
	// ArtifactProofread is how far a proofreading run got.
	ArtifactProofread ArtifactFile = "proofread"
	// ArtifactAnswer is what a recording gave where it gave no words.
	ArtifactAnswer ArtifactFile = "answer"
	// ArtifactProducer is which models produced the text.
	ArtifactProducer ArtifactFile = "producer"
	// ArtifactCopy is the bytes of a video, kept to be played.
	ArtifactCopy ArtifactFile = "copy"
	// ArtifactIndexClaim is the claim that one run is indexing a vault.
	ArtifactIndexClaim ArtifactFile = "index-claim"
	// ArtifactReviewLog is one run's file of card review answers.
	ArtifactReviewLog ArtifactFile = "review-log"
)

// ArtifactName says which file of a vault's derived store is meant, and not
// where it is kept. The Hash is the fingerprint of the source's bytes, and the
// id of the run for a review log.
type ArtifactName struct {
	File     ArtifactFile
	Producer string
	Hash     string
}

// NewArtifactName is a file of one producer's work on the source with this hash.
func NewArtifactName(file ArtifactFile, producer, hash string) ArtifactName {
	return ArtifactName{File: file, Producer: producer, Hash: hash}
}

// NewCopyName is the copy of the video at the address with this hash.
func NewCopyName(hash string) ArtifactName {
	return ArtifactName{File: ArtifactCopy, Hash: hash}
}

// NewIndexClaimName is the claim that a run is indexing the vault.
func NewIndexClaimName() ArtifactName { return ArtifactName{File: ArtifactIndexClaim} }

// NewReviewLogName is the file of the review run with this id.
func NewReviewLogName(id string) ArtifactName {
	return ArtifactName{File: ArtifactReviewLog, Hash: id}
}

// IsTimedProducer says a producer writes words with the times they were said
// at. Those are WebVTT and open in a player; everything else is prose.
func IsTimedProducer(producer string) bool { return producer == ASR || producer == Captions }

// GetAddressProducers are the producers that bring back what is at an address,
// in the order one of them is read: what a model here heard stands over what a
// site published.
func GetAddressProducers() []string { return []string{ASR, Captions, Article} }

// GetArtifactNames is every file one producer's work on these bytes is kept
// under. A transcription writes no coordinates or parts.
func GetArtifactNames(producer, hash string) []ArtifactName {
	files := []ArtifactFile{
		ArtifactText, ArtifactPartial, ArtifactBoxes, ArtifactParts,
		ArtifactCorrections, ArtifactProofread, ArtifactProducer,
	}
	if IsTimedProducer(producer) {
		files = []ArtifactFile{
			ArtifactText, ArtifactPartial, ArtifactCorrections,
			ArtifactProofread, ArtifactAnswer, ArtifactProducer,
		}
	}
	out := make([]ArtifactName, 0, len(files))
	for _, file := range files {
		out = append(out, NewArtifactName(file, producer, hash))
	}
	return out
}

// GetAddressTextNames is every file a producer wrote for one address, without
// the copy of the video.
func GetAddressTextNames(hash string) []ArtifactName {
	var out []ArtifactName
	for _, producer := range GetAddressProducers() {
		out = append(out, GetArtifactNames(producer, hash)...)
	}
	return out
}

// String is the name as a message writes it.
func (n ArtifactName) String() string {
	return string(n.File) + "/" + n.Producer + "/" + n.Hash
}
