package wire

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestDescribeTasksCarriesTheKind(t *testing.T) {
	described := describeTasks([]run.Task{
		{ID: "a", About: "a key is ignored", Kind: run.KindNotice},
		{ID: "b"},
	})

	if described[0].GetKind() != v1.TaskKind_TASK_KIND_NOTICE || described[1].GetKind() != v1.TaskKind_TASK_KIND_UNSPECIFIED {
		t.Errorf("the kinds are %v and %v", described[0].GetKind(), described[1].GetKind())
	}
}
