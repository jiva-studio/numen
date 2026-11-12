/**
 * How wide something is, measured and kept up to date.
 *
 * Separate from the rendering because a grid laid out to its available width
 * needs the width as a number, and the source of that number is not the grid's concern.
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { Ref } from 'vue'

/**
 * The width of what the ref holds, measured when it is drawn and again whenever
 * it changes. A machine with no way to watch for that measures once.
 */
export function useWidth(element: Readonly<Ref<HTMLElement | null>>): Ref<number> {
  const room = ref(0)
  let watching: ResizeObserver | null = null

  onMounted(() => {
    if (!element.value) return
    // Measured to the fraction, the way the observer below measures, so its
    // first reading is not a change and what was drawn to this is not redrawn
    // under it.
    room.value = Number.parseFloat(getComputedStyle(element.value).width) || 0
    if (typeof ResizeObserver === 'undefined') return
    watching = new ResizeObserver(([one]) => {
      room.value = one?.contentRect.width ?? 0
    })
    watching.observe(element.value)
  })

  onBeforeUnmount(() => {
    watching?.disconnect()
    watching = null
  })

  return room
}
