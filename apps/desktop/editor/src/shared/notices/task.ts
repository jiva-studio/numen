/**
 * The work the application reports to the window while it goes on behind it.
 */
import type { TallyUnit } from '@numen/shell'

/**
 * One piece of work the application is doing behind the window.
 *
 * Every kind of work is represented as a Task, and the window renders the list it
 * receives.
 */
export interface Task {
  /** Name of the work; reporting the same work again replaces it. */
  readonly id: string
  /** What is being done, in the words it is to be shown by. */
  readonly label: string
  /** What it is being done to. */
  readonly about: string
  /** How far it has got, where there is a total to count against. */
  readonly completedCount: number
  readonly total: number
  /** What that count counts. */
  readonly counting: TallyUnit
  /** Why it stopped, when it stopped with an error. */
  readonly error: string
  /** Whether a user requested this and is waiting for confirmation that it began. */
  readonly isUserStarted: boolean
  /** The plugin that reported the work and its name; both empty for the core's own work. */
  readonly pluginId: string
  readonly pluginName: string
}
