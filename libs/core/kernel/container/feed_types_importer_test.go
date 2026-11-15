package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
)

func TestMediaImporterTakesOnlyNotes(t *testing.T) {
	checkFeedTypes(t, "md.numen.tools.media.importer", []domain.ContentType{noteType}, []domain.ContentType{pdfType, epubType, pngType, mp3Type, mp4Type})
}
