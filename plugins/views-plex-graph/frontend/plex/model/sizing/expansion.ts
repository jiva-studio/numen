/**
 * A pointer left on a box opens it at its current position, until the whole of
 * a title too long for it is visible. Every other box keeps its position.
 *
 * The opening is rendered frame by frame, off the same clock the plex moves on.
 */
import { computed } from 'vue'
import { lerp, type Size } from '../../lib/arrange'
import { browserClock, type Clock } from '../transition'
import { useExpansion } from './useExpansion'
import { getHandle, type PlacedNode } from '../../lib/node'

/** How long a pointer stays on a box before it opens, in milliseconds. */
export const EXPANSION_DELAY = 500

export interface ExpandedBox {
  readonly width: number
  readonly offset: number
}

export interface ExpandedBoxBounds {
  readonly viewport: Size
  readonly margin: number
}

/**
 * The box a node opens to: the space its whole title requires, clamped inside the window.
 * It grows about its own midpoint, and shifts back inside the window where the midpoint leaves no space to grow.
 * Returns null when the box already holds the title or the window cannot fit a larger box.
 */
export function getExpandedBox(
  node: PlacedNode,
  room: number,
  bounds: ExpandedBoxBounds,
): ExpandedBox | null {
  const { viewport, margin } = bounds
  const width = Math.min(room, viewport.width - 2 * margin)
  if (width <= node.width) return null

  const getFurthest = viewport.width / 2 - margin - width / 2
  const middle = Math.min(Math.max(node.x, -getFurthest), getFurthest)
  return { width, offset: middle - node.x }
}

function interpolateBox(node: PlacedNode, expanded: ExpandedBox | null, open: number): ExpandedBox {
  if (!expanded) return { width: node.width, offset: 0 }
  return {
    width: lerp(node.width, expanded.width, open),
    offset: lerp(0, expanded.offset, open),
  }
}

export interface OpenBoxOptions {
  readonly getNode: () => PlacedNode
  readonly getExpanded: () => ExpandedBox | null
  readonly on: () => string | null
  readonly delay: () => number
  readonly hasPill?: (() => boolean) | undefined
  readonly viewportWidth?: (() => number | undefined) | undefined
  readonly clock?: Clock | undefined
}

/**
 * handle is made of is all sizes, and so all tokens.
 *
 * `on` names what the attention is on, and nothing when it is on none.
 */
export function useOpenBox(options: OpenBoxOptions) {
  const {
    getNode: node,
    getExpanded: expanded,
    on,
    delay,
    hasPill = () => true,
    viewportWidth,
    clock = browserClock,
  } = options
  const open = useExpansion(on, delay, clock)
  const box = computed(() => interpolateBox(node(), expanded(), open.value))

  const handle = computed(() => {
    const { offset, width } = box.value
    const pos = getHandle({ ...node(), x: node().x + offset, width }, hasPill(), viewportWidth?.())
    return { x: pos.x + offset, y: pos.y }
  })

  return { open, box, handle }
}
