/**
 * A hand left on a box opens it where it stands, until the whole of a title too
 * long for it is there. Every other box keeps its place.
 *
 * The opening is drawn frame by frame, off the same clock the plex moves on.
 */
import { computed, onScopeDispose, ref, watch, type Ref } from 'vue'
import { easeOut, lerp, type Size } from '../lib/arrange'
import { browserClock, type Clock } from './transition'
import { handleIn, type PlacedNode } from '../lib/node'

/** How long a hand stays on a box before it opens, in milliseconds. */
export const DWELL = 500

/**
 * How long the opening itself takes, in milliseconds. It carries the places
 * inside the node out from under the box as well as the width, and they leave
 * one behind the next.
 */
export const OPENING = 280

/** A box drawn wider than it was placed. */
export interface WideBox {
  readonly width: number
  /** How far its middle stands from where the node is placed. */
  readonly offset: number
}

/**
 * The box a node opens to: the room its whole title asks for, held inside the
 * window. It grows about its own middle, and slides back inside the window
 * where the middle leaves it no room to grow.
 *
 * Nothing where the box already holds the title, and nothing where the window
 * is no wider than the box.
 */
export function getWideBox(
  node: PlacedNode,
  wanted: number,
  viewport: Size,
  margin: number,
): WideBox | null {
  const width = Math.min(wanted, viewport.width - 2 * margin)
  if (width <= node.width) return null

  const furthest = viewport.width / 2 - margin - width / 2
  const middle = Math.min(Math.max(node.x, -furthest), furthest)
  return { width, offset: middle - node.x }
}

/**
 * The box as it is drawn: the one the node was placed with, carried towards the
 * one it opens to as far as it has got. A node with nothing more of its title
 * to show is drawn as it was placed, however long the attention rests on it.
 */
export function boxOf(node: PlacedNode, wide: WideBox | null, open: number): WideBox {
  if (!wide || open <= 0) return { width: node.width, offset: 0 }
  return {
    width: lerp(node.width, wide.width, open),
    offset: lerp(0, wide.offset, open),
  }
}

/**
 * How far open a box stands, from nothing at all to the whole way: shut until
 * the attention has been on it for the wait, then open, and shut again the
 * moment the attention leaves.
 *
 * `on` names what the attention is on, and nothing when it is on none. A thing
 * that moves is another thing: the box shuts and the wait for it begins again.
 * A wait of nothing at all never opens.
 */
export function useDwell(
  on: () => string | null,
  delay: () => number,
  clock: Clock = browserClock,
): Ref<number> {
  const open = ref(0)
  let waiting: ReturnType<typeof setTimeout> | undefined
  let frame: number | null = null

  const stopWaiting = () => {
    clearTimeout(waiting)
    waiting = undefined
  }

  const stopMoving = () => {
    if (frame !== null) clock.cancel(frame)
    frame = null
  }

  /** The rest of the way, from wherever it has got to, at the speed it opens. */
  const move = (to: number) => {
    stopMoving()
    const from = open.value
    if (from === to) return

    const span = OPENING * Math.abs(to - from)
    let started: number | null = null

    // Elapsed time comes from the callback, as it does for a move of the whole
    // picture.
    const step = (now: number) => {
      started ??= now
      const t = span <= 0 ? 1 : Math.min(1, (now - started) / span)
      open.value = lerp(from, to, easeOut(t))
      frame = t < 1 ? clock.schedule(step) : null
    }

    frame = clock.schedule(step)
  }

  watch(
    [on, delay],
    ([what, ms]) => {
      stopWaiting()
      move(0)
      if (what === null || ms <= 0) return
      waiting = setTimeout(() => move(1), ms)
    },
    { immediate: true },
  )

  onScopeDispose(() => {
    stopWaiting()
    stopMoving()
  })

  return open
}

/**
 * One node's box as it is drawn, and where its handle sits in it. What the
 * handle is made of is all sizes, and so all tokens.
 *
 * `on` names what the attention is on, and nothing when it is on none.
 */
export function useOpenBox(
  node: () => PlacedNode,
  wide: () => WideBox | null,
  on: () => string | null,
  delay: () => number,
  clock: Clock = browserClock,
) {
  const open = useDwell(on, delay, clock)
  const box = computed(() => boxOf(node(), wide(), open.value))

  const handle = computed(() => {
    const at = handleIn({ ...node(), width: box.value.width })
    return { x: at.x + box.value.offset, y: at.y }
  })

  return { open, box, handle }
}
