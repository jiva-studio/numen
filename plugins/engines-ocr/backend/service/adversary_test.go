package service

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
)

// A recognition the palette command runs is a recognition of the file in progress. A tool call for
// the same file must not start a second one.
func TestRecognitionCreateOfAScanThePaletteCommandRecognisesReportsItRunning(t *testing.T) {
	s := newToolScene(t)
	done := make(chan error, 1)
	go func() {
		_, err := newRecogniseCommand(s.tools.recognitions)(t.Context(), command.Request{Command: "recognition.create", Path: scanPath})
		done <- err
	}()
	s.waitEntered(t)

	out := s.recognise(t, "Physics", scanPath)

	if out.State != stateRunning {
		t.Errorf("recognition_create while the command recognises the scan returned %+v, want the state running", out)
	}
	close(s.recogniser.release)
	if err := <-done; err != nil {
		t.Errorf("the command: %v", err)
	}
	s.waitFinished(t, out.TaskID)
	s.waitFinished(t, out.TaskID)
	t.Logf("the scan was recognised twice: the command's run and the tool's run both finished the task %s", out.TaskID)
}

// The task id names the file alone, and the host lists a plugin's tasks by that id, so two vaults
// holding a file at the same path share one task.
func TestRecognitionCreateInTwoVaultsGivesTwoTaskIDs(t *testing.T) {
	s := newToolScene(t)
	raw, _ := s.vault.Content(scanPath)
	s.chemistry.File(scanPath, raw)

	inPhysics := s.recognise(t, "Physics", scanPath)
	s.waitEntered(t)
	inChemistry := s.recognise(t, "Chemistry", scanPath)

	if inPhysics.TaskID == inChemistry.TaskID {
		tasks := s.queue.ListTasks()
		t.Errorf("both recognitions report the task %s; the list holds %d task(s) %+v for two runs",
			inPhysics.TaskID, countTasks(tasks, inPhysics.TaskID), tasks)
	}
	close(s.recogniser.release)
}

func countTasks(tasks []host.Task, id string) int {
	n := 0
	for _, task := range tasks {
		if task.ID == id {
			n++
		}
	}
	return n
}
