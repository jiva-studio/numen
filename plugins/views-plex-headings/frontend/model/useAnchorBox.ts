/** The box an element is drawn in on the screen, measured when asked. */
import { shallowRef, type Ref } from 'vue'
import type { Box } from '../types'

export function useAnchorBox(element: Readonly<Ref<HTMLElement | null>>) {
  const box = shallowRef<Box | null>(null)

  const measure = () => {
    if (!element.value) return
    const { left, top, width, height } = element.value.getBoundingClientRect()
    box.value = { x: left, y: top, width, height }
  }

  return { box, measure }
}
