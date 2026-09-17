/** How much room a component has, and the only thing in the library that measures it. */

import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { Ref, ShallowRef } from 'vue'

import type { Size } from './geometry'

export interface Viewport {
  /**
   * Watch an element and be told every size it takes. Nothing is said about a
   * box with no width or no height: an element drawn out of sight has one, and
   * it is not a measurement. Calling what comes back stops the watching.
   */
  readonly watch: (of: HTMLElement, took: (size: Size) => void) => () => void
  /**
   * The room a component has when nobody hands it one, now and every time it
   * changes. Calling what comes back stops the watching.
   */
  readonly watchRoom: (took: (size: Size) => void) => () => void
}

export const browserViewport: Viewport = {
  watch: (of, took) => {
    if (typeof ResizeObserver === 'undefined') return () => {}

    const observer = new ResizeObserver(([entry]) => {
      const box = entry?.contentRect
      if (box && box.width > 0 && box.height > 0) {
        took({ width: box.width, height: box.height })
      }
    })
    observer.observe(of)
    return () => observer.disconnect()
  },

  watchRoom: (took) => {
    if (typeof window === 'undefined') return () => {}

    const onResize = () => took({ width: window.innerWidth, height: window.innerHeight })
    onResize()
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  },
}

/**
 * The room one element of a drawn component has, held for the component's
 * life: measured where it stands and again at every size it takes. A viewport
 * with no size is not a measurement, and a component drawn out of sight has
 * none until it is drawn where somebody can see it.
 */
export function useViewport(area: Readonly<ShallowRef<HTMLElement | null>>): {
  viewport: Ref<Size>
  measure: () => void
} {
  const viewport = ref<Size>({ width: 0, height: 0 })
  let watching: (() => void) | undefined

  const measure = (): void => {
    if (!area.value) return
    const width = area.value.clientWidth
    const height = area.value.clientHeight
    if (width <= 0 || height <= 0) return
    viewport.value = { width, height }
  }

  onMounted(() => {
    measure()
    if (!area.value) return
    watching = browserViewport.watch(area.value, (size) => {
      viewport.value = size
    })
  })

  onBeforeUnmount(() => watching?.())

  return { viewport, measure }
}
