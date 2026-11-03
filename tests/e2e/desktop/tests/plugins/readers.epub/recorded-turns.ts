/**
 * Rapid page turns of the EPUB reader, recorded: after a turn the reader shows pages only in the
 * direction of the turn, and never the page before it again. The book has two chapters.
 */
import type { Page } from 'playwright/test'
import { expect } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { openLedger, getAreaPoint } from './ledger.js'
import {
  describeSamples,
  getShiftSeries,
  findReturnsOf,
  type Direction,
  holdFrames,
  startProbe,
  takeSamples,
} from './probe.js'

/** How long a turn is watched for after the last press, in milliseconds. */
export const SETTLE_MS = 1_000

interface Run {
  /** The key pressed. */
  readonly key: string
  readonly way: Direction
  /** How many presses, and how long between two of them, in milliseconds. */
  readonly times: number
  readonly pause: number
}

/**
 * Presses a run and watches the reader until it has settled. Each press of a slow run is watched
 * on its own, and a rapid run as one.
 */
const playRun = async (page: Page, run: Run): Promise<string[]> => {
  const found: string[] = []
  const slow = run.pause >= SETTLE_MS
  await takeSamples(page)
  for (let time = 0; time < run.times; time += 1) {
    await page.keyboard.press(run.key)
    if (run.pause > 0) await observeFor(page, run.pause)
    if (slow) found.push(...(await judge(page, run)))
  }
  await observeFor(page, SETTLE_MS)
  if (!slow) found.push(...(await judge(page, run)))
  return found
}

const judge = async (page: Page, run: Run): Promise<string[]> => {
  const samples = await takeSamples(page)
  const returns = findReturnsOf(samples, run.way)
  if (returns.length === 0) return []
  const report = `${describeSamples(samples)} | shifts ${getShiftSeries(samples)}`
  return [`${run.key} ${run.way}: ${returns.join(', ')} :: ${report}`]
}

/** Goes to the first or the last page of the chapter in front, and waits for the reader to rest there. */
export const goToChapterEdge = async (page: Page, key: 'Home' | 'End'): Promise<void> => {
  await page.keyboard.press(key)
  await observeFor(page, SETTLE_MS)
}

/** Answers calls to the book's backend after a delay, as a large book and a busy machine do. */
export const slowDown = async (
  page: Page,
  delays: Readonly<Record<string, number>>,
): Promise<void> => {
  await page.route(/ReadBookMarkup|GetCfi|FindOffset/, async (route) => {
    const method = Object.keys(delays).find((one) => route.request().url().includes(one))
    await new Promise((resolve) => setTimeout(resolve, delays[method ?? ''] ?? 0))
    await route.continue()
  })
}

/** How long every frame is held, and how long the presses of a run are apart, in milliseconds. */
const BUSY_FRAME_MS = 20
const BUSY_PAUSE_MS = 30

/** Rounds of eight turns forward and six back. */
const BUSY_ROUNDS = 4

/**
 * Rapid turns on a machine whose frames take longer than the gap between two presses, so a turn
 * starts while the one before it is still being carried. Each run must show pages in the
 * direction of its turns alone.
 */
export const expectNoPageBeforeARapidTurnWhenBusy = async (page: Page): Promise<void> => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await openLedger(page)
  const middle = await getAreaPoint(page, 0.5)
  await page.mouse.click(middle.x, middle.y)
  await startProbe(page)
  await holdFrames(page, BUSY_FRAME_MS)
  const found: string[] = []
  for (let round = 0; round < BUSY_ROUNDS; round += 1) {
    found.push(
      ...(await playRun(page, { key: 'PageDown', way: 'forward', times: 8, pause: BUSY_PAUSE_MS })),
      ...(await playRun(page, { key: 'PageUp', way: 'back', times: 6, pause: BUSY_PAUSE_MS })),
    )
  }
  expect(found).toEqual([])
}
