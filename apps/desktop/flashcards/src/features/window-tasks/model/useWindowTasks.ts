/**
 * What is being done behind the window, such as a vault read into the index, with the failures the
 * window and its plugins report, as the background tasks the plugins list and cancel. It is a stream
 * because an indexing begins without the page asking for one.
 */
import { computed, onMounted, onUnmounted, shallowRef } from 'vue'
import { createBackgroundTaskHost, getFailedTasks, type WorkTask } from '@numen/plugin-loader'
import type { NoticeQueue } from '@numen/workbench'
import { formatError } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import { createStreamReader, type TaskApi } from '../api/tasks'

export function useWindowTasks(api: TaskApi, notices: NoticeQueue) {
  const { watchTasks, cancelTask } = api
  const tasks = shallowRef<readonly WorkTask[]>([])
  let isOpen = true
  const abortController = new AbortController()
  const showError = (text: string) => notices.show(WINDOW_SOURCE, text, 'error')
  const follow = createStreamReader({
    isOpen: () => isOpen,
    setLost: showError,
    wait: (ms) => new Promise((then) => setTimeout(then, ms)),
  })

  onMounted(() => {
    follow(
      () => watchTasks(abortController.signal),
      (list) => {
        tasks.value = list
      },
    ).catch(
      // Stryker disable next-line all: createStreamReader catches stream errors and reports them through setLost
      (error: unknown) => showError(formatError(error)),
    )
  })
  onUnmounted(() => {
    isOpen = false
    abortController.abort()
  })

  return createBackgroundTaskHost({
    tasks: computed(() => [...tasks.value, ...getFailedTasks(notices.notices.value)]),
    cancelTask,
    reportCancelError: (error) => showError(formatError(error)),
  })
}
