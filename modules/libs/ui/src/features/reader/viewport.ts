/**
 * The viewport a document is read in, measured and measured again as it changes.
 *
 * A viewport with no size is not a measurement: a reader mounted out of sight
 * has none, and the row is laid out only against one that has a size.
 */
import { onBeforeUnmount, onMounted, ref, type Ref, type ShallowRef } from 'vue'
import type { Viewport } from './strip'

export interface ViewportState {
  /** The viewport, in CSS pixels. Nothing until something has been measured. */
  readonly viewport: Ref<Viewport>
  /** Take it again. The caller says when a reader is on screen. */
  readonly measure: () => void
}

export function useViewport(area: Readonly<ShallowRef<HTMLElement | null>>): ViewportState {
  const viewport = ref<Viewport>({ wide: 0, high: 0 })
  let watching: ResizeObserver | undefined

  const measure = (): void => {
    if (!area.value) return
    const wide = area.value.clientWidth
    const high = area.value.clientHeight
    if (wide <= 0 || high <= 0) return
    viewport.value = { wide, high }
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

  return { viewport, measure }
}
