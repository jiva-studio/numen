/**
 * Adapts domain background tasks into status bar representations and handles interactions.
 */
import { computed, ref, type Ref } from 'vue'
import type { StatusBarTask } from '@numen/ui'
import type { Task } from '@/shared/notices/task'

export interface WindowStatusBarDeps {
  tasks: Ref<readonly Task[]>
  onCancel?(id: string): void
  onRetry?(id: string): void
  onAction?(id: string): void
  resolveAction?: (task: Task) => { label?: string; pluginId?: string | undefined } | undefined
}

const GROUP_PATTERNS: readonly { readonly group: string; readonly pattern: RegExp }[] = [
  { group: 'Dependencies & Setup', pattern: /provision|download|runtime|model|dependency/i },
  { group: 'Indexing', pattern: /index|book|epub/i },
  { group: 'Audio & Transcription', pattern: /transcrib|audio|speech|hear/i },
  { group: 'Text Recognition', pattern: /ocr|recogni|pdf/i },
  { group: 'Vault Scanning', pattern: /vault|scan|watch/i },
]

/**
 * Derives a human-friendly category group for a given task.
 */
function getTaskGroup(label: string, about: string): string {
  const combined = `${label} ${about}`
  for (const item of GROUP_PATTERNS) {
    if (item.pattern.test(combined)) {
      return item.group
    }
  }
  return label || 'Background Tasks'
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
    group: getTaskGroup(task.label, task.about),
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
    onAction?.(id)
  }

  return {
    tasks: statusBarTasks,
    onCancelTask,
    onRetryTask,
    onDismissTask,
    onActionTask,
  }
}
