package file_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/contenttype"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
)

func TestGetMediaTypeNamesTheTypeOfAFileByItsName(t *testing.T) {
	types := file.NewGetMediaType(contenttype.Open())

	if got := types.Execute("talks/A Lecture.MP3"); got != "audio/mpeg" {
		t.Errorf("type = %q, want audio/mpeg", got)
	}
	if got := types.Execute("notes/unnamed.zzz-unknown"); got != "" {
		t.Errorf("type = %q, want none for a name the table does not know", got)
	}
}
