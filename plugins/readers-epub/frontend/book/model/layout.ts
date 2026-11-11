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
import type { PageNavigationAction } from '@numen/ui'
import type { useBookColumns } from './columns'
import { getTurnTarget, type useBookTurns } from './turns'
import { copyPaper, type LeavingSpread } from './leaving'
import { findPending, whenArrived } from './pictures'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

const MAX_WAITING_TURNS = 8
const SETTLED_FRAMES = 4

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
  readonly leaving: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly onMove: (pos: number) => void
  readonly takeLed: () => BookLink | undefined
  readonly column: ReturnType<typeof useBookColumns>
  readonly turns: ReturnType<typeof useBookTurns>
}

interface BorderTurn {
  readonly way: 'next' | 'back'
  readonly offset: number
}

interface BookWatchContext {
  readonly prepareEntry: (led: BookLink | undefined) => void
  readonly settle: (getKeep: () => number, led?: BookLink) => void
  readonly settleKept: () => void
  readonly settleResized: () => void
  readonly getKeptOffset: () => number
  readonly keepOffset: (pos: number) => void
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
    [() => signals.viewport.value.width, () => signals.viewport.value.height],
    context.settleResized,
  )
  watch(
    () => signals.textSize.value,
    () => {
      context.settle(context.getKeptOffset)
    },
  )

  watch(
    () => props.markup,
    () => {
      const place = context.takeLed()
      context.prepareEntry(place)
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
      context.keepOffset(pos)
    },
  )

  onMounted(() => {
    context.settle(() => props.at)
    if (doc.fonts?.status !== 'loaded') {
      const settleWhenReady = () => {
        if (context.isStood.value) context.settleKept()
      }
      doc.fonts?.ready.then(settleWhenReady, settleWhenReady)
    }
    doc.fonts?.addEventListener?.('loadingdone', context.settleKept)
  })
  onBeforeUnmount(() => doc.fonts?.removeEventListener?.('loadingdone', context.settleKept))
}

interface BookElements {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly leaving: Readonly<ShallowRef<HTMLElement | null>>
}

export interface BookSettlingOptions {
  readonly elements: BookElements
  readonly props: SettledBookProps
  readonly onMove: (pos: number) => void
  readonly stoodFor: { value: StoodFor | null }
  readonly takeLed: () => BookLink | undefined
  readonly takeBorderTurn: () => BorderTurn['way'] | undefined
  readonly column: {
    readonly measure: () => void
    readonly isMeasured: { readonly value: boolean }
    readonly readColumns: (box: HTMLElement, text: HTMLElement) => void
    readonly findFragmentOffset: (frag: string) => number | undefined
    readonly markRuns: () => void
    readonly markCopy: (copy: HTMLElement) => void
    readonly unmarkCopy: (copy: HTMLElement) => void
    readonly isStood: { value: boolean }
    readonly viewport: ShallowRef<{ width: number; height: number }>
    readonly textSize: { value: number }
  }
  readonly turns: {
    readonly stand: (spread: number, how: 'smooth' | 'auto') => void
    readonly enterSpread: (
      spread: number,
      way: BorderTurn['way'],
      leaving: LeavingSpread | undefined,
    ) => void
    readonly getKeptOffset: () => number
    readonly keepOffset: (pos: number) => void
    readonly getSpreadOf: (pos: number) => number
    readonly standing: { value: number }
  }
}

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

