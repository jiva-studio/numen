/** The box an element is drawn in on the screen and the room of the window, measured when asked. */
import { onScopeDispose, ref, shallowRef, type Ref } from 'vue'
import type { Box, Size } from '../types'

export function useAnchorBox(element: Readonly<Ref<HTMLElement | null>>) {
  const box = shallowRef<Box | null>(null)
  const room = ref<Size>({ width: 0, height: 0 })

  const measure = () => {
    room.value = { width: window.innerWidth, height: window.innerHeight }
    if (!element.value) return
    const { left, top, width, height } = element.value.getBoundingClientRect()
    box.value = { x: left, y: top, width, height }
  }

  window.addEventListener('resize', measure)
  onScopeDispose(() => window.removeEventListener('resize', measure))

  return { box, room, measure }
}
