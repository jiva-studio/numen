/**
 * Types representing background tasks rendered in the status bar and details popover.
 */

export type TaskState = 'running' | 'failed' | 'cancelling' | 'completed'

export interface StatusBarTask {
  readonly id: string
  readonly label: string
  /** The name of the plugin that reported the task. Absent for the application's own tasks. */
  readonly group?: string | undefined
  readonly about?: string | undefined
  readonly progress?: number | undefined
  readonly done?: number | undefined
  readonly total?: number | undefined
  readonly counting?: string | undefined
  readonly etaSeconds?: number | undefined
  readonly state: TaskState
  readonly canCancel?: boolean | undefined
  readonly error?: string | undefined
  readonly actionLabel?: string | undefined
  readonly actionId?: string | undefined
}

export interface TaskGroupData {
  readonly name: string
  readonly tasks: readonly StatusBarTask[]
}

/** The tasks of the popover: the application's own first, then one group per plugin. */
export interface TaskListing {
  readonly coreTasks: readonly StatusBarTask[]
  readonly groups: readonly TaskGroupData[]
}

export interface StatusBarAggregate {
  readonly activeCount: number
  readonly failedCount: number
  readonly overallProgress?: number | undefined
  readonly etaSeconds?: number | undefined
  readonly isIdle: boolean
}
