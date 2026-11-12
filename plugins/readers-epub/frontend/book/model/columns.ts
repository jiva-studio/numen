/**
 * The columns one document of a book is set in: the measured width, the font size
 * derived from it, and the text ranges marked where they are laid out.
 *
 * Separate from the component, like `lib/spread.ts`: it receives the two
 * elements and the one browser measurement it may not make itself.
 */
import { computed, ref, shallowRef } from 'vue'
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
  countFilledColumns,
  countSpreads,
  type Flow,
} from '../lib/spread'
import type { SettledBookProps } from '../lib/props'

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
  readonly viewportFlow: import('vue').ComputedRef<Flow>
  readonly setting: import('vue').ComputedRef<Record<string, string>>
  readonly measure: () => void
}

function useBookFlow(
  area: Readonly<ShallowRef<HTMLElement | null>>,
  props: SettledBookProps,
  contentWidth: ShallowRef<number> | import('vue').Ref<number>,
): BookFlowState {
  const textSize = computed(() => clamp(props.textSize, { min: SMALLEST, max: LARGEST }))
  const { viewport, measure } = useViewport(area)
  const isMeasured = computed(() => viewport.value.width > 0 && viewport.value.height > 0)
  const columns = computed(() => countColumns(viewport.value.width, textSize.value))
  const viewportFlow = computed<Flow>(() => ({
    contentWidth: contentWidth.value,
    width: viewport.value.width,
    gap: GAP,
    columns: columns.value,
  }))
  const setting = computed(() =>
    createBookSetting(viewportFlow.value, viewport.value.height, textSize.value),
  )

  return {
    textSize,
    viewport,
    isMeasured,
    viewportFlow,
    setting,
    measure,
  }
}

/** `getEdge` is handed on to the marks, which is what reads it. */
export function useBookColumns(options: BookColumnsOptions) {
  const contentWidth = ref(0)
  const sizing = useBookFlow(options.area, options.props, contentWidth)
  const { viewportFlow } = sizing

  /**
   * The flow the marks were read in. The area takes its new size before the columns are read
   * again, and until then the marks and the spread in front are positions in this flow.
   */
  const laidFlow = shallowRef<Flow>(viewportFlow.value)
  const spreadCount = computed(() => countSpreads(laidFlow.value))

  const runMarks = useBookMarks(options.paper, options.props, options.getEdge)

  /** Whether the spread for the rendered document has been laid out. */
  const isStood = ref(false)

  /**
   * The columns as the browser has laid them out: the maximum column height,
   * the total length, and where each text range is.
   *
   * An area that clips its own overflow cannot report the extent of the clipped
   * content, so the length is read from the box the columns are set in.
   *
   * The columns are as wide as the box is at this moment, and the viewport learns that width only
   * after the frame, so the width the marks are read against is taken off the box. The column
   * count is the one the CSS was last set at.
   *
   * Returns the column height in pixels, which is also written to `--book-paper` on the text.
   */
  const readColumns = (box: HTMLElement, text: HTMLElement): number => {
    const paperHeight = columnHeight(box.clientHeight, getLine(text))
    text.style.setProperty('--book-paper', `${paperHeight}px`)
    runMarks.gather()
    const readFlow: Flow = { ...viewportFlow.value, width: box.clientWidth }
    const isSingleColumn =
      readFlow.columns === 1 &&
      text.clientWidth > 0 &&
      text.scrollWidth <= text.clientWidth &&
      countFilledColumns(runMarks.marks.value, readFlow) === 1
    contentWidth.value = isSingleColumn ? box.clientWidth : text.scrollWidth
    laidFlow.value = { ...readFlow, contentWidth: contentWidth.value }
    return paperHeight
  }

  return {
    sizing,
    runMarks,
    laid: { flow: laidFlow, spreadCount },
    isStood,
    readColumns,
  }
}
