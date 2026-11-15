package container

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	noteType = domain.ContentType("text/markdown")
	pdfType  = domain.ContentType("application/pdf")
	epubType = domain.ContentType("application/epub+zip")
	pngType  = domain.ContentType("image/png")
	mp3Type  = domain.ContentType("audio/mpeg")
	mp4Type  = domain.ContentType("video/mp4")
)

func checkFeedTypes(t *testing.T, id string, takes, ignores []domain.ContentType) {
	t.Helper()
	for _, one := range getPluginManifests(t) {
		if one.ID != id {
			continue
		}
		for _, declared := range one.ActivationEvents {
			if declared == manifest.ActivationEventFilesChanged {
				t.Errorf("%s declares onFilesChanged without a mask", id)
			}
		}
		for _, kind := range takes {
			if !one.IsTakingFileTypes([]domain.ContentType{kind}) {
				t.Errorf("%s ignores %s", id, kind)
			}
		}
		for _, kind := range ignores {
			if one.IsTakingFileTypes([]domain.ContentType{kind}) {
				t.Errorf("%s takes %s", id, kind)
			}
		}
		return
	}
	t.Fatalf("plugin %s not found", id)
}

func TestTaxonomyFeedTypes(t *testing.T) {
	checkFeedTypes(t, "md.numen.tools.taxonomy", []domain.ContentType{noteType}, []domain.ContentType{pdfType, epubType, pngType, mp3Type, mp4Type})
}
