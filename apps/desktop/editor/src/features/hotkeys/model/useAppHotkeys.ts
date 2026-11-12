/**
 * Manages global keyboard shortcuts for the window shell.
 */
import { onMounted, onUnmounted } from 'vue'

export function useAppHotkeys(onKeydown: (event: KeyboardEvent) => void) {
  onMounted(() => {
    globalThis.addEventListener('keydown', onKeydown)
  })

  onUnmounted(() => {
    globalThis.removeEventListener('keydown', onKeydown)
  })
}
