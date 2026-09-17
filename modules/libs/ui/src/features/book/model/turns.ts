/**
 * Where in a book the reader stands, and what turning a page comes to.
 *
 * A spread is put in front by carrying the columns to it, so this reads the
 * columns and the marks the columns were measured into, and moves nothing else.
 */
import { computed, ref } from 'vue'
import type { ComputedRef, ShallowRef } from 'vue'

import {
  findSpreadAt,
  getSpreadStart,
  inFront,
  leftInDocument,
  pagesOf,
  type Flow,
  type Mark,
} from '../lib/spread'
import type { PageTurn } from '@/shared/lib/turn'
import { turnTo } from '../lib/turn'
import type { SettledBookProps } from '../lib/props'

/** `onMove` is what the reader says when the offset in front changes. */
export function useBookTurns(
  paper: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  flow: ComputedRef<Flow>,
  spreadCount: ComputedRef<number>,
  marks: Readonly<ShallowRef<readonly Mark[]>>,
  onMove: (at: number) => void,
) {
  /** Which spread is in front, counted from the first. */
  const standing = ref(0)

  /** The spread in front, as a page of the book, and how many there are. */
  const front = computed(() =>
    pagesOf(props.book, props.span, flow.value, standing.value, marks.value),
  )

  /** How much of the chapter in front is still to come, which is measured exactly. */
  const leftInChapter = computed(() => leftInDocument(flow.value, standing.value, marks.value))

  /**
   * The spread put against the near edge of the reading area. The columns are
   * carried there, gap and all, so the last spread of a document stands where
   * every other one does.
   */
  const stand = (spread: number, how: 'smooth' | 'auto') => {
    const text = paper.value
    standing.value = Math.min(Math.max(spread, 0), Math.max(spreadCount.value - 1, 0))
    if (!text) return

    const to = `${-getSpreadStart(flow.value, standing.value)}px 0`
    if (how === 'smooth') {
      text.style.translate = to
      return
    }
    // A layout is not a turn: the columns are put where they belong at once. The
    // style is settled in between, because a browser handed the transition and
    // the distance in one recalculation animates neither.
    text.style.transition = 'none'
    text.style.translate = to
    void text.offsetWidth
    text.style.transition = ''
  }

  /** What stands in front now, said once, and nothing while nothing does. */
  const announce = () => {
    const now = inFront(marks.value, flow.value, standing.value)
    if (now !== undefined && now !== props.at) onMove(now)
  }

  const goTo = (spread: number) => {
    stand(spread, 'smooth')
    announce()
  }

  /** What the person is reading now, to be kept in front while the text is set again. */
  const getKeptOffset = () => inFront(marks.value, flow.value, standing.value) ?? props.at

  /** The spread one offset of the book's text falls on. */
  const getSpreadOf = (at: number) => findSpreadAt(marks.value, flow.value, at)

  /** What a turn of the page asks for, and where it lands. */
  const turn = (way: PageTurn) => {
    const to = turnTo(way, standing.value, spreadCount.value, props.span, props.book)
    if (to.spread !== undefined) return goTo(to.spread)
    if (to.offset !== undefined) onMove(to.offset)
  }

  return { standing, front, leftInChapter, stand, goTo, getKeptOffset, getSpreadOf, turn }
}
