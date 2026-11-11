/**
 * Opens an address that leads outside the window in the system browser. The
 * window has no address bar.
 */
import { onMounted, onUnmounted } from 'vue'
import { interceptExternalLinks } from '@numen/ui'
import { openExternalUrl } from '@/shared/api/wailsRuntime'
import type { NoticeQueue } from '@numen/workbench'
import { formatError, type Caught } from '@/shared/api/errors'
import { WINDOW_SOURCE } from '@/shared/lib/window'

export function useExternalLinks(notices: NoticeQueue) {
  let stop: (() => void) | undefined

  onMounted(() => {
    stop = interceptExternalLinks((href) => {
      openExternalUrl(href).catch((error: Caught) => {
        notices.show(WINDOW_SOURCE, formatError(error), 'error')
      })
    })
  })
  onUnmounted(() => {
    // Stryker disable next-line OptionalChaining: stop is assigned on mount before unmount runs
    stop?.()
  })
}
