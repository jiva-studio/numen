package file_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/sharedmimeinfo"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
)

func TestGetMediaTypeNamesTheTypeOfAFileByItsName(t *testing.T) {
	types := file.NewGetMediaType(sharedmimeinfo.Open())

	if got := types.Execute("talks/A Lecture.MP3"); got != "audio/mpeg" {
		t.Errorf("type = %q, want audio/mpeg", got)
	}
	if got := types.Execute("notes/unnamed.zzz-unknown"); got != "" {
		t.Errorf("type = %q, want none for a name the table does not know", got)
	}
}
