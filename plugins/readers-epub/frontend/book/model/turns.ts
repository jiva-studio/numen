/**
 * Reading position in a book, and the result of turning a page.
 *
 * A spread is shown by translating the columns to it, so this reads the columns
 * and the marks measured from them and changes nothing else.
 */
import { computed, ref } from 'vue'
import type { ComputedRef, ShallowRef } from 'vue'

import {
  findSpreadAt,
  getSpreadStart,
  inFront,
  leftInDocument,
  createPages,
  type Flow,
  type Mark,
} from '../lib/spread'
import type { PageNavigationAction } from '@numen/ui'
import { goToPage } from '../lib/pageStep'
import type { SettledBookProps } from '../lib/props'

function translatePaper(text: HTMLElement, to: string, how: 'smooth' | 'auto'): void {
  if (how === 'smooth') {
    text.style.translate = to
    return
  }
  text.style.transition = 'none'
  text.style.translate = to
  void text.offsetWidth
  text.style.transition = ''
}

export interface BookTurnsOptions {
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly flow: ComputedRef<Flow>
  readonly spreadCount: ComputedRef<number>
  readonly marks: Readonly<ShallowRef<readonly Mark[]>>
  readonly onMove: (pos: number) => void
}

/** `onMove` is called when the current offset changes. */
export function useBookTurns(options: BookTurnsOptions) {
  /** Which spread is in front, counted from the first. */
  const activeSpread = ref(0)

  /** The spread in front, as a page of the book, and how many there are. */
  const front = computed(() =>
    createPages({
      book: options.props.book,
      document: options.props.span,
      flow: options.flow.value,
      spread: activeSpread.value,
      marks: options.marks.value,
    }),
  )

  /** How much of the chapter in front is still to come, which is measured exactly. */
  const leftInChapter = computed(() =>
    leftInDocument(options.flow.value, activeSpread.value, options.marks.value),
  )

  /**
   * The spread positioned at the near edge of the reading area. The columns are
   * translated there, gap included, so the last spread of a document is
   * positioned like every other one.
   */
  const stand = (spread: number, how: 'smooth' | 'auto') => {
    activeSpread.value = Math.min(Math.max(spread, 0), Math.max(options.spreadCount.value - 1, 0))
    if (!options.paper.value) return
    const to = `${-getSpreadStart(options.flow.value, activeSpread.value)}px 0`
    translatePaper(options.paper.value, to, how)
  }

  const announce = () => {
    const now = inFront(options.marks.value, options.flow.value, activeSpread.value)
    if (now !== undefined && now !== options.props.at) options.onMove(now)
  }

  const goTo = (spread: number) => {
    stand(spread, 'smooth')
    announce()
  }

  /** The offset currently being read, preserved while the text is laid out again. */
  const getKeptOffset = () =>
    inFront(options.marks.value, options.flow.value, activeSpread.value) ?? options.props.at

  /** The spread one offset of the book's text falls on. */
  const getSpreadOf = (pos: number) => findSpreadAt(options.marks.value, options.flow.value, pos)

  /** The target of a page turn, and where it lands. */
  const turn = (way: PageNavigationAction) => {
    const to = goToPage(way, {
      spread: activeSpread.value,
      count: options.spreadCount.value,
      span: options.props.span,
      book: options.props.book,
    })
    if (to.spread !== undefined) return goTo(to.spread)
    if (to.offset !== undefined) options.onMove(to.offset)
  }

  return {
    standing: activeSpread,
    front,
    leftInChapter,
    stand,
    goTo,
    getKeptOffset,
    getSpreadOf,
    turn,
  }
}
