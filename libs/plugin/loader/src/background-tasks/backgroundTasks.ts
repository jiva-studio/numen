/**
 * The tasks of the window as the background tasks the plugins read: the work of the core and the
 * plugins, and the failures the window reports as failed tasks, with a cancel and a dismissal.
 */
import type { BackgroundTask, BackgroundTaskHost } from '@numen/plugin-sdk'
import { computed, ref, watch, type Ref } from 'vue'
import { createBackgroundTask, type WorkTask } from './backgroundTask'

interface BackgroundTaskHostDeps {
  readonly tasks: Readonly<Ref<readonly WorkTask[]>>
  /** Asks the core to stop the task; resolves once the core has cancelled it. */
  cancelTask(id: string): Promise<void>
  /** Receives the error of a cancel the core did not accept. */
  reportCancelError(error: unknown): void
  /** Runs the action of the task; a window whose tasks offer none leaves it out. */
  runAction?(id: string): void
}

export function createBackgroundTaskHost(deps: BackgroundTaskHostDeps): BackgroundTaskHost {
  const dismissedIds = ref(new Set<string>())
  // A task stays in this set from the press of Cancel until the core takes it out of the list.
  const cancellingIds = ref(new Set<string>())

  const list = computed<readonly BackgroundTask[]>(() =>
    deps.tasks.value
      .filter((task) => !dismissedIds.value.has(task.id))
      .map((task) => createBackgroundTask(task, cancellingIds.value.has(task.id))),
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
    // Nothing in a window starts a failed task again, so no task offers a retry.
    retry: () => undefined,
    dismiss: (id) => {
      dismissedIds.value.add(id)
    },
    runAction: (id) => deps.runAction?.(id),
  }
}
