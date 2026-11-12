/**
 * What is being done behind the window, such as a vault read into the index,
 * drawn in the window's list of notices. It is a stream because an indexing
 * begins without the page asking for one.
 */
import { onMounted, onUnmounted } from 'vue'
import { createFollower } from '@numen/ui'
import type { NoticeQueue } from '@numen/workbench'
import { formatError, type Caught } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'
import { watchTasks } from '../api/tasks'

export function useWindowTasks(notices: NoticeQueue) {
  let isOpen = true
  const abortController = new AbortController()
  const follow = createFollower({
    isOpen: () => isOpen,
    setLost: (text) => notices.show(WINDOW_SOURCE, text, 'error'),
    wait: (ms) => new Promise((then) => setTimeout(then, ms)),
  })

  onMounted(() => {
    follow(
      () => watchTasks(abortController.signal),
      (tasks) => {
        notices.setTasks(WINDOW_SOURCE, tasks)
      },
    ).catch(
      // Stryker disable next-line all: createFollower catches stream errors and reports them through setLost
      (error: Caught) => notices.show(WINDOW_SOURCE, formatError(error), 'error'),
    )
  })
  onUnmounted(() => {
    isOpen = false
    abortController.abort()
  })
}
