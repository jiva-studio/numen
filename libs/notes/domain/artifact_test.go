package domain_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// A sweep works through the files a producer writes, and a transcription writes
// no coordinates or parts.
func TestASweepOfATranscriptNamesWhatItSaved(t *testing.T) {
	want := []domain.ArtifactName{
		domain.NewArtifactName(domain.ArtifactText, domain.ASR, "abc123"),
		domain.NewArtifactName(domain.ArtifactPartial, domain.ASR, "abc123"),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.ASR, "abc123"),
		domain.NewArtifactName(domain.ArtifactProofread, domain.ASR, "abc123"),
		domain.NewArtifactName(domain.ArtifactAnswer, domain.ASR, "abc123"),
		domain.NewArtifactName(domain.ArtifactProducer, domain.ASR, "abc123"),
	}
	if got := domain.GetArtifactNames(domain.ASR, "abc123"); !slices.Equal(got, want) {
		t.Errorf("a sweep takes %v, want %v", got, want)
	}
}

// A sweep of a recognition takes the coordinates and the parts with the rest.
func TestASweepOfARecognitionNamesWhatItSaved(t *testing.T) {
	want := []domain.ArtifactName{
		domain.NewArtifactName(domain.ArtifactText, domain.OCR, "abc123"),
		domain.NewArtifactName(domain.ArtifactPartial, domain.OCR, "abc123"),
		domain.NewArtifactName(domain.ArtifactBoxes, domain.OCR, "abc123"),
		domain.NewArtifactName(domain.ArtifactParts, domain.OCR, "abc123"),
		domain.NewArtifactName(domain.ArtifactCorrections, domain.OCR, "abc123"),
		domain.NewArtifactName(domain.ArtifactProofread, domain.OCR, "abc123"),
		domain.NewArtifactName(domain.ArtifactProducer, domain.OCR, "abc123"),
	}
	if got := domain.GetArtifactNames(domain.OCR, "abc123"); !slices.Equal(got, want) {
		t.Errorf("a sweep takes %v, want %v", got, want)
	}
}

// The files written for an address are what each producer that reaches one
// writes.
func TestTheFilesOfAnAddressAreWhatEveryProducerSaved(t *testing.T) {
	texts := domain.GetAddressTextNames("abc123")
	for _, producer := range domain.GetAddressProducers() {
		for _, name := range domain.GetArtifactNames(producer, "abc123") {
			if !slices.Contains(texts, name) {
				t.Errorf("%v is not among the files written for an address", name)
			}
		}
	}
}
