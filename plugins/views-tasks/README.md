# Tasks

Shows the background tasks of the window in the title bar: whether any run, and the list of them with their progress.

<!-- more -->

## What You Can Do

- **See what the window is doing**: The title bar says **Idle** while no task runs, or how many tasks are done of how many, with a ring that fills as they progress. A task that failed tints the chip red and changes no text.
- **Open the list**: Click the summary for the list of tasks, newest first. Each row names the task, what it works on and how far it has got. Escape, a click outside or the close button closes the list.
- **Cancel a task**: Point at a running task and click the cross. The row shows the task as cancelling until the application has stopped it.
- **Clear a failure or a notice**: A failed task and a notice have a button that takes them out of the list. A failed task that can be started again has a retry button instead.
- **Configure a plugin**: A failure that a setting of a plugin can repair has a link that opens the settings of that plugin.

The plugin is enabled by default. Disabled, no task status is shown anywhere in the window.

## How It Works

The manifest places the view `status` in the slot `titleBar.status` of `contributes.slots`, and the frontend registers it with `register.slotView`. The view reads the tasks through `host.work.backgroundTasks`: the tasks of the core and of the plugins, the messages of the window and the notices of the plugins at warning or error level as failed tasks. Cancel, retry, dismiss and the settings link are calls of the same host.
