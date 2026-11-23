package container

import (
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/run"
	"testing"
	"testing/synctest"
)

func TestTwoSessionsSharingAHostCannotRunTwoLocalEngines(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		installation := pluginInstall{Endpoints: newAbsentEndpoints()}
		exec := getPluginExecution(nil)
		first := openPluginWork(installation, exec, run.NewTasks())
		lease, err := first.EnqueueWork(t.Context(), "ocr", false, nil)
		if err != nil {
			t.Fatal(err)
		}
		second := openPluginWork(installation, exec, run.NewTasks())
		waiting := make(chan struct{})
		finished := make(chan error, 1)
		go func() {
			turn, err := second.EnqueueWork(t.Context(), "embeddings", true, func() { close(waiting) })
			if err == nil {
				turn.Release()
			}
			finished <- err
		}()
		synctest.Wait()
		select {
		case <-waiting:
		default:
			t.Fatal("new session gave out a second turn")
		}
		lease.Release()
		if err := <-finished; err != nil {
			t.Fatal(err)
		}
	})
}
