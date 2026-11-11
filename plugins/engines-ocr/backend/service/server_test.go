package service

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	ocr "github.com/jiva-studio/numen/plugins/engines-ocr"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/settings"
)

func (s *toolFixture) runHandlerWithin(t *testing.T, req command.Request) (command.Result, error) {
	t.Helper()
	commands, err := newCommands(s.launcher)
	if err != nil {
		t.Fatal(err)
	}
	type answer struct {
		res command.Result
		err error
	}
	answers := make(chan answer, 1)
	go func() {
		res, err := commands.RunCommand(t.Context(), req)
		answers <- answer{res, err}
	}()
	select {
	case got := <-answers:
		return got.res, got.err
	case <-time.After(workDeadline):
		t.Fatalf("%s did not answer within %v while the file was recognized", req.Command, workDeadline)
		return command.Result{}, nil
	}
}

func TestTheHandlerOfCreateTextAnswersWhileTheTaskRecognises(t *testing.T) {
	s := newToolFixture(t)

	res, err := s.runHandlerWithin(t, command.Request{Command: "app:file.createText", Path: scanPath})

	if err != nil || res.Message != "recognizing "+scanPath {
		t.Fatalf("answered %+v (%v)", res, err)
	}
	s.waitStarted(t)
	close(s.recogniser.release)
	s.waitFinished(t, recognition.GetTaskID("Physics", scanPath))
	if !hasRecognition(t, s.texts, scanPath) {
		t.Error("the scan has no recognition once its task ended")
	}
}

func TestCreateTextFailsTheTaskOfAFailedRecognition(t *testing.T) {
	s := newToolFixture(t)
	s.recogniser.failure = errors.New("the page cannot be decoded")
	close(s.recogniser.release)

	_, err := s.runHandlerWithin(t, command.Request{Command: "app:file.createText", Path: scanPath})

	if err != nil {
		t.Errorf("a failed recognition was answered %v, want the failure on its task alone", err)
	}
	deadline := time.After(workDeadline)
	for !hasFailedTask(s.queue.Queue, recognition.GetTaskID("Physics", scanPath), "the page cannot be decoded") {
		select {
		case <-deadline:
			t.Fatalf("no task failed within %v; tasks %+v", workDeadline, s.queue.ListTasks())
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func hasFailedTask(queue *hosttest.Queue, id, text string) bool {
	for _, one := range queue.ListTasks() {
		if one.ID == id && strings.Contains(one.FailureReason, text) {
			return true
		}
	}
	return false
}

func TestTheHandlerOfCreateTextFailsBeforeAnyTask(t *testing.T) {
	s := newToolFixture(t)

	if _, err := s.runHandlerWithin(t, command.Request{Command: "app:file.createText", Path: "books/gone.pdf"}); err == nil {
		t.Error("a file that is not there was answered as started")
	}
	if tasks := s.queue.ListTasks(); len(tasks) != 0 {
		t.Errorf("the tasks %+v are shown for a file that is not there", tasks)
	}
	if _, err := s.runHandlerWithin(t, command.Request{Command: "app:version.delete", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("Delete text, which the core runs, answered %v, want ErrUnknownCommand", err)
	}
	if _, err := s.runHandlerWithin(t, command.Request{Command: "recognition.create", Path: scanPath}); !errors.Is(err, command.ErrUnknownCommand) {
		t.Errorf("the command of before answered %v, want ErrUnknownCommand", err)
	}
}

// Each model the manifest offers is one the engine runs, fetched from the engine's own address.
func TestEveryModelOfTheManifestIsOneTheEngineRuns(t *testing.T) {
	var manifest struct {
		Dependencies []struct {
			ID        string `json:"id"`
			SourceURL string `json:"sourceUrl"`
		} `json:"dependencies"`
		Settings []struct {
			Key     string `json:"key"`
			Options []struct {
				Value        string   `json:"value"`
				Dependencies []string `json:"dependencies"`
			} `json:"options"`
		} `json:"settings"`
	}
	if err := json.Unmarshal(ocr.Manifest, &manifest); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, one := range manifest.Dependencies {
		sources[one.ID] = one.SourceURL
	}
	count := 0
	for _, setting := range manifest.Settings {
		if setting.Key != "model" {
			continue
		}
		for _, option := range setting.Options {
			count++
			address, ok := settings.FindRecogniser(option.Value)
			if !ok {
				t.Errorf("the manifest offers %s, which the engine does not run", option.Value)
				continue
			}
			var provisioned []string
			for _, id := range option.Dependencies {
				provisioned = append(provisioned, sources[id])
			}
			if address != "" && (len(provisioned) != 1 || provisioned[0] != address) {
				t.Errorf("%s provisions %v, and the engine fetches %s", option.Value, provisioned, address)
			}
		}
	}
	if count == 0 {
		t.Fatal("the manifest offers no model")
	}
}