function createBookSettling(options: BookSettlingOptions) {
  const { area, paper, leaving } = options.elements
  const signals = { viewport: options.column.viewport, textSize: options.column.textSize }
  let entry: { way: BorderTurn['way']; leaving: LeavingSpread | undefined } | undefined

  const removeCopy = (copy: HTMLElement) => {
    copy.remove()
    options.column.unmarkCopy(copy)
  }

  const createLeaving = (text: HTMLElement, holder: HTMLElement): LeavingSpread => {
    const copy = copyPaper(text, holder, options.column.markCopy)
    return { copy, target: getTurnTarget(text), remove: () => removeCopy(copy) }
  }

  const prepareEntry = (led: BookLink | undefined) => {
    const holder = leaving.value
    for (const copy of holder ? [...holder.children] : []) removeCopy(copy as HTMLElement)
    entry = undefined
    const way = options.takeBorderTurn()
    if (!way || led) return
    const text = paper.value
    const isShown = holder && text && options.column.isStood.value
    entry = { way, leaving: isShown ? createLeaving(text, holder) : undefined }
  }

  const lay = (getKeep: () => number, led?: BookLink) => {
    const box = area.value
    const text = paper.value
    if (!box || !text) return
    const keep = getKeep()
    options.column.readColumns(box, text)
    const landed =
      led && (led.path === '' || led.path === options.props.path)
        ? options.column.findFragmentOffset(led.fragment)
        : undefined
    const spread = options.turns.getSpreadOf(landed ?? keep)
    const entering = entry
    entry = undefined
    if (entering) options.turns.enterSpread(spread, entering.way, entering.leaving)
    else options.turns.stand(spread, 'auto')
    if (landed !== undefined) options.onMove(landed)
    options.column.markRuns()
    options.stoodFor.value = { markup: options.props.markup, start: options.props.span.start }
    options.column.isStood.value = true
  }

  let measuredSize = signals.viewport.value
  const settle = (getKeep: () => number, led?: BookLink) => {
    options.column.measure()
    measuredSize = signals.viewport.value
    if (!options.column.isMeasured.value) return
    onNextFrame(() => {
      const text = paper.value
      const pending = text ? findPending(text) : []
      if (pending.length === 0) lay(getKeep, led)
      else whenArrived(pending, () => lay(getKeep, led))
    })
  }

  let isKeptQueued = false
  const settleKept = () => {
    if (isKeptQueued) return
    isKeptQueued = true
    onNextFrame(() => {
      isKeptQueued = false
      settle(options.turns.getKeptOffset)
    })
  }

  let framesSinceResize = 0
  let isResizeQueued = false
  const waitForSettledSize = () => {
    onNextFrame(() => {
      framesSinceResize += 1
      if (framesSinceResize < SETTLED_FRAMES) {
        waitForSettledSize()
        return
      }
      isResizeQueued = false
      settle(options.turns.getKeptOffset)
    })
  }
  const settleResized = () => {
    const { width, height } = signals.viewport.value
    if (!isResizeQueued && width === measuredSize.width && height === measuredSize.height) return
    framesSinceResize = 0
    if (isResizeQueued) return
    isResizeQueued = true
    waitForSettledSize()
  }

  watchBookLayout(options.props, signals, {
    prepareEntry,
    settle,
    settleKept,
    settleResized,
    getKeptOffset: options.turns.getKeptOffset,
    keepOffset: options.turns.keepOffset,
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
  const { column } = options
  const { sizing, runMarks, laid, isStood } = column
  const { textSize, viewport, isMeasured } = sizing
  const { spreadCount } = laid

  const {
    position: { front, leftInChapter, standing },
    moves: { stand, enterSpread, getKeptOffset, keepOffset, getSpreadOf, turn: turnPage },
  } = options.turns

  let borderTurn: BorderTurn | undefined
  const turn = (way: PageNavigationAction) => {
    const offset = turnPage(way)
    const isAcross = offset !== undefined && (way === 'next' || way === 'back')
    borderTurn = isAcross ? { way, offset } : undefined
  }
  const takeBorderTurn = () => {
    const last = borderTurn
    borderTurn = undefined
    if (!last || options.props.at !== last.offset) return undefined
    const { start, end } = options.props.span
    return last.offset >= start && last.offset < end ? last.way : undefined
  }

  const stoodFor = shallowRef<StoodFor | null>(null)
  const isStoodForMarkup = () =>
    isStood.value &&
    stoodFor.value?.markup === options.props.markup &&
    stoodFor.value.start === options.props.span.start

  const queuedTurns: PageNavigationAction[] = []
  const turnWhenStood = (way: PageNavigationAction) => {
    if (isStoodForMarkup() || !isMeasured.value) turn(way)
    else if (queuedTurns.length < MAX_WAITING_TURNS) queuedTurns.push(way)
  }
  watch(isStoodForMarkup, (stood) => {
    if (!stood) return
    for (const way of queuedTurns.splice(0)) turn(way)
  })

  const settling = createBookSettling({
    elements: { area: options.area, paper: options.paper, leaving: options.leaving },
    stoodFor,
    props: options.props,
    onMove: options.onMove,
    takeLed: options.takeLed,
    takeBorderTurn,
    column: {
      measure: sizing.measure,
      isMeasured,
      readColumns: column.readColumns,
      findFragmentOffset: runMarks.findFragmentOffset,
      markRuns: runMarks.markRuns,
      markCopy: runMarks.markCopy,
      unmarkCopy: runMarks.unmarkCopy,
      isStood,
      viewport,
      textSize,
    },
    turns: { stand, enterSpread, getKeptOffset, keepOffset, getSpreadOf, standing },
  })

  return {
    position: {
      spreadCount: holdUntilStood(isStoodForMarkup, () => spreadCount.value),
      front: holdUntilStood(isStoodForMarkup, () => front.value),
      leftInChapter: holdUntilStood(isStoodForMarkup, () => leftInChapter.value),
    },
    settling,
    turn: turnWhenStood,
  }
}
