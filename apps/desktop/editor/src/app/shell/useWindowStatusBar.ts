/**
 * Adapts domain background tasks into status bar representations and handles interactions.
 */
import { computed, ref, watch, type Ref } from 'vue'
import type { StatusBarTask } from '@numen/shell'
import type { Task } from '@/shared/notices/task'
import { WORDS as words } from '@/shared/words'

export interface WindowStatusBarDeps {
  tasks: Ref<readonly Task[]>
  /** Asks the core to stop the task; resolves once the core has cancelled it. */
  cancelTask?(id: string): Promise<void>
  /** Receives the error of a cancel the core did not accept. */
  reportCancelError?(error: unknown): void
  onRetry?(id: string): void
  /** Opens the settings of the plugin whose task action was pressed. */
  onAction?(pluginId: string): void
  resolveAction?: (task: Task) => { label?: string; pluginId?: string | undefined } | undefined
}

/** The group a task is listed under: the plugin that reported it, or the core. */
export function getTaskGroup(task: Task): string {
  return task.pluginName || task.pluginId || words.coreTasks
}

function getTaskCounting(counting: string): string | undefined {
  if (!counting || counting === 'items') return undefined
  return counting
}

function getTaskState(task: Task, isCancelling: boolean): StatusBarTask['state'] {
  if (task.error) return 'failed'
  return isCancelling ? 'cancelling' : 'running'
}

function createStatusBarTask(
  task: Task,
  isCancelling: boolean,
  action?: { label?: string; pluginId?: string | undefined },
): StatusBarTask {
  const hasTotal = task.total > 0
  const isFailed = Boolean(task.error)

  return {
    id: task.id,
    label: task.label,
    group: getTaskGroup(task),
    about: task.about || undefined,
    progress: hasTotal ? Math.round((task.completedCount / task.total) * 100) : undefined,
    done: hasTotal ? task.completedCount : undefined,
    total: hasTotal ? task.total : undefined,
    counting: getTaskCounting(task.counting),
    state: getTaskState(task, isCancelling),
    canCancel: !isFailed && task.isCancellable,
    error: isFailed ? task.error : undefined,
    actionLabel: action ? action.label : undefined,
    actionId: action ? action.pluginId : undefined,
  }
}

export function useWindowStatusBar({
  tasks,
  cancelTask,
  reportCancelError,
  onRetry,
  onAction,
  resolveAction,
}: WindowStatusBarDeps) {
  const dismissedIds = ref(new Set<string>())
  // A task stays in this set from the press of Cancel until the core takes it out of the list.
  const cancellingIds = ref(new Set<string>())

  const statusBarTasks = computed<readonly StatusBarTask[]>(() => {
    return tasks.value
      .filter((task) => !dismissedIds.value.has(task.id))
      .map((task): StatusBarTask => {
        const action = task.error && resolveAction ? resolveAction(task) : undefined
        return createStatusBarTask(task, cancellingIds.value.has(task.id), action)
      })
  })

  watch(tasks, (list) => {
    const listedIds = new Set(list.map((task) => task.id))
    for (const id of cancellingIds.value) {
      if (!listedIds.has(id)) cancellingIds.value.delete(id)
    }
  })

  async function onCancelTask(id: string) {
    if (!cancelTask || cancellingIds.value.has(id)) return
    cancellingIds.value.add(id)
    try {
      await cancelTask(id)
    } catch (error) {
      cancellingIds.value.delete(id)
      reportCancelError?.(error)
    }
  }

  function onRetryTask(id: string) {
    onRetry?.(id)
  }

  function onDismissTask(id: string) {
    dismissedIds.value.add(id)
  }

  function onActionTask(id: string) {
    const pluginId = tasks.value.find((task) => task.id === id)?.pluginId
    if (pluginId) onAction?.(pluginId)
  }

  return {
    tasks: statusBarTasks,
    onCancelTask,
    onRetryTask,
    onDismissTask,
    onActionTask,
  }
}
