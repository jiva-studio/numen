/**
 * What the summary of the background tasks shows, and the order of the list.
 */
import type { BackgroundTask } from '@numen/plugin-sdk'

/** The counts the summary shows. */
export interface TaskSummary {
  readonly activeCount: number
  readonly failedCount: number
  readonly progress: number | undefined
  readonly isIdle: boolean
}

const isActive = (task: BackgroundTask): boolean =>
  task.state === 'running' || task.state === 'cancelling'

/** The percent of its total a task has done; undefined for a task without a total. */
export function getProgress(task: BackgroundTask): number | undefined {
  if (task.progress.total <= 0) return undefined
  return Math.round((task.progress.completedCount / task.progress.total) * 100)
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

/** The tasks of one busy period of the window, counted for the chip. */
export interface TaskBatch {
  readonly total: number
  readonly done: number
  readonly fraction: number
}

/**
 * The batch the chip counts: every task seen since the window was last idle, the ones that ended
 * as done, and how far the whole batch has come. A task that failed is neither done nor active.
 */
export function summarizeBatch(
  tasks: readonly BackgroundTask[],
  seen: ReadonlySet<string>,
): TaskBatch {
  const total = seen.size
  if (total === 0) return { total: 0, done: 0, fraction: 0 }
  const active = tasks.filter(isActive)
  const failedIds = new Set(tasks.filter((task) => task.state === 'failed').map((task) => task.id))
  const finishedCount = [...seen].filter(
    (id) => !failedIds.has(id) && !active.some((task) => task.id === id),
  ).length
  const partial = active.reduce((sum, task) => sum + (getProgress(task) ?? 0) / 100, 0)
  return { total, done: finishedCount, fraction: Math.min(1, (finishedCount + partial) / total) }
}
