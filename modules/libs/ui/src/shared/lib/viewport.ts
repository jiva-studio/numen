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
}

/**
 * The room one element of a drawn component has, held for the component's
 * life: measured where it stands and again at every size it takes. A viewport
 * with no size is not a measurement, and a component drawn out of sight has
 * none until it is drawn where somebody can see it.
 */
export function useViewport(area: Readonly<ShallowRef<HTMLElement | null>>): {
  viewport: Ref<{ wide: number; high: number }>
  measure: () => void
} {
  const viewport = ref({ wide: 0, high: 0 })
  let watching: (() => void) | undefined

  const measure = (): void => {
    if (!area.value) return
    const wide = area.value.clientWidth
    const high = area.value.clientHeight
    if (wide <= 0 || high <= 0) return
    viewport.value = { wide, high }
  }

  onMounted(() => {
    measure()
    if (!area.value) return
    watching = browserViewport.watch(area.value, ({ width, height }) => {
      viewport.value = { wide: width, high: height }
    })
  })

  onBeforeUnmount(() => watching?.())

  return { viewport, measure }
}
