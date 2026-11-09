/**
 * What the window and the plugins report as notices, as background tasks.
 *
 * The window draws no notices of its own. A notice at warning or error level becomes a failed task in
 * the status bar's list, titled with its text; an information notice is dropped.
 */
import { computed, type ComputedRef } from 'vue'
import type { Notice, NoticeQueue } from '@numen/workbench'
import type { MessageLog } from '@/shared/notices/messages'
import type { Task } from '@/shared/notices/task'
import { WORDS as words } from '@/shared/words'
import type { useWindowDisplay } from '../window/useWindowDisplay'
import type { useSettings } from '../window/useSettings'

export interface NoticeTasksDeps {
  log: MessageLog
  plugins: NoticeQueue
  window: ReturnType<typeof useWindowDisplay>
  settings: ReturnType<typeof useSettings>
}

const createFailedTask = (id: string, text: string, about = ''): Task => ({
  id,
  label: text,
  about,
  completedCount: 0,
  total: 0,
  counting: 'items',
  error: text,
  isUserStarted: true,
  pluginId: '',
  pluginName: '',
  isCancellable: false,
  kind: 'work',
})

const isFailure = (notice: { severity?: string }): boolean =>
  notice.severity === 'error' || notice.severity === 'warning'

/** The failed tasks for the window's state, its messages and the notices of its plugins. */
export function useNoticeTasks({
  log,
  plugins,
  window,
  settings,
}: NoticeTasksDeps): ComputedRef<readonly Task[]> {
  return computed<readonly Task[]>(() => {
    const reason = window.lost.value || settings.appearance.lost.value
    const state: Task[] = [
      ...(window.unwatched.value
        ? [createFailedTask('unwatched', words.unwatched, window.unwatched.value)]
        : []),
      ...(window.error.value ? [createFailedTask('unread', words.unread, window.error.value)] : []),
      ...(reason ? [createFailedTask('lost', reason)] : []),
    ]
    const messages = log.messages.value
      .filter(isFailure)
      .map((one) => createFailedTask(one.id, one.text))
    const reported = plugins.notices.value
      .filter((one: Notice) => isFailure(one) && one.text !== '')
      .map((one) => createFailedTask(one.id, one.text, one.about ?? ''))
    return [...state, ...messages, ...reported]
  })
}
