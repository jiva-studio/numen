/**
 * What the summary of the background tasks shows, and the order of the list.
 */
import type { BackgroundTask } from '@numen/plugin-sdk'

/** The counts the summary shows. */
export interface TaskSummary {
  readonly activeCount: number
  readonly failedCount: number
  /** The average percent done of the active tasks that have a total; undefined for none. */
  readonly progress: number | undefined
  /** No task is active and none failed. */
  readonly isIdle: boolean
}

const isActive = (task: BackgroundTask): boolean =>
  task.state === 'running' || task.state === 'cancelling'

/** The percent of its total a task has done; undefined for a task without a total. */
export function getProgress(task: BackgroundTask): number | undefined {
  if (task.total <= 0) return undefined
  return Math.round((task.done / task.total) * 100)
}

export function summarizeTasks(tasks: readonly BackgroundTask[]): TaskSummary {
  const active = tasks.filter(isActive)
  const failedCount = tasks.filter((task) => task.state === 'failed').length
  const counted = active.map(getProgress).filter((one) => one !== undefined)
  const progress =
    counted.length > 0
      ? Math.round(counted.reduce((sum, one) => sum + one, 0) / counted.length)
      : undefined
  return {
    activeCount: active.length,
    failedCount,
    progress,
    isIdle: active.length === 0 && failedCount === 0,
  }
}

/** The tasks newest first: the window lists them oldest first. */
export function orderTasks(tasks: readonly BackgroundTask[]): readonly BackgroundTask[] {
  return [...tasks].reverse()
}
