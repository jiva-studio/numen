/** How much room a component has, and the only thing in the library that measures it. */

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
