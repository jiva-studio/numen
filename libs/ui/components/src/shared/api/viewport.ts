/** The available size of a component, and the only measurement entry point in the library. */

import type { Size } from '../lib/math/geometry'

export interface Viewport {
  readonly watch: (of: HTMLElement, took: (size: Size) => void) => () => void
  readonly watchFreeSpace: (took: (size: Size) => void) => () => void
  readonly watchWhole: (of: HTMLElement, took: (size: Size) => void) => () => void
}

const stopNothing = (): void => undefined

const getFirstBorderBox = (
  sizes: readonly ResizeObserverSize[] | undefined,
): ResizeObserverSize | undefined => sizes?.[0]

export const browserViewport: Viewport = {
  watch: (of, took) => {
    if (typeof ResizeObserver === 'undefined') return stopNothing

    const observer = new ResizeObserver(([entry]) => {
      const box = entry?.contentRect
      if (box && box.width > 0 && box.height > 0) {
        took({ width: box.width, height: box.height })
      }
    })
    observer.observe(of)
    return () => observer.disconnect()
  },

  watchWhole: (of, took) => {
    if (typeof ResizeObserver === 'undefined') return stopNothing

    const observer = new ResizeObserver(([entry]) => {
      const box = getFirstBorderBox(entry?.borderBoxSize)
      took({
        width: box?.inlineSize ?? of.offsetWidth,
        height: box?.blockSize ?? of.offsetHeight,
      })
    })
    observer.observe(of)
    return () => observer.disconnect()
  },

  watchFreeSpace: (took) => {
    if (typeof window === 'undefined') return stopNothing

    const onResize = () => took({ width: window.innerWidth, height: window.innerHeight })
    onResize()
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  },
}
