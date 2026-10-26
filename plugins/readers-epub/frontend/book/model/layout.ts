/**
 * The layout of a book drawn in columns: the columns set against the room they
 * are read in, and the spread in front carried to the edge of it.
 *
 * What the columns are is `columns.ts` and where the reader stands is
 * `turns.ts`. What is here is the settling that joins them — the text set
 * again, with one offset kept in front — and what asks for it.
 */
import { onMounted, watch } from 'vue'
import type { ShallowRef } from 'vue'

import { onNextFrame } from '@numen/ui'
import { useBookColumns } from './columns'
import { useBookTurns } from './turns'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/**
 * The room one document of a book is read in, and everything measured from it.
 * `onMove` is what the reader says when the offset in front changes, and
 * `takeLed` hands over the place a link led to, once.
 */
export interface BookLayoutOptions {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly onMove: (pos: number) => void
  readonly takeLed: () => BookLink | undefined
  readonly getEdge: (of: HTMLElement) => number
}

interface BookWatchContext {
  readonly settle: (keep: number, led?: BookLink) => void
  readonly getKeptOffset: () => number
  readonly takeLed: () => BookLink | undefined
  readonly standing: { value: number }
  readonly getSpreadOf: (pos: number) => number
  readonly stand: (spread: number, how: 'smooth' | 'auto') => void
  readonly isStood: { value: boolean }
}

function watchBookLayout(
  props: SettledBookProps,
  signals: { viewport: ShallowRef<{ width: number; height: number }>; textSize: { value: number } },
  context: BookWatchContext,
): void {
  watch(
    [
      () => signals.viewport.value.width,
      () => signals.viewport.value.height,
      () => signals.textSize.value,
    ],
    () => {
      context.settle(context.getKeptOffset())
    },
  )

  watch(
    () => props.markup,
    () => {
      const place = context.takeLed()
      context.isStood.value = false
      context.standing.value = 0
      context.settle(props.at, place)
    },
  )

  watch(
    () => props.at,
    (pos) => {
      if (pos < props.span.from || pos >= props.span.to) return
      const want = context.getSpreadOf(pos)
      if (want !== context.standing.value) context.stand(want, 'smooth')
    },
  )

  onMounted(() => {
    context.settle(props.at)
    void document.fonts?.ready.then(() => context.settle(context.getKeptOffset()))
  })
}

export interface BookSettlingOptions {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly onMove: (pos: number) => void
  readonly takeLed: () => BookLink | undefined
  readonly column: {
    readonly measure: () => void
    readonly isMeasured: { readonly value: boolean }
    readonly readColumns: (box: HTMLElement, text: HTMLElement) => void
    readonly findFragmentOffset: (frag: string) => number | undefined
    readonly markRuns: () => void
    readonly isStood: { value: boolean }
  }
  readonly turns: {
    readonly stand: (spread: number, how: 'smooth' | 'auto') => void
    readonly getKeptOffset: () => number
    readonly getSpreadOf: (pos: number) => number
    readonly standing: { value: number }
  }
  readonly signals: {
    readonly viewport: ShallowRef<{ width: number; height: number }>
    readonly textSize: { value: number }
  }
}

/** Handles re-settling and watching dimensions / links in book layout. */
export function createBookSettling(options: BookSettlingOptions) {
  const settle = (keep: number, led?: BookLink) => {
    options.column.measure()
    if (!options.column.isMeasured.value) return
    onNextFrame(() => {
      const box = options.area.value
      const text = options.paper.value
      if (!box || !text) return
      options.column.readColumns(box, text)
      const landed =
        led && (led.path === '' || led.path === options.props.path)
          ? options.column.findFragmentOffset(led.fragment)
          : undefined
      options.turns.stand(options.turns.getSpreadOf(landed ?? keep), 'auto')
      if (landed !== undefined) options.onMove(landed)
      options.column.markRuns()
      options.column.isStood.value = true
    })
  }

  watchBookLayout(options.props, options.signals, {
    settle,
    getKeptOffset: options.turns.getKeptOffset,
    takeLed: options.takeLed,
    standing: options.turns.standing,
    getSpreadOf: options.turns.getSpreadOf,
    stand: options.turns.stand,
    isStood: options.column.isStood,
  })

  return { settle }
}

/**
 * The room one document of a book is read in, and everything measured from it.
 * `onMove` is what the reader says when the offset in front changes, and
 * `takeLed` hands over the place a link led to, once.
 */
export function useBookLayout(options: BookLayoutOptions) {
  const { textSize, viewport, isMeasured, flow, spreadCount, setting, marks, ...column } =
    useBookColumns(options)

  const { front, leftInChapter, stand, getKeptOffset, getSpreadOf, turn, standing } = useBookTurns({
    paper: options.paper,
    props: options.props,
    flow,
    spreadCount,
    marks,
    onMove: options.onMove,
  })

  const { settle } = createBookSettling({
    area: options.area,
    paper: options.paper,
    props: options.props,
    onMove: options.onMove,
    takeLed: options.takeLed,
    column: {
      measure: column.measure,
      isMeasured,
      readColumns: column.readColumns,
      findFragmentOffset: column.findFragmentOffset,
      markRuns: column.markRuns,
      isStood: column.isStood,
    },
    turns: { stand, getKeptOffset, getSpreadOf, standing },
    signals: { viewport, textSize },
  })

  const state = {
    textSize,
    isMeasured,
    isStood: column.isStood,
    flow,
    spreadCount,
    front,
    leftInChapter,
    setting,
    getKeptOffset,
    settle,
    findFragmentOffset: column.findFragmentOffset,
    turn,
  }
  return state
}
