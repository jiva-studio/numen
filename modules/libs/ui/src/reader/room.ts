/**
 * The room a document is read in, measured and measured again as it changes.
 *
 * A room with no size is not a measurement: a reader mounted out of sight has
 * none, and the row is laid out only against a room that has one.
 */
import { onBeforeUnmount, onMounted, ref, type Ref, type ShallowRef } from 'vue'
import type { Room } from './strip'

export interface RoomState {
  /** The room, in CSS pixels. Nothing until something has been measured. */
  readonly room: Ref<Room>
  /** Take it again. The caller says when a reader is on screen. */
  readonly measure: () => void
}

export function useRoom(area: Readonly<ShallowRef<HTMLElement | null>>): RoomState {
  const room = ref<Room>({ wide: 0, high: 0 })
  let watching: ResizeObserver | undefined

  const measure = (): void => {
    if (!area.value) return
    const wide = area.value.clientWidth
    const high = area.value.clientHeight
    if (wide <= 0 || high <= 0) return
    room.value = { wide, high }
  }

  onMounted(() => {
    measure()
    if (!area.value || typeof ResizeObserver === 'undefined') return
    watching = new ResizeObserver(measure)
    watching.observe(area.value)
  })

  onBeforeUnmount(() => {
    watching?.disconnect()
  })

  return { room, measure }
}
