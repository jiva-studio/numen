/**
 * Something carried from one place in an order to another, by the pointer or
 * by the keyboard.
 *
 * Everything that knows about events is here; where a landing is allowed and
 * what a landing comes to are the caller's, and each is stated once.
 */
import { shallowRef, type ShallowRef } from 'vue'
import { stepped, type Landing, type Way } from './order'

/**
 * What following a carry takes: the order it runs along, and the rules.
 *
 * `At` is where letting go may put the thing carried. Where a place in the
 * order takes nothing — a row that is pinned — it is `Landing | undefined`, and
 * `nowhere` is the landing that stands for that.
 */
export interface Carry<At extends Landing | undefined> {
  /** The order, as the thing carried is stepped along it. */
  readonly order: () => readonly string[]
  /** Where a landing stands while the pointer is over nothing that takes one. */
  readonly nowhere: At
  /** Whether letting the thing carried go there moves it. */
  readonly lands: (carried: string, at: Landing) => boolean
  /** What a landing that is allowed comes to. */
  readonly moves: (carried: string, at: Landing) => void
}

/** What a carry answers: what is being carried, where it would land, and the gestures. */
export interface Carrying<At extends Landing | undefined> {
  /** What is under the pointer's hand, and nothing while nothing is carried. */
  readonly carried: ShallowRef<string | null>
  /** Where letting go would put it. */
  readonly at: ShallowRef<At>
  /** It was taken up. */
  readonly lift: (what: string, press: DragEvent) => void
  /** It is over a place that would take it. */
  readonly over: (at: At, press: DragEvent) => void
  /** The carry is over, and nothing was let go. */
  readonly release: () => void
  /** It was let go where it stands. */
  readonly drop: () => void
  /** It was asked to go one place along the order. */
  readonly step: (what: string, way: Way, press: KeyboardEvent) => void
}

export function useCarry<At extends Landing | undefined>(carry: Carry<At>): Carrying<At> {
  const carried = shallowRef<string | null>(null)
  const at: ShallowRef<At> = shallowRef(carry.nowhere)

  const lift = (what: string, press: DragEvent): void => {
    carried.value = what
    press.dataTransfer?.setData('text/plain', what)
  }

  const over = (lands: At, press: DragEvent): void => {
    if (carried.value === null) return
    press.preventDefault()
    at.value = lands
  }

  const release = (): void => {
    carried.value = null
    at.value = carry.nowhere
  }

  const drop = (): void => {
    const held = carried.value
    const lands = at.value
    release()
    if (held === null || lands === undefined) return
    if (carry.lands(held, lands)) carry.moves(held, lands)
  }

  const step = (what: string, way: Way, press: KeyboardEvent): void => {
    const lands = stepped(carry.order(), what, way)
    if (lands === undefined || !carry.lands(what, lands)) return
    press.preventDefault()
    carry.moves(what, lands)
  }

  return { carried, at, lift, over, release, drop, step }
}
