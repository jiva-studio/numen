import { computed, type Ref } from 'vue'
import type { NoticeQueue } from '@numen/workbench'
import { cancelTask, type Task } from '@/entities/window'
import { formatError } from '@/shared/api/errors'
import type { MessageLog, MessageWriter } from '@/shared/model/notices/messages'
import { createBackgroundTaskHost } from './backgroundTasks'
import { resolveTaskAction } from './taskActions'
import { useNoticeTasks, type NoticeTasksDeps } from './useNoticeTasks'

export interface WindowBackgroundTasksDeps {
  readonly log: MessageLog
  readonly plugins: NoticeQueue
  readonly window: NoticeTasksDeps['window'] & { readonly tasks: Readonly<Ref<readonly Task[]>> }
  readonly settings: NoticeTasksDeps['settings']
  readonly writeMessage: MessageWriter
  readonly openSettings: (pluginId: string) => void
}

/** The tasks of the window and its failures, as the background tasks a plugin lists and cancels. */
export function createWindowBackgroundTasks(deps: WindowBackgroundTasksDeps) {
  const noticeTasks = useNoticeTasks({
    log: deps.log,
    plugins: deps.plugins,
    window: deps.window,
    settings: deps.settings,
  })
  return createBackgroundTaskHost({
    tasks: computed(() => [...deps.window.tasks.value, ...noticeTasks.value]),
    cancelTask,
    reportCancelError: (error) =>
      deps.writeMessage(`The task was not cancelled: ${formatError(error)}`, 'error'),
    resolveAction: resolveTaskAction,
    openSettings: deps.openSettings,
  })
}
