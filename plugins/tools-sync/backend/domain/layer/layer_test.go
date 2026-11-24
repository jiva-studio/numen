package layer_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/layer"
)

func TestClassifyLayer(t *testing.T) {
	for name, one := range map[string]struct {
		path string
		size int64
		mime string
		want layer.Layer
	}{
		"a note":                            {"Shopping list.md", 2 << 10, "text/markdown", layer.AtOnce},
		"a note of 80 MiB":                  {"Journal.md", 80 << 20, "text/markdown", layer.AtOnce},
		"a note named with capitals":        {"Journal.MD", 1 << 10, "", layer.AtOnce},
		"a small picture":                   {"Pictures/Lake.png", 300 << 10, "image/png", layer.AtOnce},
		"a picture of exactly 5 MiB":        {"Pictures/Lake.jpg", layer.AtOnceImageLimit, "image/jpeg", layer.AtOnce},
		"a picture of 5 MiB and one byte":   {"Pictures/Lake.jpg", layer.AtOnceImageLimit + 1, "image/jpeg", layer.OnTap},
		"a document":                        {"Books/Guide.pdf", 5 << 20, "application/pdf", layer.OnTap},
		"a small document":                  {"Books/Leaflet.pdf", 4 << 10, "application/pdf", layer.OnTap},
		"a book":                            {"Books/Novel.epub", 1 << 20, "application/epub+zip", layer.OnTap},
		"a recording":                       {"Audio/Talk.mp3", 1 << 20, "audio/mpeg", layer.OnTap},
		"a film":                            {"Video/Trip.mp4", 1 << 20, "video/mp4", layer.OnTap},
		"an archive":                        {"Backup.zip", 1 << 10, "application/zip", layer.OnTap},
		"a file that is not a note by name": {"notes.md.bak", 1 << 10, "application/octet-stream", layer.OnTap},
	} {
		if got := layer.ClassifyLayer(one.path, one.size, one.mime); got != one.want {
			t.Errorf("%s: got %d, want %d", name, got, one.want)
		}
	}
}
