/**
 * Opens an address that leads outside the window through the Wails runtime
 * served beside the page. The window has no address bar.
 */
import { onMounted, onUnmounted } from 'vue'
import { formatErrorMessage } from '@numen/plugin-sdk'
import { interceptExternalLinks } from '@numen/ui'
import type { NoticeQueue } from '@numen/shell'
import { WINDOW_SOURCE } from './useWindowTasks'

const RUNTIME_PATH = '/wails/runtime.js'

/** The runtime module, or undefined where the page is not served by Wails. */
const wails = import(/* @vite-ignore */ RUNTIME_PATH).catch(() => undefined)

async function openExternal(href: string): Promise<void> {
  const runtime = await wails
  await runtime?.Browser.OpenURL(href)
}

export function useExternalLinks(notices: NoticeQueue) {
  let stop: (() => void) | undefined

  onMounted(() => {
    stop = interceptExternalLinks((href) => {
      openExternal(href).catch((error: unknown) => {
        notices.show(WINDOW_SOURCE, formatErrorMessage(error), 'error')
      })
    })
  })
  onUnmounted(() => {
    stop?.()
  })
}
