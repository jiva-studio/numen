/**
 * The background tasks of the core and the plugins, in the terms of the windows: what each counts,
 * who started it, and whether it can be cancelled.
 */
import type { Client } from '@connectrpc/connect'
import type { WindowService } from '@numen/protocol'
import { TASK_KIND, TASK_UNIT, type TaskUnitCode } from '../clients'

/** What a count counts. Bytes are formatted as sizes and seconds as a clock; the rest is counted one by one. */
export type CoreTaskUnit = 'items' | 'bytes' | 'seconds'

/** How far a task has come. */
export interface CoreTaskProgress {
  readonly completedCount: number
  readonly total: number
  readonly counting: CoreTaskUnit
}

/** Who started a task: the plugin that runs it, or none where the core does. */
export interface CoreTaskSource {
  readonly isUserStarted: boolean
  readonly pluginId: string
  readonly pluginName: string
}

/** One piece of work the application is doing behind a window. */
export interface CoreTask {
  readonly id: string
  readonly label: string
  readonly about: string
  readonly progress: CoreTaskProgress
  readonly error: string
  readonly source: CoreTaskSource
  readonly isCancellable: boolean
  readonly kind: 'work' | 'notice'
}

/** The calls of WindowService the task list is read and acted on with. */
export type TaskRpc = Pick<Client<typeof WindowService>, 'watchTasks' | 'cancelTask'>

/** The tasks of one window. */
export interface TaskClient {
  /**
   * Everything the application is doing in the background, until the signal aborts. The whole
   * list is sent whenever any of it changes, and the first list is sent at once.
   */
  watchTasks(signal: AbortSignal): AsyncGenerator<readonly CoreTask[]>
  /**
   * Asks the core to stop a task listed as cancellable. It resolves once the core has cancelled
   * the work, and rejects for a task that is not listed or cannot be cancelled.
   */
  cancelTask(id: string): Promise<void>
}

/** Keyed by the schema; a unit it has no word for is counted one by one. */
const COUNTED: Record<TaskUnitCode, CoreTaskUnit> = {
  [TASK_UNIT.UNSPECIFIED]: 'items',
  [TASK_UNIT.THINGS]: 'items',
  [TASK_UNIT.BYTES]: 'bytes',
  [TASK_UNIT.SECONDS]: 'seconds',
}

const getCountingUnit = (unit: number): CoreTaskUnit =>
  (COUNTED as Partial<Record<number, CoreTaskUnit>>)[unit] ?? 'items'

/** The tasks of the window of that name, over the window service. */
export function createTaskClient(rpc: TaskRpc, window: string): TaskClient {
  return {
    watchTasks: async function* (signal) {
      for await (const update of rpc.watchTasks({ window }, { signal })) {
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
    },
    cancelTask: async (id) => {
      await rpc.cancelTask({ window, id })
    },
  }
}
