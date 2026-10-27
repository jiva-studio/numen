/**
 * Adapts domain background tasks into status bar representations and handles interactions.
 */
import { computed, ref, type Ref } from 'vue'
import type { StatusBarTask } from '@numen/shell'
import type { Task } from '@/shared/notices/task'
import { WORDS as words } from '@/shared/words'

export interface WindowStatusBarDeps {
  tasks: Ref<readonly Task[]>
  onCancel?(id: string): void
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

function createStatusBarTask(
  task: Task,
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
    state: isFailed ? 'failed' : 'running',
    canCancel: !isFailed,
    error: isFailed ? task.error : undefined,
    actionLabel: action ? action.label : undefined,
    actionId: action ? action.pluginId : undefined,
  }
}

export function useWindowStatusBar({
  tasks,
  onCancel,
  onRetry,
  onAction,
  resolveAction,
}: WindowStatusBarDeps) {
  const dismissedIds = ref(new Set<string>())

  const statusBarTasks = computed<readonly StatusBarTask[]>(() => {
    return tasks.value
      .filter((task) => !dismissedIds.value.has(task.id))
      .map((task): StatusBarTask => {
        const action = task.error && resolveAction ? resolveAction(task) : undefined
        return createStatusBarTask(task, action)
      })
  })

  function onCancelTask(id: string) {
    onCancel?.(id)
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
