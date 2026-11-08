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
  readonly label: string
  readonly about: string
  readonly state: BackgroundTaskState
  readonly done: number
  readonly total: number
  readonly counting: string
  readonly error: string
  readonly canCancel: boolean
  readonly canRetry: boolean
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
