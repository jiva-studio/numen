/**
 * The work the application reports to the window while it goes on behind it.
 */
import type { TallyUnit } from '@numen/workbench'

/**
 * One piece of work the application is doing behind the window.
 *
 * Every work item is a Task, and the window renders the list it
 * receives.
 */
interface TaskProgress {
  readonly completedCount: number
  readonly total: number
  readonly counting: TallyUnit
}

/** Who started a task: the plugin that runs it, or none where the core does. */
interface TaskSource {
  readonly isUserStarted: boolean
  readonly pluginId: string
  readonly pluginName: string
}

export interface Task {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly progress: TaskProgress
  readonly error: string
  readonly source: TaskSource
  readonly isCancellable: boolean
  readonly kind: 'work' | 'notice'
}
