import { computed, type Ref } from 'vue'
import { createBackgroundTaskHost, type NoticeQueue } from '@numen/workbench'
import { runInBackground } from '@numen/ui'
import type { PluginHealthItem } from '@/entities/plugin'
import { cancelTask, type Task } from '@/entities/window'
import { formatError } from '@/shared/api/errors'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import { getHealthTasks, runHealthAction, type HealthTaskDeps } from './healthTasks'
import { useNoticeTasks, type NoticeTasksDeps } from './useNoticeTasks'

export interface WindowBackgroundTasksDeps extends HealthTaskDeps {
  readonly log: MessageLog
  readonly plugins: NoticeQueue
  readonly window: NoticeTasksDeps['window'] & { readonly tasks: Readonly<Ref<readonly Task[]>> }
  readonly settings: NoticeTasksDeps['settings']
  readonly health: Readonly<Ref<readonly PluginHealthItem[]>>
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
  const healthTasks = computed(() => getHealthTasks(deps.health.value, deps))
  return createBackgroundTaskHost({
    tasks: computed(() => [...deps.window.tasks.value, ...healthTasks.value, ...noticeTasks.value]),
    cancelTask,
    reportCancelError: (error) =>
      deps.writeMessage(`The task was not cancelled: ${formatError(error)}`, 'error'),
    runAction: (id) =>
      runInBackground(runHealthAction(id, deps.health.value, deps), (error) =>
        deps.writeMessage(formatError(error), 'error'),
      ),
  })
}
