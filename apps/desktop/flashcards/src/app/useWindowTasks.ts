/**
 * What is being done behind the window, such as a vault read into the index,
 * drawn in the window's list of notices. It is a stream because an indexing
 * begins without the page asking for one.
 */
import { onMounted, onUnmounted } from 'vue'
import { createFollower } from '@numen/ui'
import type { NoticeQueue } from '@numen/shell'
import { windowService } from '@/shared/clients'
import { TARGET } from './host'

/** The name the window's own notices stand under in the list. */
export const WINDOW_SOURCE = 'window'

export function useWindowTasks(notices: NoticeQueue) {
  let isOpen = true
  const abortController = new AbortController()
  const follow = createFollower({
    isOpen: () => isOpen,
    setLost: (text) => notices.show(WINDOW_SOURCE, text, 'alarm'),
    wait: (ms) => new Promise((then) => setTimeout(then, ms)),
  })

  onMounted(() => {
    void follow(
      () => windowService.watchTasks({ window: TARGET }, { signal: abortController.signal }),
      (response) => {
        notices.setTasks(
          WINDOW_SOURCE,
          response.tasks.map((task) => ({
            id: task.id,
            label: task.title,
            about: task.about,
            error: task.error,
            isUserStarted: task.isUserStarted,
          })),
        )
      },
    )
  })
  onUnmounted(() => {
    isOpen = false
    abortController.abort()
  })
}
