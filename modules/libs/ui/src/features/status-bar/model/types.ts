/**
 * Types representing background tasks rendered in the status bar and details popover.
 */

export type TaskState = 'running' | 'failed' | 'cancelling' | 'completed'

export interface StatusBarTask {
  readonly id: string
  readonly label: string
  readonly group: string
  readonly detail?: string | undefined
  readonly progress?: number | undefined
  readonly currentCount?: number | undefined
  readonly totalCount?: number | undefined
  readonly unit?: string | undefined
  readonly etaSeconds?: number | undefined
  readonly state: TaskState
  readonly canCancel?: boolean | undefined
  readonly errorReason?: string | undefined
}

export interface TaskGroupData {
  readonly name: string
  readonly tasks: readonly StatusBarTask[]
}

export interface StatusBarAggregate {
  readonly activeCount: number
  readonly failedCount: number
  readonly overallProgress?: number | undefined
  readonly etaSeconds?: number | undefined
  readonly isIdle: boolean
}
