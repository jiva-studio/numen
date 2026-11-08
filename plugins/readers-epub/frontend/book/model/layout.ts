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
import { useBookColumns } from './columns'
import { useBookTurns, getTurnTarget } from './turns'
import { copyPaper, type LeavingSpread } from './leaving'
import { findPending, whenArrived } from '../lib/pictures'
import type { BookLink } from '../lib/link'
import type { SettledBookProps } from '../lib/props'

/** How many turns wait for a document that is being drawn; a key held down asks for more. */
const MAX_WAITING_TURNS = 8

/**
 * Frames without a change of size after which a resize is over. A pointer dragging a border
 * reports at its own rate, so a drag can leave a frame or two without a change.
 */
const SETTLED_FRAMES = 4

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
  readonly leaving: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly onMove: (pos: number) => void
  readonly takeLed: () => BookLink | undefined
  readonly getEdge: (of: HTMLElement) => number
}

/** A page turn past an end of the document: its direction and the offset it asked the owner for. */
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
    doc.fonts?.ready.then(context.settleKept, context.settleKept)
    // A face that loads once a page needs it sets the text again.
    doc.fonts?.addEventListener?.('loadingdone', context.settleKept)
  })
  onBeforeUnmount(() => doc.fonts?.removeEventListener?.('loadingdone', context.settleKept))
}

export interface BookSettlingOptions {
  readonly area: Readonly<ShallowRef<HTMLElement | null>>
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly leaving: Readonly<ShallowRef<HTMLElement | null>>
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
  readonly signals: {
    readonly viewport: ShallowRef<{ width: number; height: number }>
    readonly textSize: { value: number }
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

/** Handles re-settling and watching dimensions / links in book layout. */
export function createBookSettling(options: BookSettlingOptions) {
  /** The turn the next layout carries the columns in by, and the copy of the spread it carries out. */
  let entry: { way: BorderTurn['way']; leaving: LeavingSpread | undefined } | undefined

  const removeCopy = (copy: HTMLElement) => {
    copy.remove()
    options.column.unmarkCopy(copy)
  }

  /** Copies the spread in front of `text` into `holder`, with its highlights. */
  const createLeaving = (text: HTMLElement, holder: HTMLElement): LeavingSpread => {
    const copy = copyPaper(text, holder, options.column.markCopy)
    return { copy, target: getTurnTarget(text), remove: () => removeCopy(copy) }
  }

  /**
   * Runs while the paper still holds the markup before. A page turn into this markup keeps a copy
   * of the spread in front to carry out; any other move drops the copies there are.
   */
  const prepareEntry = (led: BookLink | undefined) => {
    const holder = options.leaving.value
    for (const copy of holder ? [...holder.children] : []) removeCopy(copy as HTMLElement)
    entry = undefined
    const way = options.takeBorderTurn()
    if (!way || led) return
    const text = options.paper.value
    const isShown = holder && text && options.column.isStood.value
    entry = { way, leaving: isShown ? createLeaving(text, holder) : undefined }
  }

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

  let measuredSize = options.signals.viewport.value
  /** Reads the columns on the next frame, or once the pictures in them have arrived. */
  const settle = (getKeep: () => number, led?: BookLink) => {
    options.column.measure()
    measuredSize = options.signals.viewport.value
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

  /**
   * Sets the columns again with the offset in front kept, once the area has kept one size for
   * `SETTLED_FRAMES` frames. Reading the columns measures every run of the text, and a border being
   * dragged changes the size on almost every frame. The size `settle` has just measured is the one
   * the columns are read at, and no resize.
   */
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
    const { width, height } = options.signals.viewport.value
    if (!isResizeQueued && width === measuredSize.width && height === measuredSize.height) return
    framesSinceResize = 0
    if (isResizeQueued) return
    isResizeQueued = true
    waitForSettledSize()
  }

  watchBookLayout(options.props, options.signals, {
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
  const { textSize, viewport, isMeasured, flow, spreadCount, setting, marks, ...column } =
    useBookColumns(options)

  const {
    front,
    leftInChapter,
    stand,
    enterSpread,
    getKeptOffset,
    keepOffset,
    getSpreadOf,
    turn: turnPage,
    standing,
  } = useBookTurns({
    paper: options.paper,
    props: options.props,
    flow,
    spreadCount,
    marks,
    onMove: options.onMove,
  })

  /** The last page turn, while it is past an end of the document. */
  let borderTurn: BorderTurn | undefined
  const turn = (way: PageNavigationAction) => {
    const offset = turnPage(way)
    const isAcross = offset !== undefined && (way === 'next' || way === 'back')
    borderTurn = isAcross ? { way, offset } : undefined
  }
  /** Whether the markup now drawn holds the offset the last turn asked for, and the reader is there. */
  const takeBorderTurn = () => {
    const last = borderTurn
    borderTurn = undefined
    if (!last || options.props.at !== last.offset) return undefined
    const { start, end } = options.props.span
    return last.offset >= start && last.offset < end ? last.way : undefined
  }

  const stoodFor = shallowRef<StoodFor | null>(null)
  const isStoodForMarkup = () =>
    column.isStood.value &&
    stoodFor.value?.markup === options.props.markup &&
    stoodFor.value.start === options.props.span.start

  /**
   * Turns asked for while the columns are being set for the markup in front wait for them. Until
   * then the spread and the marks belong to the document before, and a turn made from them names a
   * place in that document, which is then drawn again.
   */
  const queuedTurns: PageNavigationAction[] = []
  const turnWhenStood = (way: PageNavigationAction) => {
    if (isStoodForMarkup() || !isMeasured.value) turn(way)
    else if (queuedTurns.length < MAX_WAITING_TURNS) queuedTurns.push(way)
  }
  watch(isStoodForMarkup, (isStood) => {
    if (!isStood) return
    for (const way of queuedTurns.splice(0)) turn(way)
  })

  const { settle, settleKept } = createBookSettling({
    area: options.area,
    stoodFor,
    paper: options.paper,
    leaving: options.leaving,
    props: options.props,
    onMove: options.onMove,
    takeLed: options.takeLed,
    takeBorderTurn,
    column: {
      measure: column.measure,
      isMeasured,
      readColumns: column.readColumns,
      findFragmentOffset: column.findFragmentOffset,
      markRuns: column.markRuns,
      markCopy: column.markCopy,
      unmarkCopy: column.unmarkCopy,
      isStood: column.isStood,
    },
    turns: { stand, enterSpread, getKeptOffset, keepOffset, getSpreadOf, standing },
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
    turn: turnWhenStood,
  }
  return state
}
