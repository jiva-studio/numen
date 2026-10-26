/**
 * The hand on a book: a swipe across the reading area turns the page, and so
 * does a press near either edge. Words taken up in the text are being read, and
 * letting go of them turns nothing.
 */
import type { ShallowRef } from 'vue'

import type { PageNavigationAction } from '@numen/ui'
import { dragPage } from '../lib/pageStep'

export interface BookHandOptions {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly turn: (way: PageNavigationAction) => void
  readonly getEdge: (of: HTMLElement) => number
}

/**
 * `turn` is what the reader does with the way the hand asked for, and `getEdge`
 * is the near edge of the reading area, which a press is measured from.
 */
export function createBookHand(options: BookHandOptions) {
  /** Where the hand went down, while it is down. */
  let hand: number | undefined

  /** Whether words of the text stand taken up. */
  const isSelecting = (): boolean => {
    const currentSelection = window.getSelection()
    if (
      !currentSelection ||
      currentSelection.isCollapsed ||
      currentSelection.toString().trim() === ''
    )
      return false
    const text = options.paper.value
    return !!text && !!currentSelection.anchorNode && text.contains(currentSelection.anchorNode)
  }

  const takeDown = (event: PointerEvent) => {
    hand = event.button === 0 ? event.clientX : undefined
  }

  /** The hand lifted. A link is followed and turns nothing. */
  const letGo = (event: PointerEvent) => {
    const from = hand
    hand = undefined
    const box = options.area.value
    if (from === undefined || !box) return
    if ((event.target as HTMLElement | null)?.closest?.('a')) return

    const edge = options.getEdge(box)
    const way = dragPage(from - edge, event.clientX - edge, {
      wide: box.clientWidth,
      hasSelection: isSelecting(),
    })
    if (way) options.turn(way)
  }

  return { takeDown, letGo }
}
