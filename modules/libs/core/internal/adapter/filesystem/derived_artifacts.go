package filesystem

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// reviewLogSuffix is what a file of review answers is named with.
const reviewLogSuffix = ".jsonl"

// extensions is what each file of a producer's work ends in. A text with the
// times it was said at is WebVTT, and opens in a player under that name.
var extensions = map[domain.ArtifactFile]struct{ prose, timed string }{
	domain.ArtifactText:        {".txt", ".vtt"},
	domain.ArtifactPartial:     {".partial", ".partial.vtt"},
	domain.ArtifactCorrections: {".corrected", ".corrected.vtt"},
	domain.ArtifactParts:       {".parts", ".parts"},
	domain.ArtifactBoxes:       {".boxes", ".boxes"},
	domain.ArtifactProofread:   {".proofread", ".proofread"},
	domain.ArtifactAnswer:      {".answer", ".answer"},
	domain.ArtifactProducer:    {".json", ".json"},
}

// getKey is the name an artifact is kept under inside the store: the hash,
// then the producer where the folder holds more than one, then what the file
// is. Every producer of a text with times shares one folder.
func getKey(name domain.ArtifactName) string {
	switch name.File {
	case domain.ArtifactCopy:
		return CopyDir + "/" + name.Hash + domain.CopyExtension
	case domain.ArtifactIndexClaim:
		return IndexDir + "/indexing"
	case domain.ArtifactReviewLog:
		return FlashcardsDir + "/" + name.Hash + reviewLogSuffix
	default:
		ext := extensions[name.File]
		if domain.IsTimedProducer(name.Producer) {
			return TranscriptDir + "/" + name.Hash + "." + name.Producer + ext.timed
		}
		return name.Producer + "/" + name.Hash + ext.prose
	}
}

// ReadArtifact returns what is kept under a name.
func (d *DerivedStore) ReadArtifact(ctx context.Context, name domain.ArtifactName) ([]byte, error) {
	return d.Read(ctx, getKey(name))
}

// OpenArtifact is one file of the store to read a part of.
func (d *DerivedStore) OpenArtifact(ctx context.Context, name domain.ArtifactName) (io.ReadSeekCloser, int64, error) {
	return d.Open(ctx, getKey(name))
}

// WriteArtifact puts content under a name.
func (d *DerivedStore) WriteArtifact(ctx context.Context, name domain.ArtifactName, content []byte) error {
	return d.Write(ctx, getKey(name), content)
}

// WriteArtifactStream puts what a reader gives under a name.
func (d *DerivedStore) WriteArtifactStream(
	ctx context.Context, name domain.ArtifactName, from io.Reader,
) (int64, error) {
	return d.WriteStream(ctx, getKey(name), from)
}

// AppendArtifact adds to what is under a name.
func (d *DerivedStore) AppendArtifact(ctx context.Context, name domain.ArtifactName, content []byte) error {
	return d.Append(ctx, getKey(name), content)
}

// RemoveArtifact takes a name out of the store.
func (d *DerivedStore) RemoveArtifact(ctx context.Context, name domain.ArtifactName) error {
	return d.Remove(ctx, getKey(name))
}

// ClaimArtifact holds a name until the returned function is called.
func (d *DerivedStore) ClaimArtifact(ctx context.Context, name domain.ArtifactName) (func() error, error) {
	return d.ClaimName(ctx, getKey(name))
}
