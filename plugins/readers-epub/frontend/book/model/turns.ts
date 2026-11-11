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
import type { LeavingSpread } from './leaving'

function flushLayout(element: HTMLElement): number {
  return element.offsetWidth
}

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

function getTurnDuration(text: HTMLElement): number {
  const motion = getComputedStyle(text).getPropertyValue('--numen-motion').trim()
  const value = Number.parseFloat(motion)
  if (!Number.isFinite(value)) return 0
  return motion.endsWith('ms') ? value : value * 1000
}

interface RunningTurn {
  frame: number
  to: string
}

const runningTurns = new WeakMap<HTMLElement, RunningTurn>()

/** The `translate` in pixels a running turn of `text` goes to, or the one `text` has without a turn. */
export function getTurnTarget(text: HTMLElement): number {
  const target = Number.parseFloat(runningTurns.get(text)?.to ?? text.style.translate)
  return Number.isFinite(target) ? target : 0
}

function stopTurn(text: HTMLElement): void {
  const running = runningTurns.get(text)
  if (running) cancelAnimationFrame(running.frame)
  runningTurns.delete(text)
}

function translatePaper(text: HTMLElement, to: string, duration: number, onEnd?: () => void): void {
  if (runningTurns.get(text)?.to === to) return
  stopTurn(text)
  const from = Number.parseFloat(text.style.translate)
  const target = Number.parseFloat(to)
  if (duration <= 0 || !Number.isFinite(from) || from === target) {
    text.style.translate = to
    flushLayout(text)
    onEnd?.()
    return
  }
  let began: number | undefined
  const running: RunningTurn = { frame: 0, to }
  runningTurns.set(text, running)
  const draw = (now: number): void => {
    began ??= Math.max(now, performance.now())
    const time = Math.min(Math.max((now - began) / duration, 0), 1)
    if (time >= 1) {
      text.style.translate = to
      runningTurns.delete(text)
      onEnd?.()
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
  readonly getDuration?: (text: HTMLElement) => number
}

/** `onMove` is called when the current offset changes. */
export function useBookTurns(options: BookTurnsOptions) {
  const activeSpread = ref(0)
  const getDuration = options.getDuration ?? getTurnDuration

  const front = computed(() =>
    createPages({
      book: options.props.book,
      document: options.props.span,
      flow: options.flow.value,
      spread: activeSpread.value,
      marks: options.marks.value,
    }),
  )

  const leftInChapter = computed(() =>
    leftInDocument(options.flow.value, activeSpread.value, options.marks.value),
  )

  const stand = (spread: number, how: 'smooth' | 'auto') => {
    activeSpread.value = Math.min(Math.max(spread, 0), Math.max(options.spreadCount.value - 1, 0))
    if (!options.paper.value) return
    const to = `${-getSpreadStart(options.flow.value, activeSpread.value)}px 0`
    translatePaper(options.paper.value, to, how === 'smooth' ? getDuration(options.paper.value) : 0)
  }

  const enterSpread = (
    spread: number,
    way: 'next' | 'back',
    leaving: LeavingSpread | undefined,
  ) => {
    const text = options.paper.value
    activeSpread.value = Math.min(Math.max(spread, 0), Math.max(options.spreadCount.value - 1, 0))
    if (!text) {
      leaving?.remove()
      return
    }
    const step = way === 'next' ? options.flow.value.width : -options.flow.value.width
    const rest = -getSpreadStart(options.flow.value, activeSpread.value)
    const from = leaving ? Number.parseFloat(leaving.copy.style.translate) || 0 : 0
    const remaining = leaving ? from - leaving.target : 0
    stopTurn(text)
    text.style.translate = `${rest + step + remaining}px 0`
    translatePaper(text, `${rest}px 0`, getDuration(text))
    if (!leaving) return
    leaving.copy.style.translate = `${from}px 0`
    translatePaper(
      leaving.copy,
      `${leaving.target - step}px 0`,
      getDuration(leaving.copy),
      leaving.remove,
    )
  }

  const announce = () => {
    const now = inFront(options.marks.value, options.flow.value, activeSpread.value)
    if (now !== undefined && now !== options.props.at) options.onMove(now)
  }

  const goTo = (spread: number) => {
    stand(spread, 'smooth')
    announce()
  }

  let keptOffset: number | undefined
  const isInFront = (pos: number) =>
    pos >= options.props.span.start &&
    pos < options.props.span.end &&
    findSpreadAt(options.marks.value, options.flow.value, pos) === activeSpread.value
  const getKeptOffset = () => {
    if (keptOffset !== undefined && isInFront(keptOffset)) return keptOffset
    const now = inFront(options.marks.value, options.flow.value, activeSpread.value)
    if (now === undefined) return options.props.at
    keptOffset = now
    return now
  }
  const keepOffset = (pos: number) => {
    keptOffset = pos
  }

  const getSpreadOf = (pos: number) => findSpreadAt(options.marks.value, options.flow.value, pos)

  const turn = (way: PageNavigationAction): number | undefined => {
    const to = goToPage(way, {
      spread: activeSpread.value,
      count: options.spreadCount.value,
      span: options.props.span,
      book: options.props.book,
    })
    if (to.spread !== undefined) {
      goTo(to.spread)
      return undefined
    }
    if (to.offset !== undefined) options.onMove(to.offset)
    return to.offset
  }

  return {
    position: { standing: activeSpread, front, leftInChapter },
    moves: { stand, enterSpread, goTo, getKeptOffset, keepOffset, getSpreadOf, turn },
  }
}
