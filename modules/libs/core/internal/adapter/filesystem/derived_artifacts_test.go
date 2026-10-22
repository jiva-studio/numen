package filesystem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Where an artifact is kept on disk is what existing vaults hold, so each name
// lands at the path it always did.
func TestEveryArtifactIsCachedWhereExistingVaultsKeepIt(t *testing.T) {
	for _, one := range []struct {
		name domain.ArtifactName
		at   string
	}{
		{domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"), "transcript/abc123.asr.vtt"},
		{domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, "abc123"), "transcript/abc123.asr.partial.vtt"},
		{domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"), "transcript/abc123.asr.corrected.vtt"},
		{domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, "abc123"), "transcript/abc123.asr.proofread"},
		{domain.NewArtifactName(domain.ArtifactAnswer, domain.ASR, "abc123"), "transcript/abc123.asr.answer"},
		{domain.NewArtifactName(domain.ArtifactProducer, domain.ASR, "abc123"), "transcript/abc123.asr.json"},
		{domain.NewArtifactName(domain.ArtifactText, domain.Captions, "abc123"), "transcript/abc123.captions.vtt"},
		{domain.NewArtifactName(domain.ArtifactText, domain.OCR, "abc123"), "ocr/abc123.txt"},
		{domain.NewArtifactName(domain.ArtifactPartial, domain.OCR, "abc123"), "ocr/abc123.partial"},
		{domain.NewArtifactName(domain.ArtifactBoxes, domain.OCR, "abc123"), "ocr/abc123.boxes"},
		{domain.NewArtifactName(domain.ArtifactParts, domain.OCR, "abc123"), "ocr/abc123.parts"},
		{domain.NewArtifactName(domain.ArtifactCorrections, domain.OCR, "abc123"), "ocr/abc123.corrected"},
		{domain.NewArtifactName(domain.ArtifactProofread, domain.OCR, "abc123"), "ocr/abc123.proofread"},
		{domain.NewArtifactName(domain.ArtifactProducer, domain.OCR, "abc123"), "ocr/abc123.json"},
		{domain.NewArtifactName(domain.ArtifactText, domain.Article, "abc123"), "article/abc123.txt"},
		{domain.NewCopyName("abc123"), "copy/abc123.mp4"},
		{domain.NewIndexClaimName(), "index/indexing"},
		{domain.NewReviewLogName("01ARZ3"), "flashcards/01ARZ3.jsonl"},
	} {
		t.Run(one.at, func(t *testing.T) {
			_, root := store(t)
			derived, err := filesystem.OpenDerived(root, filesystem.Options{},
				filesystem.OCRDir, filesystem.TranscriptDir, filesystem.ArticleDir,
				filesystem.CopyDir, filesystem.IndexDir, filesystem.FlashcardsDir)
			if err != nil {
				t.Fatal(err)
			}
			if err := derived.WriteArtifact(t.Context(), one.name, []byte("kept")); err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(filepath.Join(root, filesystem.DefaultServiceDir, filepath.FromSlash(one.at)))
			if err != nil || string(raw) != "kept" {
				t.Errorf("%v is not kept at %s: %q, %v", one.name, one.at, raw, err)
			}
		})
	}
}
