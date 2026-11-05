package wire

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
)

func TestDescribeTasksCarriesTheNoticeFlag(t *testing.T) {
	described := describeTasks([]run.Task{
		{ID: "a", About: "a key is ignored", IsNotice: true},
		{ID: "b"},
	})

	if !described[0].GetIsNotice() || described[1].GetIsNotice() {
		t.Errorf("the notice flags are %v and %v", described[0].GetIsNotice(), described[1].GetIsNotice())
	}
}
