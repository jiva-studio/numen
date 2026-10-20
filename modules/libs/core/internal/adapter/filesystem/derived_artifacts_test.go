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
func TestEveryArtifactIsKeptWhereExistingVaultsKeepIt(t *testing.T) {
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

// The files of review answers are listed by the run that wrote each, and a file
// of another kind in the folder is not one.
func TestReviewLogsAreListedByRun(t *testing.T) {
	_, root := store(t)
	derived, err := filesystem.OpenDerived(root, filesystem.Options{}, filesystem.FlashcardsDir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	for _, id := range []string{"02B", "01A"} {
		if err := derived.AppendArtifact(ctx, domain.NewReviewLogName(id), []byte("answer")); err != nil {
			t.Fatal(err)
		}
	}
	if err := derived.WriteArtifact(ctx, domain.NewReviewLogName("03C"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(root, filesystem.DefaultServiceDir, filesystem.FlashcardsDir, "notes.txt")
	if err := os.WriteFile(other, []byte("not an answer"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := derived.ListReviewLogs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Name.Hash != "01A" || got[1].Name.Hash != "02B" || got[2].Name.Hash != "03C" {
		t.Fatalf("listed %v, want the three runs in order", got)
	}
	if got[0].Size != len("answer") {
		t.Errorf("the first run is %d bytes, want %d", got[0].Size, len("answer"))
	}
}
