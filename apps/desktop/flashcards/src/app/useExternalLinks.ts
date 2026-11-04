/**
 * Opens an address that leads outside the window in the system browser. The
 * window has no address bar.
 */
import { onMounted, onUnmounted } from 'vue'
import { formatErrorMessage } from '@numen/core-client'
import { openExternalUrl } from '@numen/plugin-host'
import { interceptExternalLinks } from '@numen/ui'
import type { NoticeQueue } from '@numen/shell'
import { WINDOW_SOURCE } from './useWindowTasks'

export function useExternalLinks(notices: NoticeQueue) {
  let stop: (() => void) | undefined

  onMounted(() => {
    stop = interceptExternalLinks((href) => {
      openExternalUrl(href).catch((error: unknown) => {
        notices.show(WINDOW_SOURCE, formatErrorMessage(error), 'error')
      })
    })
  })
  onUnmounted(() => {
    // Stryker disable next-line OptionalChaining: stop is assigned on mount before unmount runs
    stop?.()
  })
}
