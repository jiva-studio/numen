package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestPdfReaderFeedTypes(t *testing.T) {
	checkFeedTypes(t, "md.numen.readers.pdf", []domain.ContentType{pdfType}, []domain.ContentType{noteType, epubType, pngType, mp3Type, mp4Type})
}

func TestEpubReaderFeedTypes(t *testing.T) {
	checkFeedTypes(t, "md.numen.readers.epub", []domain.ContentType{epubType}, []domain.ContentType{noteType, pdfType, pngType, mp3Type, mp4Type})
}
