# Running plugins

Shows in the title bar how many plugins run, so you can see that Numen starts only the plugins it needs and that each one stops after its work.

<!-- more -->

## What You Can Do

- **Count the running plugins**: The title bar shows `◈ 3`, the number of plugins whose process runs. A plugin that has only a frontend has no process and is not counted.
- **See each plugin**: Click the item for the list. Press Escape or click elsewhere to close it.

For each plugin the list shows:

- **Its name and its health**: `Healthy`, or the text of its worst health item, the same item the plugin's page in the settings shows.
- **Running for**: how long ago its process started.
- **In use**: the calls, streams and tasks that keep it running, while it has any.
- **Idle for** and **Stops in**: once nothing keeps it running, how long it has been idle and how long until Numen stops it. Numen stops a plugin that has been idle for 3 minutes and looks for one every 30 seconds, so a plugin whose time has passed shows `Stops at the next check`.
- **Runs inside the core**: for a plugin Numen serves inside its own process.

The list updates as plugins start, stop, take work and finish it.

The plugin is off by default: turn it on in the settings of the plugins. It has no settings. It only reads the list and changes nothing.

## How It Works

The plugin has no backend. Its frontend watches `context.work.processes` of the SDK, which needs the permission `plugins:read`. The window follows `WatchPluginProcesses` of the core's `PluginService`, which sends the list at once and again on every change, at most once a second. The core takes the list from the supervisors of the plugin processes: the start time, the last time a lease was taken or returned, the leases held, and, for an idle plugin, the time from which the idle stop takes it. It adds the name from the manifest, the worst item of the health list and the time of the core when it took the list. Between two lists the plugin moves that time on once a second by a monotonic clock, so the times count without asking the core again.
