/**
 * Something dragged from one place in an order to another, by the pointer or
 * by the keyboard.
 *
 * Everything that knows about events is here; where a landing is allowed and
 * what a landing comes to are the caller's, and each is stated once.
 */
import { shallowRef, type ShallowRef } from 'vue'
import { stepped, type InsertionPoint, type StepDirection } from './order'

/**
 * What following a drag takes: the order it runs along, and the rules.
 *
 * `At` is where letting go may put the thing dragged. Where a place in the
 * order takes nothing — a row that is pinned — it is
 * `InsertionPoint | undefined`, and `nowhere` is the landing that stands for
 * that.
 */
export interface DragDeps<At extends InsertionPoint | undefined> {
  /** The order, as the thing dragged is stepped along it. */
  readonly order: () => readonly string[]
  /** Where a landing stands while the pointer is over nothing that takes one. */
  readonly nowhere: At
  /** Whether letting the thing dragged go there moves it. */
  readonly lands: (dragged: string, at: InsertionPoint) => boolean
  /** What a landing that is allowed comes to. */
  readonly moves: (dragged: string, at: InsertionPoint) => void
}

/** What a drag answers: what is being dragged, where it would land, and the gestures. */
export interface DragState<At extends InsertionPoint | undefined> {
  /** What is under the pointer's hand, and nothing while nothing is dragged. */
  readonly dragged: ShallowRef<string | null>
  /** Where letting go would put it. */
  readonly at: ShallowRef<At>
  /** It was taken up. */
  readonly lift: (what: string, press: DragEvent) => void
  /** It is over a place that would take it. */
  readonly over: (at: At, press: DragEvent) => void
  /** The drag is over, and nothing was let go. */
  readonly release: () => void
  /** It was let go where it stands. */
  readonly drop: () => void
  /** It was asked to go one place along the order. */
  readonly step: (what: string, direction: StepDirection, press: KeyboardEvent) => void
}

export function useDrag<At extends InsertionPoint | undefined>(
  drag: DragDeps<At>,
): DragState<At> {
  const dragged = shallowRef<string | null>(null)
  const at: ShallowRef<At> = shallowRef(drag.nowhere)

  const lift = (what: string, press: DragEvent): void => {
    dragged.value = what
    press.dataTransfer?.setData('text/plain', what)
  }

  const over = (lands: At, press: DragEvent): void => {
    if (dragged.value === null) return
    press.preventDefault()
    at.value = lands
  }

  const release = (): void => {
    dragged.value = null
    at.value = drag.nowhere
  }

  const drop = (): void => {
    const held = dragged.value
    const lands = at.value
    release()
    if (held === null || lands === undefined) return
    if (drag.lands(held, lands)) drag.moves(held, lands)
  }

  const step = (what: string, direction: StepDirection, press: KeyboardEvent): void => {
    const lands = stepped(drag.order(), what, direction)
    if (lands === undefined || !drag.lands(what, lands)) return
    press.preventDefault()
    drag.moves(what, lands)
  }

  return { dragged, at, lift, over, release, drop, step }
}
