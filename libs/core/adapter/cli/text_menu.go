package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/core/usecase/run"
	"github.com/jiva-studio/numen/libs/core/usecase/textversion"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// errNoMenuEntry is a file whose menu offers no entry of the action asked for.
var errNoMenuEntry = errors.New("no enabled plugin offers it")

// runTranscribeCommand asks the plugin that reads the recording's type for its text, and waits for
// the plugin to finish.
func runTranscribeCommand(ctx context.Context, out io.Writer, deps Deps, args []string) error {
	if len(args) != 2 {
		return errors.New("usage: numen-cli transcribe <vault> <file>")
	}
	v, err := findVault(deps, args[0])
	if err != nil {
		return err
	}
	return runTextMenuEntry(ctx, out, deps, v, args[1], textversion.ActionRecognise)
}

// proofreadTextCommand asks a plugin that reads the format of the file's live text for a version
// made from it, and waits for the plugin to finish.
func proofreadTextCommand(ctx context.Context, out io.Writer, deps Deps, v domain.Vault, path string) error {
	return runTextMenuEntry(ctx, out, deps, v, path, textversion.ActionProofread)
}

// runTextMenuEntry carries out the first entry of the action the file's menu offers, and waits
// until the plugin carrying it out reports no task about the file. The file is then cut from its
// live text.
func runTextMenuEntry(
	ctx context.Context, out io.Writer, deps Deps, v domain.Vault, path string, action textversion.MenuAction,
) error {
	if deps.TextMenu == nil {
		return fmt.Errorf("%s %s: %w", action, path, errNoMenuEntry)
	}
	menu, err := deps.TextMenu(ctx, v)
	if err != nil {
		return err
	}
	defer closeIfOpen(menu.Close)

	entry, err := findMenuEntry(ctx, menu, v, path, action)
	if err != nil {
		return err
	}
	started := time.Now()
	tasks := menu.Tasks.Watch(ctx)
	fmt.Fprintf(out, "%s %s with %s\n", action, path, entry.PluginID)
	if err := menu.Menu.RunMenuEntry(ctx, v, path, entry); err != nil {
		return err
	}
	if err := waitForTasks(ctx, out, tasks, entry.PluginID, path); err != nil {
		return err
	}
	if menu.Cut != nil {
		if err := menu.Cut(ctx, v, path); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "%s %s in %s\n", action, path, time.Since(started).Round(time.Second))
	return nil
}

// findMenuEntry is the first entry of the action the file's menu offers.
func findMenuEntry(
	ctx context.Context, menu TextMenu, v domain.Vault, path string, action textversion.MenuAction,
) (textversion.MenuEntry, error) {
	entries, err := menu.Menu.GetMenu(ctx, v, path)
	if err != nil {
		return textversion.MenuEntry{}, err
	}
	for _, entry := range entries {
		if entry.Action == action {
			return entry, nil
		}
	}
	return textversion.MenuEntry{}, fmt.Errorf("%s %s: %w", action, path, errNoMenuEntry)
}

// waitForTasks prints how far the plugin's task about the file has got until the plugin reports
// none, and fails with the error a task stopped on.
func waitForTasks(ctx context.Context, out io.Writer, tasks <-chan []run.Task, pluginID, path string) error {
	isDisplayed := false
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case current := <-tasks:
			task, isListed := findPluginTask(current, pluginID, path)
			switch {
			case !isListed:
				if isDisplayed {
					fmt.Fprintln(out)
				}
				return nil
			case task.Error != "":
				if isDisplayed {
					fmt.Fprintln(out)
				}
				return fmt.Errorf("%s: %s", task.Title, task.Error)
			case task.Total > 0:
				fmt.Fprintf(out, "  %s: %d of %d\r", task.Title, task.Count, task.Total)
				isDisplayed = true
			}
		}
	}
}

// findPluginTask is the task the plugin reports about the file.
func findPluginTask(tasks []run.Task, pluginID, path string) (run.Task, bool) {
	for _, task := range tasks {
		if task.About == path && strings.HasPrefix(task.ID, run.GetPluginTaskPrefix(pluginID)) {
			return task, true
		}
	}
	return run.Task{}, false
}
