package service

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/command"
	"github.com/jiva-studio/numen/plugins/tools-media-importer/backend/usecase/link"
)

func TestDownloadMediaDownloadsTheItemPicked(t *testing.T) {
	for item, want := range map[string]string{"audio": ".audio.wav", "video": ".video.wav", "": ".video.wav"} {
		t.Run("item "+item, func(t *testing.T) {
			commands, store, tasks := newMockCommandsWithLinks(t)
			made := runImport(t, commands, videoAddress)

			_, err := commands.RunCommand(t.Context(), command.Request{Command: commandDownloadMedia, Path: made.OpenedPath, ItemID: item})

			if err != nil {
				t.Fatal(err)
			}
			tasks.WaitForEnd(t, link.GetCopyTaskID("Mills", made.OpenedPath))
			files, err := store.List(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			for _, one := range files {
				if strings.HasSuffix(one.Name, ".wav") {
					names = append(names, one.Name)
				}
			}
			if len(names) != 1 || !strings.HasSuffix(names[0], want) {
				t.Errorf("Download media of the item %q keeps %v, want one copy ending in %s", item, names, want)
			}
		})
	}
}

func TestDownloadMediaRejectsAnUnknownItem(t *testing.T) {
	commands, _ := newMockCommands(t)
	made := runImport(t, commands, videoAddress)

	_, err := commands.RunCommand(t.Context(), command.Request{Command: commandDownloadMedia, Path: made.OpenedPath, ItemID: "subtitles"})

	if !errors.Is(err, command.ErrBadArguments) {
		t.Errorf("Download media of the item subtitles answered %v, want ErrBadArguments", err)
	}
}

func TestTheHandlersAnswerInOnePlainForm(t *testing.T) {
	commands, _, tasks := newMockCommandsWithLinks(t)
	made := runImport(t, commands, videoAddress)
	path := made.OpenedPath
	if made.Message != "Created "+path {
		t.Errorf("Import link answered %q", made.Message)
	}

	for _, step := range []struct {
		command, want, task string
	}{
		{commandCreateText, "Fetching the text of " + path, "text:Mills:" + path},
		{commandDownloadMedia, "Downloading the media of " + path, link.GetCopyTaskID("Mills", path)},
		{commandDeleteMedia, "Deleted the media of " + path, ""},
	} {
		answer, err := commands.RunCommand(t.Context(), command.Request{Command: step.command, Path: path})
		if err != nil || answer.Message != step.want {
			t.Fatalf("%s answered %q (%v), want %q", step.command, answer.Message, err, step.want)
		}
		if step.task != "" {
			tasks.WaitForEnd(t, step.task)
		}
	}
}
