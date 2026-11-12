/**
 * The background work of the window, as the plugin that shows it reads and acts on it: the tasks of
 * the core and of the plugins, and the notices of the window and its plugins as failed tasks.
 */
import type { Disposable } from './types'

// BackgroundTaskState is how a task stands: running, being cancelled, ended in failure, or a notice.
export type BackgroundTaskState = 'running' | 'cancelling' | 'failed' | 'notice'

// BackgroundTaskProgress is how far a task has come. Both counts are zero for a task without a total, and
// counting is empty where the task counts items one by one.
export interface BackgroundTaskProgress {
  readonly completedCount: number
  readonly total: number
  readonly counting: string
}

// BackgroundTaskAllowed is what the user may do with a task.
export interface BackgroundTaskAllowed {
  readonly cancel: boolean
  readonly retry: boolean
}

// BackgroundTask is one piece of work the window lists.
export interface BackgroundTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly state: BackgroundTaskState
  readonly progress: BackgroundTaskProgress
  readonly error: string
  readonly allowed: BackgroundTaskAllowed
  // The label of the fix a failed task offers; empty where it offers none.
  readonly actionLabel: string
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
  // Runs the action of the task.
  runAction(id: string): void
}
