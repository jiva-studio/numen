/**
 * The work the application reports to the window while it goes on behind it.
 */
import type { TallyUnit } from '@numen/shell'

/**
 * One piece of work the application is doing behind the window.
 *
 * Every work item is a Task, and the window renders the list it
 * receives.
 */
export interface Task {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly completedCount: number
  readonly total: number
  readonly counting: TallyUnit
  readonly error: string
  readonly isUserStarted: boolean
  readonly pluginId: string
  readonly pluginName: string
  readonly isCancellable: boolean
  readonly kind: 'work' | 'notice'
}
