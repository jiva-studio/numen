/**
 * The layout of a book rendered in columns: the columns are fitted to the
 * container they are read in, and the visible spread is moved to the container
 * edge.
 *
 * `columns.ts` defines the columns and `turns.ts` tracks the current position.
 * This module joins the two: it re-lays out the text, keeps one offset current,
 * and triggers that re-layout.
 */
import { onBeforeUnmount, onMounted, shallowRef, watch } from 'vue'
import type { Ref, ShallowRef } from 'vue'

import { onNextFrame, doc } from '@numen/ui'
import { useBookColumns } from './columns'
import { useBookTurns } from './turns'
import { findPending, whenArrived } from '../lib/pictures'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/** The document the columns stand for: its markup and where it begins in the book. */
interface StoodFor {
  readonly markup: string
  readonly start: number
}

/**
 * The container in which one document of a book is read, and the measurements
 * derived from it. `onMove` is called when the front offset changes, and
 * `takeLed` returns the position a followed link led to, once.
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
  readonly settle: (getKeep: () => number, led?: BookLink) => void
  readonly settleKept: () => void
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
      context.settle(context.getKeptOffset)
    },
  )

  watch(
    () => props.markup,
    () => {
      const place = context.takeLed()
      context.isStood.value = false
      context.standing.value = 0
      context.settle(() => props.at, place)
    },
  )

  watch(
    () => props.target,
    (jump) => {
      if (!jump) return
      const pos = jump.offset
      if (pos < props.span.start || pos >= props.span.end) return
      const want = context.getSpreadOf(pos)
      if (want !== context.standing.value) context.stand(want, 'smooth')
    },
  )

  onMounted(() => {
    context.settle(() => props.at)
    doc.fonts?.ready.then(context.settleKept, context.settleKept)
    // A face that loads once a page needs it sets the text again.
    doc.fonts?.addEventListener?.('loadingdone', context.settleKept)
  })
  onBeforeUnmount(() => doc.fonts?.removeEventListener?.('loadingdone', context.settleKept))
}

export interface BookSettlingOptions {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly onMove: (pos: number) => void
  /** The document the columns last stood for. */
  readonly stoodFor: { value: StoodFor | null }
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

/**
 * A value that follows `read` only while the columns stand at their spread for the markup in
 * front. A document just drawn is carried to its spread on the next frame, and until then its
 * spread, its marks and its span belong to two documents, so what they say together is a page the
 * reader never stood on. The props arrive one by one before the watcher of the markup runs, so the
 * columns count as standing only for the markup and the span they were set for.
 */
function holdUntilStood<T>(isStood: () => boolean, read: () => T): Readonly<Ref<T>> {
  const followed = shallowRef(read()) as Ref<T>
  watch(
    [isStood, read],
    () => {
      if (isStood()) followed.value = read()
    },
    { flush: 'sync' },
  )
  return followed
}

/** Handles re-settling and watching dimensions / links in book layout. */
export function createBookSettling(options: BookSettlingOptions) {
  /** `getKeep` is read when the columns are read, so a turn made since the request is kept. */
  const lay = (getKeep: () => number, led?: BookLink) => {
    const box = options.area.value
    const text = options.paper.value
    if (!box || !text) return
    const keep = getKeep()
    options.column.readColumns(box, text)
    const landed =
      led && (led.path === '' || led.path === options.props.path)
        ? options.column.findFragmentOffset(led.fragment)
        : undefined
    options.turns.stand(options.turns.getSpreadOf(landed ?? keep), 'auto')
    if (landed !== undefined) options.onMove(landed)
    options.column.markRuns()
    options.stoodFor.value = { markup: options.props.markup, start: options.props.span.start }
    options.column.isStood.value = true
  }

  /** Reads the columns on the next frame, or once the pictures in them have arrived. */
  const settle = (getKeep: () => number, led?: BookLink) => {
    options.column.measure()
    if (!options.column.isMeasured.value) return
    onNextFrame(() => {
      const text = options.paper.value
      const pending = text ? findPending(text) : []
      if (pending.length === 0) lay(getKeep, led)
      else whenArrived(pending, () => lay(getKeep, led))
    })
  }

  /**
   * Sets the columns again with the offset in front kept, for a change in the text that nothing
   * resizes: a picture or a face that arrives after the columns were read. Requests made within
   * one frame are answered once.
   */
  let isKeptQueued = false
  const settleKept = () => {
    if (isKeptQueued) return
    isKeptQueued = true
    onNextFrame(() => {
      isKeptQueued = false
      settle(options.turns.getKeptOffset)
    })
  }

  watchBookLayout(options.props, options.signals, {
    settle,
    settleKept,
    getKeptOffset: options.turns.getKeptOffset,
    takeLed: options.takeLed,
    standing: options.turns.standing,
    getSpreadOf: options.turns.getSpreadOf,
    stand: options.turns.stand,
    isStood: options.column.isStood,
  })

  return { settle, settleKept }
}

/**
 * The container in which one document of a book is read, and the measurements
 * derived from it. `onMove` is called when the front offset changes, and
 * `takeLed` returns the position a followed link led to, once.
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

  const stoodFor = shallowRef<StoodFor | null>(null)
  const isStoodForMarkup = () =>
    column.isStood.value &&
    stoodFor.value?.markup === options.props.markup &&
    stoodFor.value.start === options.props.span.start

  const { settle, settleKept } = createBookSettling({
    area: options.area,
    stoodFor,
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
    spreadCount: holdUntilStood(isStoodForMarkup, () => spreadCount.value),
    front: holdUntilStood(isStoodForMarkup, () => front.value),
    leftInChapter: holdUntilStood(isStoodForMarkup, () => leftInChapter.value),
    setting,
    getKeptOffset,
    settle,
    settleKept,
    findFragmentOffset: column.findFragmentOffset,
    turn,
  }
  return state
}
