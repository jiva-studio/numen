/**
 * The background work of a window as the plugin that lists it reads it: the tasks of the core and of
 * the plugins, and the failures the window and its plugins report, as failed tasks. Plain values.
 */
import type { CoreTask } from '@numen/core-client'
import type { BackgroundTask, BackgroundTaskState, Severity } from '@numen/plugin-sdk'

/** A task from the window, with the action its failure offers. */
export interface WorkTask extends CoreTask {
  readonly action?: { readonly label: string } | undefined
}

/** A message reported in a window, as a failure it can be. */
interface ReportedMessage {
  readonly id: string
  readonly text: string
  readonly about?: string | undefined
  readonly severity?: Severity | undefined
}

/** A failure the window reports, as a failed task under that identity. */
export const createFailedTask = (id: string, text: string, about = ''): WorkTask => ({
  id,
  label: text,
  about,
  progress: { completedCount: 0, total: 0, counting: 'items' },
  error: text,
  source: { isUserStarted: true, pluginId: '', pluginName: '' },
  isCancellable: false,
  kind: 'work',
})

/** The messages at warning or error level that say something, as failed tasks titled with their text. */
export const getFailedTasks = (messages: readonly ReportedMessage[]): WorkTask[] =>
  messages
    .filter((one) => (one.severity === 'error' || one.severity === 'warning') && one.text !== '')
    .map((one) => createFailedTask(one.id, one.text, one.about ?? ''))

function getState(task: WorkTask, isCancelling: boolean): BackgroundTaskState {
  if (task.error) return 'failed'
  if (task.kind === 'notice') return 'notice'
  return isCancelling ? 'cancelling' : 'running'
}

/** One task as the list draws it, with the fix of a failed one where it has one. */
export function createBackgroundTask(task: WorkTask, isCancelling: boolean): BackgroundTask {
  const hasTotal = task.progress.total > 0
  const isFailed = Boolean(task.error)
  return {
    id: task.id,
    label: task.label,
    about: task.about,
    state: getState(task, isCancelling),
    progress: {
      completedCount: hasTotal ? task.progress.completedCount : 0,
      total: hasTotal ? task.progress.total : 0,
      counting: task.progress.counting === 'items' ? '' : task.progress.counting,
    },
    error: task.error,
    allowed: { cancel: !isFailed && task.isCancellable, retry: false },
    actionLabel: isFailed ? (task.action?.label ?? '') : '',
  }
}
