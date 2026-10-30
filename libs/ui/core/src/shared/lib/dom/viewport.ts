/** How much room a component has, and the only thing in the library that measures it. */

import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Ref } from 'vue'

import type { Size } from '../math/geometry'

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
  /**
   * Watch an element and be told the room it takes up with its padding and its
   * edge, which is what anything standing clear of it has to clear. Calling
   * what comes back stops the watching.
   */
  readonly watchWhole: (of: HTMLElement, took: (size: Size) => void) => () => void
}

/** The stop function of a watch that has nothing to observe. */
const stopNothing = (): void => undefined

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
      const box = entry?.borderBoxSize?.[0]
      took({
        width: box?.inlineSize ?? of.offsetWidth,
        height: box?.blockSize ?? of.offsetHeight,
      })
    })
    observer.observe(of)
    return () => observer.disconnect()
  },

  watchRoom: (took) => {
    if (typeof window === 'undefined') return stopNothing

    const onResize = () => took({ width: window.innerWidth, height: window.innerHeight })
    onResize()
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  },
}

/**
 * Follows the element for the component's life: watched from the moment it
 * stands, and again whenever it is replaced by another. `onFound` hears of each
 * element before the watching of it begins.
 */
interface FollowOptions {
  readonly took: (size: Size) => void
  readonly onFound?: () => void
}

const useFollowed = (
  area: Readonly<Ref<HTMLElement | null>>,
  watchWith: Viewport['watch'],
  options: FollowOptions,
): void => {
  const { took, onFound } = options
  let followed: HTMLElement | null = null
  let stop: (() => void) | undefined

  const follow = (element: HTMLElement | null): void => {
    if (element === followed) return
    stop?.()
    stop = undefined
    followed = element
    if (!element) return
    onFound?.()
    stop = watchWith(element, took)
  }

  onMounted(() => follow(area.value))
  watch(area, follow, { flush: 'post' })
  onBeforeUnmount(() => stop?.())
}

/**
 * The room one element of a drawn component has, held for the component's
 * life: measured at its current position and again at every size it takes. A viewport
 * with no size is not a measurement, and a component drawn out of sight has
 * none until it is drawn where somebody can see it.
 */
export function useViewport(area: Readonly<Ref<HTMLElement | null>>): {
  viewport: Ref<Size>
  measure: () => void
} {
  const viewport = ref<Size>({ width: 0, height: 0 })

  const measure = (): void => {
    if (!area.value) return
    const width = area.value.clientWidth
    const height = area.value.clientHeight
    if (width <= 0 || height <= 0) return
    viewport.value = { width, height }
  }

  useFollowed(area, browserViewport.watch, {
    took: (size) => {
      viewport.value = size
    },
    onFound: measure,
  })

  return { viewport, measure }
}

/**
 * The room one element takes up with its padding and its edge, which is what
 * anything standing clear of it has to clear, held for the component's life.
 */
export function useWholeViewport(area: Readonly<Ref<HTMLElement | null>>): {
  viewport: Ref<Size>
} {
  const viewport = ref<Size>({ width: 0, height: 0 })

  useFollowed(area, browserViewport.watchWhole, {
    took: (size) => {
      viewport.value = size
    },
  })

  return { viewport }
}

/**
 * The room a component has when nobody hands it one, held for the component's
 * life and measured at once. `onChange` hears of each size as it is taken.
 */
export function useRoom(
  port: Viewport = browserViewport,
  onChange?: (size: Size) => void,
): { room: Ref<Size> } {
  const room = ref<Size>({ width: 0, height: 0 })
  let stop: (() => void) | undefined

  onMounted(() => {
    stop = port.watchRoom((size) => {
      room.value = size
      onChange?.(size)
    })
  })

  onBeforeUnmount(() => stop?.())

  return { room }
}
