import { onBeforeUnmount, watch } from 'vue'
import { win } from '@numen/ui'

/** Calls the listener on the Escape key, ahead of the page, for as long as the view is open. */
export function useWindowEscape(isOpen: () => boolean, onEscape: () => void): void {
  let detach: (() => void) | null = null

  const onKey = (event: KeyboardEvent): void => {
    if (event.key !== 'Escape') return
    event.preventDefault()
    onEscape()
  }

  const stop = (): void => {
    detach?.()
    detach = null
  }

  watch(isOpen, (open) => {
    if (!open) return stop()
    win.addEventListener('keydown', onKey, true)
    detach = () => win.removeEventListener('keydown', onKey, true)
  })

  onBeforeUnmount(stop)
}
