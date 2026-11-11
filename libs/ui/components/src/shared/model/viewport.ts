import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { Ref } from 'vue'

import { browserViewport, type Viewport } from '../api/viewport'
import type { Size } from '../lib/math/geometry'

/**
 * Follows the element for the component's lifetime: it is observed from mount
 * and again whenever it is replaced by another element. `onFound` is called
 * with each element before observation of it begins.
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
  // Stryker disable next-line ObjectLiteral,StringLiteral: post flush timing aligns with Vue template ref updates
  watch(area, follow, { flush: 'post' })
  onBeforeUnmount(() => stop?.())
}

/**
 * The size of one element of a rendered component, kept for the component's
 * lifetime: measured at its current position and again at every size change.
 * A zero-sized viewport is not a measurement, so a component that is not
 * rendered has none until it is displayed.
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
 * The size of one element including padding and border, which is the extent
 * that adjacent content must clear, kept for the component's lifetime.
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
 * The available size of a component when no size is passed in, kept for the
 * component's lifetime and measured immediately. `onChange` is called with each
 * new size.
 */
export function useFreeSpace(
  port: Viewport = browserViewport,
  onChange?: (size: Size) => void,
): { room: Ref<Size> } {
  const room = ref<Size>({ width: 0, height: 0 })
  let stop: (() => void) | undefined

  onMounted(() => {
    stop = port.watchFreeSpace((size) => {
      room.value = size
      onChange?.(size)
    })
  })

  // Stryker disable next-line OptionalChaining: stop is assigned on mount before unmount runs
  onBeforeUnmount(() => stop?.())

  return { room }
}
