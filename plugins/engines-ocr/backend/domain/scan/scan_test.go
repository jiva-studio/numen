package scan_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/domain/scan"
)

func TestIsScanTakesPDFsAndImagesAlone(t *testing.T) {
	for file, want := range map[string]bool{
		"books/ohm.pdf": true, "books/OHM.PDF": true, "photos/board.jpeg": true, "photos/board.jpg": true,
		"photos/board.png": true, "photos/board.gif": true, "photos/board.webp": true,
		"notes/ohm.md": false, "audio/lecture.mp3": false, "books/ohm.pdf.part": false, "pdf": false,
	} {
		if got := scan.IsScan(file); got != want {
			t.Errorf("IsScan(%q) = %v, want %v", file, got, want)
		}
	}
}
