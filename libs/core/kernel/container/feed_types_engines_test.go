package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestOcrTakesOnlyPdfAndImages(t *testing.T) {
	checkFeedTypes(t, "md.numen.engines.ocr", []domain.ContentType{pdfType, pngType}, []domain.ContentType{noteType, epubType, mp3Type, mp4Type})
}

func TestTranscriptionTakesOnlyAudioAndVideo(t *testing.T) {
	checkFeedTypes(t, "md.numen.engines.transcription", []domain.ContentType{mp3Type, mp4Type}, []domain.ContentType{noteType, pdfType, epubType, pngType})
}
