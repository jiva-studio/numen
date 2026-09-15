/**
 * How big a menu turned out to be, and where that size puts it.
 *
 * Only the drawing knows the size, so it arrives here as a value and the
 * placement is worked out from it.
 */
import { computed, ref, type ComputedRef } from 'vue'
import { placeMenu } from './item'
import type { Position, Size } from '@/shared/lib/geometry'

export interface MenuPlacementOptions {
  /** Where it was asked for, in the coordinates of the area it is drawn into. */
  readonly at: () => Position
  /** The area it is placed in, and the browser's own where there is none. */
  readonly viewport: () => Size | null
  /** Kept clear of that area's edges. */
  readonly margin: () => number
}

export interface MenuPlacementState {
  /** Where the menu is drawn. */
  readonly placed: ComputedRef<Position>
  /** How big the menu turned out to be, as the drawing measured it. */
  readonly setSize: (size: Size) => void
}

export function useMenuPlacement(options: MenuPlacementOptions): MenuPlacementState {
  const size = ref<Size>({ width: 0, height: 0 })

  /** The area to stay inside. The browser's, unless a caller measures its own. */
  const room = computed<Size>(
    () => options.viewport() ?? { width: window.innerWidth, height: window.innerHeight },
  )

  const placed = computed(() =>
    placeMenu({
      at: options.at(),
      size: size.value,
      viewport: room.value,
      margin: options.margin(),
    }),
  )

  const setSize = (now: Size): void => {
    size.value = now
  }

  return { placed, setSize }
}
