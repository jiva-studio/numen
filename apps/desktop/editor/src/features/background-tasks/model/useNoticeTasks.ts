/**
 * What the window and the plugins report as notices, as background tasks.
 *
 * The window draws no notices of its own. A notice at warning or error level becomes a failed task in
 * the status bar's list, titled with its text; an information notice is dropped.
 */
import { computed, type ComputedRef, type Ref } from 'vue'
import { createFailedTask, getFailedTasks, type NoticeQueue } from '@numen/workbench'
import type { MessageLog } from '@/shared/model/notices/messages'
import type { Task } from '@/entities/window'
import { WORDS as words } from '@/shared/ui/words'

export interface NoticeTasksDeps {
  log: MessageLog
  plugins: NoticeQueue
  window: {
    readonly lost: Readonly<Ref<string>>
    readonly unwatched: Readonly<Ref<string>>
    readonly error: Readonly<Ref<string>>
  }
  settings: { readonly appearance: { readonly state: { readonly lost: Readonly<Ref<string>> } } }
}

/** The failed tasks for the window's state, its messages and the notices of its plugins. */
export function useNoticeTasks({
  log,
  plugins,
  window,
  settings,
}: NoticeTasksDeps): ComputedRef<readonly Task[]> {
  return computed<readonly Task[]>(() => {
    const reason = window.lost.value || settings.appearance.state.lost.value
    const state: Task[] = [
      ...(window.unwatched.value
        ? [createFailedTask('unwatched', words.unwatched, window.unwatched.value)]
        : []),
      ...(window.error.value ? [createFailedTask('unread', words.unread, window.error.value)] : []),
      ...(reason ? [createFailedTask('lost', reason)] : []),
    ]
    return [
      ...state,
      ...getFailedTasks(log.messages.value),
      ...getFailedTasks(plugins.notices.value),
    ]
  })
}
