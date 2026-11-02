/**
 * The recorded page-turn scenario of the EPUB reader, for the specs of one window width each: after
 * a turn the reader shows pages only in the direction of the turn, and never the page before it
 * again. The progress is saved through the core as in the app, and the book has two chapters.
 */
import type { Page } from 'playwright/test'
import { expect, readSavedStates } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { openLedger, getAreaPoint, LEDGER } from './ledger.js'
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

/** How many turns the scenario makes in all, at the least. */
const MIN_TURNS = 20

interface Run {
  /** The key pressed, or `click` for a press near the right edge. */
  readonly key: string
  readonly way: Direction
  /** How many presses, and how long between two of them, in milliseconds. */
  readonly times: number
  readonly pause: number
}

/** The runs of one scenario: slow and rapid, forward and back, one chapter and the next. */
const RUNS: readonly Run[] = [
  { key: 'ArrowRight', way: 'forward', times: 1, pause: 0 },
  { key: 'ArrowRight', way: 'forward', times: 5, pause: SETTLE_MS },
  { key: 'PageDown', way: 'forward', times: 8, pause: 70 },
  { key: 'ArrowLeft', way: 'back', times: 4, pause: SETTLE_MS },
  { key: 'PageUp', way: 'back', times: 6, pause: 70 },
  { key: 'click', way: 'forward', times: 4, pause: SETTLE_MS },
  { key: 'click', way: 'forward', times: 6, pause: 90 },
]

/** From the first page of the second chapter back into the end of the first. */
const BACK_ACROSS: readonly Run[] = [
  { key: 'ArrowLeft', way: 'back', times: 1, pause: 0 },
  { key: 'ArrowLeft', way: 'back', times: 3, pause: SETTLE_MS },
  { key: 'PageUp', way: 'back', times: 5, pause: 70 },
]

const press = async (page: Page, key: string): Promise<void> => {
  if (key !== 'click') {
    await page.keyboard.press(key)
    return
  }
  const edge = await getAreaPoint(page, 0.95)
  await page.mouse.click(edge.x, edge.y)
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
    await press(page, run.key)
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

/** The runs of the scenario at the window width given, and what they must not show. */
export const expectNoPageBeforeATurn = async (page: Page, width: number): Promise<void> => {
  await page.setViewportSize({ width, height: 900 })
  await slowDown(page, { ReadBookMarkup: 250, GetCfi: 120, FindOffset: 120 })
  await openLedger(page)
  const middle = await getAreaPoint(page, 0.5)
  await page.mouse.click(middle.x, middle.y)
  await startProbe(page)
  const found: string[] = []

  // Inside the first chapter, then across its end into the second, then back across it.
  for (const run of RUNS) found.push(...(await playRun(page, run)))
  await goToChapterEdge(page, 'End')
  for (const run of RUNS) found.push(...(await playRun(page, run)))
  // Across the start of the second chapter, back into the first.
  await goToChapterEdge(page, 'Home')
  for (const run of BACK_ACROSS) found.push(...(await playRun(page, run)))

  const turns = RUNS.reduce((sum, one) => sum + one.times, 0) * 2
  expect(turns).toBeGreaterThanOrEqual(MIN_TURNS)
  expect(found).toEqual([])
  // The progress was saved through the core, not mocked away.
  expect((await readSavedStates(page.request)).get(LEDGER) ?? '').not.toBe('')
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
