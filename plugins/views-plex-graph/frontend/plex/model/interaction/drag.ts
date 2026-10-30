/**
 * An item dragged over the plex from outside it. Everything that handles events
 * and screen pixels is here; the relation a position evaluates to is computed
 * in `arrange/drop.ts`, as a value.
 */
import { win } from '@numen/ui'
import { computed, onScopeDispose, ref, watch, type Ref } from 'vue'
import { getDropRelation, type PlexOptions, type Size } from '../../lib/arrange'
import { getPosition } from './gesture'
import type { PlexFrame } from '../../lib/frame'
import type { Position } from '../../lib/node'
import type { PlexRelation } from '../../lib/relation'

const CAPTURE = { capture: true } as const

export interface PlexDragState {
  /** Where the pointer is, in the plex's own coordinates. */
  readonly at: Ref<Position | null>
  /** The relation letting go here evaluates to, so it can be shown before it does. */
  readonly relation: Ref<PlexRelation | null>
}

/** The dependencies of a drag: the drawing surface, the layout, and the rules. */
export interface PlexDragDeps {
  /** The drawing, which turns screen pixels into the plex's own coordinates. */
  readonly getSurface: () => SVGSVGElement | null
  /** What is being dragged, each of them opaque. Empty while nothing is. */
  readonly getDragged: () => readonly string[]
  readonly getFrame: () => PlexFrame
  readonly getOptions: () => PlexOptions
  readonly getViewport: () => Size
  /** Relations a gesture is allowed to produce. */
  readonly getAllowedRelations: () => readonly PlexRelation[]
  /** The distance from the focus at which the pointer selects a direction. */
  readonly getThreshold: () => number
  readonly commit: (dragged: readonly string[], relation: PlexRelation) => void
}

/**
 * Follow a pointer dragging something across the plex until it is let go or
 * given up on.
 *
 * The gesture began somewhere the plex cannot see, so it is followed on the
 * window for as long as there is something to drag, and it is over the
 * instant the pointer comes up wherever that is.
 */
export function usePlexDrag(drag: PlexDragDeps): PlexDragState {
  /** Where the pointer is, and nothing at all while nothing is being dragged. */
  const pos = ref<Position | null>(null)

  const relation = computed<PlexRelation | null>(() => {
    const now = pos.value
    if (!now) return null
    return getDropRelation({
      frame: drag.getFrame(),
      options: drag.getOptions(),
      viewport: drag.getViewport(),
      point: now,
      relations: drag.getAllowedRelations(),
      threshold: drag.getThreshold(),
    })
  })

  /** What the drag under way installed on the window, if anything. */
  let detach: (() => void) | null = null

  /**
   * The identifiers being dragged, kept for the duration of the gesture. The
   * source of the drag may drop them on the same release this commits on.
   */
  let activeDragIds: readonly string[]

  const stop = () => {
    detach?.()
    detach = null
    pos.value = null
  }

  const move = (event: PointerEvent) => {
    const element = drag.getSurface()
    pos.value = element ? getPosition(element, event) : null
  }

  const finish = (event: PointerEvent) => {
    move(event)
    const dragged = activeDragIds
    const resolved = relation.value
    stop()
    if (resolved) drag.commit(dragged, resolved)
  }

  const follow = () => {
    activeDragIds = drag.getDragged()

    const onMove = (event: PointerEvent) => move(event)
    const onUp = (up: PointerEvent) => finish(up)
    // The browser takes the pointer away, and no `pointerup` follows.
    const onLost = () => stop()
    const onKey = (key: KeyboardEvent) => {
      if (key.key === 'Escape') stop()
    }

    detach = () => {
      win.removeEventListener('pointermove', onMove)
      win.removeEventListener('pointerup', onUp, CAPTURE)
      win.removeEventListener('pointercancel', onLost)
      win.removeEventListener('keydown', onKey)
    }

    win.addEventListener('pointermove', onMove)
    // The release is handled in the capture phase, before the drag source's
    // own handler, which drops the items in the bubble phase.
    win.addEventListener('pointerup', onUp, CAPTURE)
    win.addEventListener('pointercancel', onLost)
    win.addEventListener('keydown', onKey)
  }

  watch(
    () => drag.getDragged().length > 0,
    (isDragging) => (isDragging ? follow() : stop()),
    { immediate: true },
  )

  // A plex can go while something is still being dragged over it.
  onScopeDispose(stop)

  return { at: pos, relation }
}
