/**
 * The measured size of a menu and the position derived from it.
 *
 * Only the rendering measures the size, so it is passed in as a value and the
 * placement is computed from it.
 */
import { computed, ref, type ComputedRef } from 'vue'
import { placeMenu } from '../items/item'
import type { Position, Size } from '@/shared/lib/math/geometry'

export interface MenuPlacementOptions {
  /** The requested position, in the coordinates of the area the menu is drawn into. */
  readonly at: () => Position
  /** The area it is placed in, and the browser's own where there is none. */
  readonly viewport: () => Size | null
  /** Kept clear of that area's edges. */
  readonly margin: () => number
  readonly asking?: () => number
}

export interface MenuPlacementState {
  /** Where the menu is drawn. */
  readonly placed: ComputedRef<Position>
  /** How big the menu turned out to be, as the drawing measured it. */
  readonly setSize: (size: Size) => void
  readonly resetSize: () => void
  readonly menuStyle: ComputedRef<Record<string, string>>
}

export function useMenuPlacement(options: MenuPlacementOptions): MenuPlacementState {
  const size = ref<Size>({ width: 0, height: 0 })
  const openedWidth = ref(0)

  /** The area to stay inside. The browser's, unless a caller measures its own. */
  const room = computed<Size>(
    () => options.viewport() ?? { width: window.innerWidth, height: window.innerHeight },
  )

  const placement = computed(() =>
    placeMenu({
      at: options.at(),
      size: size.value,
      viewport: room.value,
      margin: options.margin(),
    }),
  )

  const menuStyle = computed(() => ({
    left: `${placement.value.x}px`,
    top: `${placement.value.y}px`,
    '--asking': `${options.asking?.() ?? 0}px`,
    '--opened-width': `${openedWidth.value}px`,
    '--room': options.viewport()
      ? `${Math.max(options.viewport()!.height - 2 * options.margin(), 0)}px`
      : `calc(100dvh - ${2 * options.margin()}px)`,
  }))

  const setSize = (now: Size): void => {
    size.value = now
    if (now.width > openedWidth.value) {
      openedWidth.value = now.width
    }
  }

  const resetSize = (): void => {
    openedWidth.value = 0
  }

  return { placed: placement, setSize, resetSize, menuStyle }
}
