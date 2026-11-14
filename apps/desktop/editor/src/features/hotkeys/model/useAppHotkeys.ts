/**
 * Manages global keyboard shortcuts for the window shell.
 */
import { onMounted, onUnmounted } from 'vue'
import { win } from '@numen/ui'

export function useAppHotkeys(onKeydown: (event: KeyboardEvent) => void) {
  onMounted(() => {
    win.addEventListener('keydown', onKeydown)
  })

  onUnmounted(() => {
    win.removeEventListener('keydown', onKeydown)
  })
}
