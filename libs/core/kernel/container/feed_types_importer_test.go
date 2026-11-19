package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

func TestMediaImporterFeedTypes(t *testing.T) {
	checkFeedTypes(t, "md.numen.tools.media.importer", nil, []domain.ContentType{noteType, pdfType, epubType, pngType, mp3Type, mp4Type})
	for _, plugin := range getPluginManifests(t) {
		if plugin.ID == "md.numen.tools.media.importer" && !plugin.IsTakingFileTypes([]domain.FileType{{Content: noteType, Note: "link"}}) {
			t.Error("media importer ignores a link note")
		}
	}
}
