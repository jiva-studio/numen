/**
 * The layout of a book drawn in columns: the room it is read in measured, the
 * text set against it, the spread in front carried there, and the runs asked
 * about marked where they stand.
 *
 * Apart from the component the way `lib/turn.ts` is: the component hands over the
 * two elements, what it was told, and the one measurement this makes of the
 * browser, and answers for nothing here.
 */
import { computed, onMounted, ref, watch } from 'vue'
import type { ShallowRef } from 'vue'

import { useViewport } from '@/shared/lib/viewport'
import { onNextFrame } from '@/shared/lib/clock'
import { useBookMarks } from './marks'
import type { BookLink } from '../lib/link'
import { GAP, LARGEST, SMALLEST, columnHeight, clamp, columnWidth, columnsIn, findSpreadAt, getSpreadStart, inFront, leftInDocument, pagesOf, spreads, type Flow } from '../lib/spread'
import { turnTo, type PageTurn } from '../lib/turn'
import type { SettledBookProps } from '../lib/props'

/**
 * The room one document of a book is read in, and everything measured from it.
 * `onMove` is what the reader says when the offset in front changes, and
 * `takeLed` hands over the place a link led to, once.
 */
export function useBookLayout(
  area: Readonly<ShallowRef<HTMLElement | null>>,
  paper: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  onMove: (at: number) => void,
  takeLed: () => BookLink | undefined,
  edgeOf: (of: HTMLElement) => number,
) {
  /** How large the text is set, held inside what a book may be read at. */
  const textSize = computed(() => clamp(props.textSize, SMALLEST, LARGEST))

  /** The reading area, taken again whenever it changes. */
  const { viewport, measure } = useViewport(area)

  /** How far the columns run, taken once they have been laid out. */
  const along = ref(0)

  /** Which spread is in front, counted from the first. */
  const standing = ref(0)

  const { marks, gather, placeAt, markRuns } = useBookMarks(paper, props, edgeOf)

  /**
   * Whether the reading area has been measured. A book is turned and never
   * scrolled, so its text is drawn only against an area of a known size.
   */
  const measured = computed(() => viewport.value.width > 0 && viewport.value.height > 0)

  const columns = computed(() => columnsIn(viewport.value.width, textSize.value))

  const flow = computed<Flow>(() => ({
    along: along.value,
    width: viewport.value.width,
    gap: GAP,
    columns: columns.value,
  }))

  /** How many spreads the document is read in. */
  const spreadCount = computed(() => spreads(flow.value))

  /** The spread in front, as a page of the book, and how many there are. */
  const front = computed(() => pagesOf(props.book, props.span, flow.value, standing.value, marks.value))

  /** How much of the chapter in front is still to come, which is measured exactly. */
  const leftInChapter = computed(() => leftInDocument(flow.value, standing.value, marks.value))

  /**
   * What the columns are set with. A column's own width and height are among
   * them: a percentage inside a column resolves against a box whose height is not
   * settled.
   */
  const setting = computed(() => ({
    '--book-run': columns.value === 1 ? '200%' : '100%',
    '--book-gap': `${GAP}px`,
    '--book-column': `${columnWidth(flow.value)}px`,
    '--book-height': `${viewport.value.height}px`,
    '--book-size': `calc(var(--numen-prose-size) * ${textSize.value})`,
  }))

  /**
   * How tall one line of the text is set, taken off a run of the text itself. The
   * lines the column has to end between are the ones the prose is set in, and the
   * box holding it is set in another. A line height the browser will put no
   * number to leaves the column at the height of the area.
   */
  const lineOf = (text: HTMLElement): number => {
    const run = text.querySelector<HTMLElement>('p') ?? text
    const said = Number.parseFloat(getComputedStyle(run).lineHeight)
    return Number.isFinite(said) ? said : 0
  }

  /**
   * The spread put against the near edge of the reading area. The columns are
   * carried there rather than scrolled to: a scroll stops at the end of what it
   * has to scroll, and the gap the last column keeps beside it is not part of
   * that, so the last spread of a document would stand half a gap short.
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

  /**
   * The text set in columns again, with one offset kept in front. The columns are
   * measured after the browser has laid them out, and only an area that overflows
   * has a length to measure.
   *
   * A place a link led to is found here, in markup that has only now been drawn.
   */
  const settle = (keep: number, led?: BookLink) => {
    measure()
    if (!measured.value) return
    onNextFrame(() => {
      const box = area.value
      const text = paper.value
      if (!box || !text) return
      text.style.setProperty('--book-paper', `${columnHeight(box.clientHeight, lineOf(text))}px`)
      // How far the columns run is asked of the box they are set in. The area
      // around it clips what overflows, and a box that clips is not asked how far
      // what it clipped reaches.
      along.value = text.scrollWidth
      gather()
      const landed =
        led && (led.path === '' || led.path === props.path) ? placeAt(led.fragment) : undefined
      stand(findSpreadAt(marks.value, flow.value, landed ?? keep), 'auto')
      if (landed !== undefined) onMove(landed)
      markRuns()
    })
  }

  /** What a turn of the page asks for, and where it lands. */
  const turn = (way: PageTurn) => {
    const to = turnTo(way, standing.value, spreadCount.value, props.span, props.book)
    if (to.spread !== undefined) return goTo(to.spread)
    if (to.offset !== undefined) onMove(to.offset)
  }

  // A reading area of another size, or a text of another size, is another set of
  // columns.
  watch([() => viewport.value.width, () => viewport.value.height, textSize], () => {
    settle(getKeptOffset())
  })

  // Another document is opened at the offset asked for, and there is nothing to
  // keep in front.
  watch(
    () => props.markup,
    () => {
      const place = takeLed()
      standing.value = 0
      settle(props.at, place)
    },
  )

  // An offset asked for from outside is turned to. One reached by the hand is
  // already in front.
  watch(
    () => props.at,
    (at) => {
      if (at < props.span.begins || at >= props.span.ends) return
      const want = findSpreadAt(marks.value, flow.value, at)
      if (want !== standing.value) stand(want, 'smooth')
    },
  )

  onMounted(() => {
    settle(props.at)
    // The columns are counted over the type the book is set in, which arrives
    // after the markup does.
    void document.fonts?.ready.then(() => settle(getKeptOffset()))
  })

  return {
    textSize,
    measured,
    flow,
    spreadCount,
    front,
    leftInChapter,
    setting,
    getKeptOffset,
    settle,
    placeAt,
    turn,
  }
}
