/**
 * The background work of a window as the plugin that lists it reads it: the tasks of the core and of
 * the plugins, and the failures the window and its plugins report, as failed tasks. Plain values.
 */
import type { Severity, TallyUnit } from './tally'

/**
 * One piece of work the application is doing behind the window. source names the plugin that runs it, and
 * no plugin where the core runs it.
 */
export interface WorkTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly progress: {
    readonly completedCount: number
    readonly total: number
    readonly counting: TallyUnit
  }
  readonly error: string
  readonly source: {
    readonly isUserStarted: boolean
    readonly pluginId: string
    readonly pluginName: string
  }
  readonly isCancellable: boolean
  readonly kind: 'work' | 'notice'
  readonly action?: BackgroundTaskAction | undefined
}

/** How a task stands: running, being cancelled, ended in failure, or a notice. */
export type BackgroundTaskState = 'running' | 'cancelling' | 'failed' | 'notice'

/** The fix a failed task offers; the label is empty where it offers none. */
export interface BackgroundTaskAction {
  readonly label: string
}

/** One task as the list draws it. `counting` is empty where the task counts items one by one. */
export interface BackgroundTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly state: BackgroundTaskState
  readonly progress: {
    readonly completedCount: number
    readonly total: number
    readonly counting: string
  }
  readonly error: string
  readonly allowed: { readonly cancel: boolean; readonly retry: boolean }
  /** The label of the fix a failed task offers; empty where it offers none. */
  readonly actionLabel: string
}

/** A message reported in a window, as a failure it can be. */
export interface ReportedMessage {
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
