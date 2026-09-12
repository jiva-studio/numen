/**
 * The hand on a book: a swipe across the reading area turns the page, and so
 * does a press near either edge. Words taken up in the text are being read, and
 * letting go of them turns nothing.
 */
import type { ShallowRef } from 'vue'

import { handTurn, type PageTurn } from '../lib/turn'

/**
 * `turn` is what the reader does with the way the hand asked for, and `edgeOf`
 * is the near edge of the reading area, which a press is measured from.
 */
export function useBookHand(
  area: Readonly<ShallowRef<HTMLElement | null>>,
  paper: Readonly<ShallowRef<HTMLElement | null>>,
  turn: (way: PageTurn) => void,
  edgeOf: (of: HTMLElement) => number,
) {
  /** Where the hand went down, while it is down. */
  let hand: number | undefined

  /** Whether words of the text stand taken up. */
  const isSelecting = (): boolean => {
    const taken = window.getSelection()
    if (!taken || taken.isCollapsed || taken.toString().trim() === '') return false
    const text = paper.value
    return !!text && !!taken.anchorNode && text.contains(taken.anchorNode)
  }

  const takeDown = (event: PointerEvent) => {
    hand = event.button === 0 ? event.clientX : undefined
  }

  /** The hand lifted. A link is followed and turns nothing. */
  const letGo = (event: PointerEvent) => {
    const from = hand
    hand = undefined
    const box = area.value
    if (from === undefined || !box) return
    if ((event.target as HTMLElement | null)?.closest?.('a')) return

    const edge = edgeOf(box)
    const way = handTurn(from - edge, event.clientX - edge, box.clientWidth, isSelecting())
    if (way) turn(way)
  }

  return { takeDown, letGo }
}
