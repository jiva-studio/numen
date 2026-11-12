import { computed, type Ref } from 'vue'
import { createBackgroundTaskHost, type NoticeQueue } from '@numen/workbench'
import { runInBackground } from '@numen/ui'
import type { PluginHealthItem } from '@/entities/plugin'
import { cancelTask, type Task } from '@/entities/window'
import { formatError } from '@/shared/api/errors'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import { getHealthTasks, runHealthAction, type HealthTaskDeps } from './healthTasks'
import { useNoticeTasks, type NoticeTasksDeps } from './useNoticeTasks'

/** The health items of the plugins, and what their tasks read of the window. */
interface HealthSource extends HealthTaskDeps {
  readonly items: Readonly<Ref<readonly PluginHealthItem[]>>
}

export interface WindowBackgroundTasksDeps {
  readonly log: MessageLog
  readonly plugins: NoticeQueue
  readonly window: NoticeTasksDeps['window'] & { readonly tasks: Readonly<Ref<readonly Task[]>> }
  readonly settings: NoticeTasksDeps['settings']
  readonly health: HealthSource
  readonly writeMessage: MessageWriter
}

/**
 * The tasks of the window, its failures and the health items of the plugins at error severity, as
 * the background tasks a plugin lists and cancels.
 */
export function createWindowBackgroundTasks(deps: WindowBackgroundTasksDeps) {
  const noticeTasks = useNoticeTasks({
    log: deps.log,
    plugins: deps.plugins,
    window: deps.window,
    settings: deps.settings,
  })
  const healthTasks = computed(() => getHealthTasks(deps.health.items.value, deps.health))
  return createBackgroundTaskHost({
    tasks: computed(() => [...deps.window.tasks.value, ...healthTasks.value, ...noticeTasks.value]),
    cancelTask,
    reportCancelError: (error) =>
      deps.writeMessage(`The task was not cancelled: ${formatError(error)}`, 'error'),
    runAction: (id) =>
      runInBackground(runHealthAction(id, deps.health.items.value, deps.health), (error) =>
        deps.writeMessage(formatError(error), 'error'),
      ),
  })
}
