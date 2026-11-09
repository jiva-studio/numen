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
  readonly at: () => Position
  readonly viewport: () => Size | null
  readonly margin: () => number
  readonly asking?: () => number
}

export interface MenuPlacementState {
  readonly placed: ComputedRef<Position>
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

  const getRoom = (): string => {
    const viewport = options.viewport()
    return viewport
      ? `${Math.max(viewport.height - 2 * options.margin(), 0)}px`
      : `calc(100dvh - ${2 * options.margin()}px)`
  }

  const menuStyle = computed(() => ({
    left: `${placement.value.x}px`,
    top: `${placement.value.y}px`,
    '--asking': `${options.asking?.() ?? 0}px`,
    '--opened-width': `${openedWidth.value}px`,
    '--room': getRoom(),
  }))

  const setSize = (now: Size): void => {
    size.value = now
    // Stryker disable next-line EqualityOperator: assigning an equal width is a no-op
    if (now.width > openedWidth.value) {
      openedWidth.value = now.width
    }
  }

  const resetSize = (): void => {
    openedWidth.value = 0
  }

  return { placed: placement, setSize, resetSize, menuStyle }
}
