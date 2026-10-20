package domain_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestGetArtifactKindsFollowTheKindAndTheAddress(t *testing.T) {
	for _, tc := range []struct {
		name string
		kind domain.SourceKind
		text domain.AddressText
		want []domain.ArtifactKind
	}{
		{"book", domain.KindBook, domain.AddressTextNone,
			[]domain.ArtifactKind{domain.ArtifactKindRecognition, domain.ArtifactKindRecognitionCorrected}},
		{"recording", domain.KindRecording, domain.AddressTextNone,
			[]domain.ArtifactKind{domain.ArtifactKindTranscript, domain.ArtifactKindTranscriptCorrected}},
		{"url with captions", domain.KindURL, domain.AddressTextCaptions,
			[]domain.ArtifactKind{domain.ArtifactKindTranscript, domain.ArtifactKindCopy}},
		{"url with a page", domain.KindURL, domain.AddressTextArticle,
			[]domain.ArtifactKind{domain.ArtifactKindArticle}},
		{"url no tool reaches", domain.KindURL, domain.AddressTextNone, nil},
		{"captions on a book", domain.KindBook, domain.AddressTextCaptions,
			[]domain.ArtifactKind{domain.ArtifactKindRecognition, domain.ArtifactKindRecognitionCorrected}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.GetArtifactKinds(tc.kind, tc.text); !slices.Equal(got, tc.want) {
				t.Errorf("GetArtifactKinds(%v, %v) = %v, want %v", tc.kind, tc.text, got, tc.want)
			}
		})
	}
}
