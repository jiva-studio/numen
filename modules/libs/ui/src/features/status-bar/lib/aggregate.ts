/**
 * Pure calculations for aggregate status bar progress and group arrangements.
 */
import type { StatusBarAggregate, StatusBarTask, TaskGroupData } from '../model/types'

interface ActiveAccumulator {
  determinateSum: number
  determinateCount: number
  maxEta: number | undefined
}

function accumulateTask(task: StatusBarTask, acc: ActiveAccumulator): void {
  if (task.progress !== undefined) {
    acc.determinateSum += task.progress
    acc.determinateCount++
  }
  if (task.etaSeconds && task.etaSeconds > 0) {
    acc.maxEta = acc.maxEta === undefined ? task.etaSeconds : Math.max(acc.maxEta, task.etaSeconds)
  }
}

/**
 * Calculates aggregate counts, combined progress, and maximum ETA across tasks.
 */
export function calculateAggregate(tasks: readonly StatusBarTask[]): StatusBarAggregate {
  let activeCount = 0
  let failedCount = 0
  const acc: ActiveAccumulator = { determinateSum: 0, determinateCount: 0, maxEta: undefined }

  for (const task of tasks) {
    if (task.state === 'running' || task.state === 'cancelling') {
      activeCount++
      accumulateTask(task, acc)
    } else if (task.state === 'failed') {
      failedCount++
    }
  }

  const isIdle = activeCount === 0 && failedCount === 0
  const overallProgress =
    acc.determinateCount > 0 ? Math.round(acc.determinateSum / acc.determinateCount) : undefined

  return {
    activeCount,
    failedCount,
    overallProgress,
    etaSeconds: acc.maxEta,
    isIdle,
  }
}

/**
 * Groups tasks by category name preserving order of first appearance.
 */
export function groupTasks(tasks: readonly StatusBarTask[]): readonly TaskGroupData[] {
  const map = new Map<string, StatusBarTask[]>()

  for (const task of tasks) {
    const list = map.get(task.group)
    if (list) {
      list.push(task)
    } else {
      map.set(task.group, [task])
    }
  }

  return Array.from(map.entries()).map(([name, groupTasks]) => ({
    name,
    tasks: groupTasks,
  }))
}

/**
 * Formats estimated remaining seconds into human-readable duration string.
 */
export function formatEta(seconds: number | undefined): string {
  if (seconds === undefined || seconds <= 0) {
    return ''
  }

  if (seconds < 60) {
    return `~${Math.round(seconds)}s`
  }

  if (seconds < 3600) {
    return `~${Math.ceil(seconds / 60)}m`
  }

  return `~${(seconds / 3600).toFixed(1)}h`
}
