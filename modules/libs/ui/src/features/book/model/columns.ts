/**
 * The columns one document of a book is set in: the room measured, the type
 * sized against it, and the runs of the text marked where they came to stand.
 *
 * Apart from the component the way `lib/spread.ts` is: it is handed the two
 * elements and the one measurement of the browser it may not make itself.
 */
import { computed, ref } from 'vue'
import type { ShallowRef } from 'vue'

import { clamp } from '@/shared/lib/bounds'
import { useViewport } from '@/shared/lib/viewport'
import { useBookMarks } from './marks'
import {
  GAP,
  LARGEST,
  SMALLEST,
  columnHeight,
  columnWidth,
  countColumns,
  countSpreads,
  type Flow,
} from '../lib/spread'
import type { SettledBookProps } from '../lib/props'

/**
 * How tall one line of the text is set, taken off a run of the text itself. The
 * lines the column has to end between are the ones the prose is set in, and the
 * box holding it is set in another. A line height the browser will put no
 * number to leaves the column at the height of the area.
 */
const getLine = (text: HTMLElement): number => {
  const run = text.querySelector<HTMLElement>('p') ?? text
  const parsedHeight = Number.parseFloat(getComputedStyle(run).lineHeight)
  return Number.isFinite(parsedHeight) ? parsedHeight : 0
}

function createBookSetting(flow: Flow, height: number, textSize: number) {
  return {
    '--book-run': flow.columns === 1 ? '200%' : '100%',
    '--book-gap': `${GAP}px`,
    '--book-column': `${columnWidth(flow)}px`,
    '--book-height': `${height}px`,
    '--book-size': `calc(var(--numen-prose-size) * ${textSize})`,
  }
}

export interface BookColumnsOptions {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly getEdge: (of: HTMLElement) => number
}

/** `getEdge` is handed on to the marks, which is what reads it. */
export function useBookColumns(options: BookColumnsOptions) {
  /** How large the text is set, held inside what a book may be read at. */
  const textSize = computed(() => clamp(options.props.textSize, { min: SMALLEST, max: LARGEST }))

  /** The reading area, taken again whenever it changes. */
  const { viewport, measure } = useViewport(options.area)

  /** How far the columns run, taken once they have been laid out. */
  const alongOffset = ref(0)

  const { marks, gather, findFragmentOffset, markRuns } = useBookMarks(
    options.paper,
    options.props,
    options.getEdge,
  )

  /**
   * Whether the reading area has been measured. A book is turned and never
   * scrolled, so its text is drawn only against an area of a known size.
   */
  const isMeasured = computed(() => viewport.value.width > 0 && viewport.value.height > 0)

  /** Whether the spread in front of the drawn document has been stood. */
  const isStood = ref(false)

  const columns = computed(() => countColumns(viewport.value.width, textSize.value))

  const flow = computed<Flow>(() => ({
    along: alongOffset.value,
    width: viewport.value.width,
    gap: GAP,
    columns: columns.value,
  }))

  /** How many spreads the document is read in. */
  const spreadCount = computed(() => countSpreads(flow.value))

  const setting = computed(() =>
    createBookSetting(flow.value, viewport.value.height, textSize.value),
  )

  /**
   * The columns as the browser has just laid them out: how tall a column may be,
   * how far they run, and where each run of the text stands.
   *
   * An area that clips its own overflow is not asked how far what it clipped
   * reaches, so the length is asked of the box the columns are set in.
   */
  const readColumns = (box: HTMLElement, text: HTMLElement): void => {
    text.style.setProperty('--book-paper', `${columnHeight(box.clientHeight, getLine(text))}px`)
    alongOffset.value = text.scrollWidth
    gather()
  }

  return {
    textSize,
    viewport,
    isMeasured,
    isStood,
    flow,
    spreadCount,
    setting,
    marks,
    measure,
    readColumns,
    findFragmentOffset,
    markRuns,
  }
}
