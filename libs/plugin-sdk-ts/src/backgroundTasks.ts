/**
 * The background work of the window, as the plugin that shows it reads and acts on it: the tasks of
 * the core and of the plugins, and the notices of the window and its plugins as failed tasks.
 */
import type { Disposable } from './types'

// BackgroundTaskState is how a task stands: running, being cancelled, ended in failure, or a notice.
export type BackgroundTaskState = 'running' | 'cancelling' | 'failed' | 'notice'

// BackgroundTask is one piece of work the window lists.
export interface BackgroundTask {
  readonly id: string
  // What is being done, in the words it is shown by.
  readonly label: string
  // What it is being done to; empty for nothing in particular.
  readonly about: string
  readonly state: BackgroundTaskState
  // How much of the total is done; both 0 where there is no total.
  readonly done: number
  readonly total: number
  // What the total counts, such as `files`; empty for items.
  readonly counting: string
  // Why the task failed; empty while it has not.
  readonly error: string
  // Whether `cancel` stops it.
  readonly canCancel: boolean
  // Whether `retry` starts it again.
  readonly canRetry: boolean
  // The words of the link that opens the settings of the plugin a failure names, and that plugin;
  // empty where there is no link.
  readonly actionLabel: string
  readonly actionPluginId: string
}

// BackgroundTaskHost is the background work of the window.
export interface BackgroundTaskHost {
  // Calls the handler with the tasks at once and again on every change.
  watch(handler: (tasks: readonly BackgroundTask[]) => void): Disposable
  // Asks the core to stop the task; settles once it has, and rejects with why it did not.
  cancel(id: string): Promise<void>
  // Starts a failed task again.
  retry(id: string): void
  // Takes a task out of the list.
  dismiss(id: string): void
  // Opens the settings of the plugin the action of a task names.
  openSettings(pluginId: string): void
}
