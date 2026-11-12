/**
 * The background tasks of the core and the plugins, as the window lists them.
 */
import { TASK_KIND, TASK_UNIT, type TaskUnitCode } from '@numen/core-client'
import type { TallyUnit } from '@numen/workbench'
import { windowService } from '@/shared/api/clients'
import type { Task } from '../lib/task'
import { WINDOW } from './window'

/** What a piece of work counts, in the words the window uses. A unit it has no word for is counted one by one. */
const counted: Record<TaskUnitCode, TallyUnit> = {
  [TASK_UNIT.UNSPECIFIED]: 'items',
  [TASK_UNIT.THINGS]: 'items',
  [TASK_UNIT.BYTES]: 'bytes',
  [TASK_UNIT.SECONDS]: 'seconds',
}

const getCountingUnit = (unit: number): TallyUnit =>
  (counted as Partial<Record<number, TallyUnit>>)[unit] ?? 'items'

/**
 * Everything the application is doing in the background, until the caller aborts the signal.
 *
 * The whole list is sent whenever any of it changes, and the first list is sent immediately.
 * It is a stream because work can begin without a request from the window: for example an
 * agent reading a document, which the user sees here.
 */
export async function* watchTasks(signal: AbortSignal): AsyncGenerator<readonly Task[]> {
  for await (const update of windowService.watchTasks({ window: WINDOW }, { signal })) {
    yield update.tasks.map((task) => ({
      id: task.id,
      label: task.title,
      about: task.about,
      progress: {
        completedCount: Number(task.done),
        total: Number(task.total),
        counting: getCountingUnit(task.unit),
      },
      error: task.error,
      source: {
        isUserStarted: task.isUserStarted,
        pluginId: task.pluginId,
        pluginName: task.pluginName,
      },
      isCancellable: task.isCancellable,
      kind: task.kind === TASK_KIND.NOTICE ? 'notice' : 'work',
    }))
  }
}

/**
 * Asks the core to stop a task that `watchTasks` lists as cancellable. It resolves once the
 * core has cancelled the work; the task leaves the list when the work has stopped. It rejects
 * for a task that is not listed or cannot be cancelled.
 */
export async function cancelTask(id: string): Promise<void> {
  await windowService.cancelTask({ window: WINDOW, id })
}
