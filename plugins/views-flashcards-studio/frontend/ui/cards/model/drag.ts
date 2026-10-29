/**
 * An item dragged from one position in an order to another, by the pointer or by
 * the keyboard.
 *
 * All event handling is here; where a drop is allowed and what a drop does are
 * supplied by the caller, and each is defined once.
 */
import { shallowRef, type ShallowRef } from 'vue'
import { getStepInsertionPoint, type InsertionPoint, type StepDirection } from '../lib/order'

/**
 * What a drag requires: the order it runs along, and the rules.
 *
 * `At` is where releasing may place the dragged item. Where a position in the
 * order accepts nothing, such as a pinned row, `At` is
 * `InsertionPoint | undefined`, and `nowhere` is the drop target that
 * represents it.
 */
export interface DragDeps<At extends InsertionPoint | undefined> {
  /** The order, as the thing dragged is stepped along it. */
  readonly order: () => readonly string[]
  /** The drop target used while the pointer is over nothing that accepts a drop. */
  readonly nowhere: At
  /** Whether letting the thing dragged go there moves it. */
  readonly isMoved: (dragged: string, target: InsertionPoint) => boolean
  /** What a drop that is allowed evaluates to. */
  readonly move: (dragged: string, target: InsertionPoint) => void
}

/** What a drag answers: what is being dragged, where it would land, and the gestures. */
export interface DragState<At extends InsertionPoint | undefined> {
  /** The item under the pointer, and null while nothing is dragged. */
  readonly dragged: ShallowRef<string | null>
  /** Where letting go would put it. */
  readonly at: ShallowRef<At>
  /** It was taken up. */
  readonly lift: (what: string, press: DragEvent) => void
  /** It is over a place that would take it. */
  readonly hover: (target: At, press: DragEvent) => void
  /** The drag is over, and nothing was let go. */
  readonly release: () => void
  /** It was let go at its current position. */
  readonly drop: () => void
  /** The item is moved one position along the order. */
  readonly step: (what: string, direction: StepDirection, press: KeyboardEvent) => void
}

export function useDrag<At extends InsertionPoint | undefined>(drag: DragDeps<At>): DragState<At> {
  const dragged = shallowRef<string | null>(null)
  const dropTarget: ShallowRef<At> = shallowRef(drag.nowhere)

  const lift = (what: string, press: DragEvent): void => {
    dragged.value = what
    press.dataTransfer?.setData('text/plain', what)
  }

  const hover = (lands: At, press: DragEvent): void => {
    if (dragged.value === null) return
    press.preventDefault()
    dropTarget.value = lands
  }

  const release = (): void => {
    dragged.value = null
    dropTarget.value = drag.nowhere
  }

  const drop = (): void => {
    const target = dragged.value
    const lands = dropTarget.value
    release()
    if (target === null || lands === undefined) return
    if (drag.isMoved(target, lands)) drag.move(target, lands)
  }

  const step = (what: string, direction: StepDirection, press: KeyboardEvent): void => {
    const lands = getStepInsertionPoint(drag.order(), what, direction)
    if (lands === undefined || !drag.isMoved(what, lands)) return
    press.preventDefault()
    drag.move(what, lands)
  }

  return { dragged, at: dropTarget, lift, hover, release, drop, step }
}
