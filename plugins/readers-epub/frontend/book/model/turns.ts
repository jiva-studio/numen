/**
 * Reading position in a book, and the result of turning a page.
 *
 * A spread is shown by translating the columns to it, so this reads the columns
 * and the marks measured from them and changes nothing else.
 */
import { computed, ref } from 'vue'
import type { ComputedRef, Ref, ShallowRef } from 'vue'

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

/** Reading the width forces a layout pass, so the style written before it is applied first. */
function flushLayout(element: HTMLElement): number {
  return element.offsetWidth
}

/** The curve of `--numen-easing`, which the turn follows frame by frame. */
const EASING = { x1: 0.32, y1: 0.72, x2: 0, y2: 1 }

/** How far along the curve a turn is when that share of its time has passed. */
export function getEasedProgress(time: number): number {
  const { x1, y1, x2, y2 } = EASING
  const getBezier = (a: number, b: number, u: number) =>
    3 * a * (1 - u) ** 2 * u + 3 * b * (1 - u) * u ** 2 + u ** 3
  let low = 0
  let high = 1
  for (let step = 0; step < 24; step += 1) {
    const middle = (low + high) / 2
    if (getBezier(x1, x2, middle) < time) low = middle
    else high = middle
  }
  return getBezier(y1, y2, (low + high) / 2)
}

/** The length of a turn, from the `--numen-motion` the paper inherits, in milliseconds. */
function getTurnDuration(text: HTMLElement): number {
  const motion = getComputedStyle(text).getPropertyValue('--numen-motion').trim()
  const value = Number.parseFloat(motion)
  if (!Number.isFinite(value)) return 0
  return motion.endsWith('ms') ? value : value * 1000
}

interface RunningTurn {
  frame: number
  /** The `translate` the turn is going to. */
  to: string
}

const runningTurns = new WeakMap<HTMLElement, RunningTurn>()

/** Stops the turn of a paper. The columns are where the last frame wrote them. */
function stopTurn(text: HTMLElement): void {
  const running = runningTurns.get(text)
  if (running) cancelAnimationFrame(running.frame)
  runningTurns.delete(text)
}

/**
 * Moves the columns by writing `translate` on every frame, not by a CSS transition: when a
 * transition of `translate` ended, WebKitGTK drew the columns for one frame as they had been drawn
 * earlier, with the left of the text at one place and the right of it at another, and the page
 * seemed to go back and come forward again. A turn written frame by frame has no end that the
 * compositor handles, the last frame writes the place the turn goes to as it is, and the place the
 * columns stand at is always the one in the style, so a turn started during another one goes on from it.
 * A layout that asks for the place a turn is already going to leaves that turn running, since a
 * layout the turn did not wait for would otherwise cut it short.
 */
function translatePaper(text: HTMLElement, to: string, how: 'smooth' | 'auto'): void {
  if (runningTurns.get(text)?.to === to) return
  stopTurn(text)
  const duration = how === 'smooth' ? getTurnDuration(text) : 0
  const from = Number.parseFloat(text.style.translate)
  const target = Number.parseFloat(to)
  if (duration <= 0 || !Number.isFinite(from) || from === target) {
    text.style.translate = to
    flushLayout(text)
    return
  }
  const began = performance.now()
  const running: RunningTurn = { frame: 0, to }
  runningTurns.set(text, running)
  const draw = (now: number): void => {
    const time = Math.min((now - began) / duration, 1)
    if (time >= 1) {
      text.style.translate = to
      runningTurns.delete(text)
      return
    }
    text.style.translate = `${from + (target - from) * getEasedProgress(time)}px 0`
    running.frame = requestAnimationFrame(draw)
  }
  running.frame = requestAnimationFrame(draw)
}

export interface BookTurnsOptions {
  readonly paper: Readonly<ShallowRef<HTMLElement | null>>
  readonly props: SettledBookProps
  readonly flow: Readonly<Ref<Flow>>
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
