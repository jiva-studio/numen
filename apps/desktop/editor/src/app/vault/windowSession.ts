/**
 * The window's own streams and requests to the core: focus, background tasks, quit and flush.
 */
import { TASK_KIND } from '@numen/core-client'
import { windowService, workspace } from '@/shared/api/clients'
import { WINDOW } from '@/shared/lib/window'
import { counted, flushResults } from './words'
import type { VaultPort } from '@/app/ports/vault'

type WindowSessionCore = Pick<
  VaultPort,
  'watchFocus' | 'watchTasks' | 'cancelTask' | 'watchQuit' | 'reportFlush'
>

export const windowSessionCore: WindowSessionCore = {
  watchFocus: (signal) => workspace.watchFocus({}, { signal }),
  async *watchTasks(signal) {
    for await (const update of windowService.watchTasks({ window: WINDOW }, { signal })) {
      yield update.tasks.map((task) => ({
        id: task.id,
        label: task.title,
        about: task.about,
        completedCount: Number(task.done),
        total: Number(task.total),
        counting: counted[task.unit] ?? 'items',
        error: task.error,
        isUserStarted: task.isUserStarted,
        pluginId: task.pluginId,
        pluginName: task.pluginName,
        isCancellable: task.isCancellable,
        kind: task.kind === TASK_KIND.NOTICE ? 'notice' : 'work',
      }))
    }
  },
  cancelTask: async (id) => {
    await windowService.cancelTask({ window: WINDOW, id })
  },
  watchQuit: (signal) => windowService.watchQuit({ window: WINDOW }, { signal }),
  reportFlush: async (token, outcome) => {
    await windowService.reportFlush({
      window: WINDOW,
      token,
      result: flushResults[outcome ?? 'nothing'],
    })
  },
}
