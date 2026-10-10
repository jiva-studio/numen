/**
 * Adapts domain background tasks into status bar representations and handles interactions.
 */
import { computed, ref, type Ref } from 'vue'
import type { StatusBarTask, TaskState } from '@numen/ui'
import { calculateAggregate } from '@numen/ui'
import type { Task } from '@/shared/notices/task'

export interface WindowStatusBarDeps {
  tasks: Ref<readonly Task[]>
  onCancel?(id: string): void
  onRetry?(id: string): void
}

const GROUP_PATTERNS: readonly { readonly group: string; readonly pattern: RegExp }[] = [
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

export function useWindowStatusBar({ tasks, onCancel, onRetry }: WindowStatusBarDeps) {
  const dismissedIds = ref(new Set<string>())

  const statusBarTasks = computed<readonly StatusBarTask[]>(() => {
    return tasks.value
      .filter((task) => !dismissedIds.value.has(task.id))
      .map((task): StatusBarTask => {
        const hasTotal = task.total > 0
        const progress = hasTotal ? Math.round((task.done / task.total) * 100) : undefined
        const isFailed = task.error !== ''
        const state: TaskState = isFailed ? 'failed' : 'running'
        const group = getTaskGroup(task.label, task.about)

        return {
          id: task.id,
          label: task.label,
          group,
          detail: task.about || undefined,
          progress,
          currentCount: hasTotal ? task.done : undefined,
          totalCount: hasTotal ? task.total : undefined,
          unit: task.counting === 'things' ? undefined : task.counting || undefined,
          state,
          canCancel: !isFailed,
          errorReason: isFailed ? task.error : undefined,
        }
      })
  })

  const aggregate = computed(() => calculateAggregate(statusBarTasks.value))

  function onCancelTask(id: string) {
    onCancel?.(id)
  }

  function onRetryTask(id: string) {
    onRetry?.(id)
  }

  function onDismissTask(id: string) {
    dismissedIds.value.add(id)
  }

  return {
    tasks: statusBarTasks,
    aggregate,
    onCancelTask,
    onRetryTask,
    onDismissTask,
  }
}
