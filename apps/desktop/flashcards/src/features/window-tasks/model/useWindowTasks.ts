/**
 * What is being done behind the window, such as a vault read into the index, with the failures the
 * window and its plugins report, as the background tasks the plugins list and cancel. It is a stream
 * because an indexing begins without the page asking for one.
 */
import { computed, onMounted, onUnmounted, shallowRef } from 'vue'
import { createFollower } from '@numen/ui'
import {
  createBackgroundTaskHost,
  getFailedTasks,
  type NoticeQueue,
  type WorkTask,
} from '@numen/workbench'
import { formatError, type Caught } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import { cancelTask, watchTasks } from '../api/tasks'

export function useWindowTasks(notices: NoticeQueue) {
  const tasks = shallowRef<readonly WorkTask[]>([])
  let isOpen = true
  const abortController = new AbortController()
  const showError = (text: string) => notices.show(WINDOW_SOURCE, text, 'error')
  const follow = createFollower({
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
      // Stryker disable next-line all: createFollower catches stream errors and reports them through setLost
      (error: Caught) => showError(formatError(error)),
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
