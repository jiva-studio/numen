/**
 * The columns one document of a book is set in: the room measured, the type
 * sized against it, and the runs of the text marked where they came to stand.
 *
 * Apart from the component the way `lib/spread.ts` is: it is handed the two
 * elements and the one measurement of the browser it may not make itself.
 */
import { computed, ref } from 'vue'
import type { ShallowRef } from 'vue'

import { clamp } from '@numen/ui'
import { useViewport } from '@numen/ui'
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

export interface BookFlowState {
  readonly textSize: import('vue').ComputedRef<number>
  readonly viewport: ShallowRef<{ width: number; height: number }>
  readonly isMeasured: import('vue').ComputedRef<boolean>
  readonly flow: import('vue').ComputedRef<Flow>
  readonly spreadCount: import('vue').ComputedRef<number>
  readonly setting: import('vue').ComputedRef<Record<string, string>>
  readonly measure: () => void
}

/** The flow and sizing of the book reading area. */
export function useBookFlow(
  area: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  contentWidth: ShallowRef<number> | import('vue').Ref<number>,
): BookFlowState {
  const textSize = computed(() => clamp(props.textSize, { min: SMALLEST, max: LARGEST }))
  const { viewport, measure } = useViewport(area)
  const isMeasured = computed(() => viewport.value.width > 0 && viewport.value.height > 0)
  const columns = computed(() => countColumns(viewport.value.width, textSize.value))
  const flow = computed<Flow>(() => ({
    contentWidth: contentWidth.value,
    width: viewport.value.width,
    gap: GAP,
    columns: columns.value,
  }))
  const spreadCount = computed(() => countSpreads(flow.value))
  const setting = computed(() =>
    createBookSetting(flow.value, viewport.value.height, textSize.value),
  )

  return {
    textSize,
    viewport,
    isMeasured,
    flow,
    spreadCount,
    setting,
    measure,
  }
}

/** `getEdge` is handed on to the marks, which is what reads it. */
export function useBookColumns(options: BookColumnsOptions) {
  const contentWidth = ref(0)
  const { textSize, viewport, isMeasured, flow, spreadCount, setting, measure } = useBookFlow(
    options.area,
    options.props,
    contentWidth,
  )

  const { marks, gather, findFragmentOffset, markRuns } = useBookMarks(
    options.paper,
    options.props,
    options.getEdge,
  )

  /** Whether the spread in front of the drawn document has been stood. */
  const isStood = ref(false)

  /**
   * The columns as the browser has just laid them out: how tall a column may be,
   * how far they run, and where each run of the text stands.
   *
   * An area that clips its own overflow is not asked how far what it clipped
   * reaches, so the length is asked of the box the columns are set in.
   */
  const readColumns = (box: HTMLElement, text: HTMLElement): void => {
    text.style.setProperty('--book-paper', `${columnHeight(box.clientHeight, getLine(text))}px`)
    contentWidth.value = text.scrollWidth
    gather()
  }

  const state = {
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
  return state
}
