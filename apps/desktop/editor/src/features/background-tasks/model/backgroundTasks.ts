/**
 * The tasks of the window as the background tasks the plugins read: the work of the core and the
 * plugins, and the notices of the window and its plugins as failed tasks.
 */
import { computed, ref, watch, type Ref } from 'vue'
import type { BackgroundTask, BackgroundTaskHost } from '@numen/plugin-loader'
import type { Task } from '@/entities/window'
import type { Caught } from '@/shared/api/errors'

/** The settings action of a failed task: the words of its link and the plugin it opens. */
interface TaskAction {
  readonly label: string
  readonly pluginId: string
}

export interface BackgroundTaskHostDeps {
  readonly tasks: Readonly<Ref<readonly Task[]>>
  /** Asks the core to stop the task; resolves once the core has cancelled it. */
  cancelTask(id: string): Promise<void>
  /** Receives the error of a cancel the core did not accept. */
  reportCancelError(error: Caught): void
  resolveAction(task: Task): TaskAction | undefined
  openSettings(pluginId: string): void
}

function getState(task: Task, isCancelling: boolean): BackgroundTask['state'] {
  if (task.error) return 'failed'
  if (task.kind === 'notice') return 'notice'
  return isCancelling ? 'cancelling' : 'running'
}

function createBackgroundTask(
  task: Task,
  isCancelling: boolean,
  action: TaskAction | undefined,
): BackgroundTask {
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
    action: { label: action?.label ?? '', pluginId: action?.pluginId ?? '' },
  }
}

export function createBackgroundTaskHost(deps: BackgroundTaskHostDeps): BackgroundTaskHost {
  const dismissedIds = ref(new Set<string>())
  // A task stays in this set from the press of Cancel until the core takes it out of the list.
  const cancellingIds = ref(new Set<string>())

  const list = computed<readonly BackgroundTask[]>(() =>
    deps.tasks.value
      .filter((task) => !dismissedIds.value.has(task.id))
      .map((task) =>
        createBackgroundTask(
          task,
          cancellingIds.value.has(task.id),
          task.error ? deps.resolveAction(task) : undefined,
        ),
      ),
  )

  watch(deps.tasks, (tasks) => {
    const listedIds = new Set(tasks.map((task) => task.id))
    for (const id of cancellingIds.value) {
      if (!listedIds.has(id)) cancellingIds.value.delete(id)
    }
    // A dismissal holds while the task is listed, so a later run of the same work under the same id shows.
    for (const id of dismissedIds.value) {
      if (!listedIds.has(id)) dismissedIds.value.delete(id)
    }
  })

  const cancel = async (id: string): Promise<void> => {
    if (cancellingIds.value.has(id)) return
    cancellingIds.value.add(id)
    try {
      await deps.cancelTask(id)
    } catch (error) {
      cancellingIds.value.delete(id)
      deps.reportCancelError(error)
      throw error
    }
  }

  return {
    watch: (handler) => {
      handler(list.value)
      const stop = watch(list, handler)
      return { dispose: stop }
    },
    cancel,
    // Nothing in the editor starts a failed task again, so no task offers a retry.
    retry: () => undefined,
    dismiss: (id) => {
      dismissedIds.value.add(id)
    },
    openSettings: (pluginId) => deps.openSettings(pluginId),
  }
}
