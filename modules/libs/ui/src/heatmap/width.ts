/**
 * How wide something is, measured and kept measured.
 *
 * Apart from the drawing because a grid laid out to the room it has needs the
 * room as a number, and where that number comes from is not the grid's concern.
 */
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { Ref } from 'vue'

/**
 * The width of what the ref holds, measured when it is drawn and again whenever
 * it changes. A machine with no way to watch for that measures once.
 */
export function useWidth(held: Ref<HTMLElement | null>): Ref<number> {
  const room = ref(0)
  let watching: ResizeObserver | null = null

  onMounted(() => {
    if (!held.value) return
    room.value = held.value.clientWidth
    if (typeof ResizeObserver === 'undefined') return
    watching = new ResizeObserver(([one]) => {
      room.value = one?.contentRect.width ?? 0
    })
    watching.observe(held.value)
  })

  onBeforeUnmount(() => {
    watching?.disconnect()
    watching = null
  })

  return room
}
