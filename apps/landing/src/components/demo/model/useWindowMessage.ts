import { onBeforeUnmount, onMounted } from 'vue'
import { win } from '@numen/ui'

/** Calls the listener with each message the window receives, from mount to unmount. */
export function useWindowMessage(listener: (event: MessageEvent) => void): void {
  onMounted(() => win.addEventListener('message', listener))
  onBeforeUnmount(() => win.removeEventListener('message', listener))
}
