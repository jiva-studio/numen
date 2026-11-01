/**
 * Page turns in the EPUB reader, recorded on every frame, mutation, scroll and transition event:
 * after a turn the reader shows pages only in the direction of the turn, and never the page before
 * it again. The progress is saved through the core as in the app, and the book has two chapters.
 */
import type { Page } from 'playwright/test'
import { test, expect, readSavedStates } from '../../support/test.js'
import { openLedger, getAreaPoint, LEDGER } from './ledger.js'
import {
  describeSamples,
  getShiftSeries,
  findReturnsOf,
  startProbe,
  takeSamples,
  type Direction,
} from './probe.js'

/** How long a turn is watched for after the last press, in milliseconds. */
const SETTLE_MS = 1_000

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
    if (run.pause > 0) await page.waitForTimeout(run.pause)
    if (slow) found.push(...(await judge(page, run)))
  }
  await page.waitForTimeout(SETTLE_MS)
  if (!slow) found.push(...(await judge(page, run)))
  return found
}

const judge = async (page: Page, run: Run): Promise<string[]> => {
  const samples = await takeSamples(page)
  if (process.env['PROBE_LOG'])
    console.log(
      `${run.key} ${run.way}: ${describeSamples(samples)} | shifts ${getShiftSeries(samples)}`,
    )
  const returns = findReturnsOf(samples, run.way)
  if (returns.length === 0) return []
  return [`${run.key} ${run.way}: ${returns.join(', ')} :: ${describeSamples(samples)}`]
}

/** Goes to the first or the last page of the chapter in front, and waits for the reader to rest there. */
const goToChapterEdge = async (page: Page, key: 'Home' | 'End'): Promise<void> => {
  await page.keyboard.press(key)
  await page.waitForTimeout(SETTLE_MS)
}

/** The widths of the window: one column of text fits across it, and two. */
const WIDTHS = [
  { name: 'one column', width: 1280 },
  { name: 'two columns', width: 2400 },
] as const

test.describe('EPUB page turns, recorded', () => {
  for (const { name, width } of WIDTHS) {
    test(`the reader never shows the page before a turn again in ${name}`, async ({ page }) => {
      test.setTimeout(180_000)
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
    })
  }
})

/** Answers calls to the book's backend after a delay, as a large book and a busy machine do. */
const slowDown = async (page: Page, delays: Readonly<Record<string, number>>): Promise<void> => {
  await page.route(/ReadBookMarkup|GetCfi|FindOffset/, async (route) => {
    const method = Object.keys(delays).find((one) => route.request().url().includes(one))
    await new Promise((resolve) => setTimeout(resolve, delays[method ?? ''] ?? 0))
    await route.continue()
  })
}

test.describe('EPUB page turns made while the backend is slow', () => {
  test('a chapter that arrives late does not take over a turn made after it was requested', async ({
    page,
  }) => {
    await slowDown(page, { ReadBookMarkup: 600, GetCfi: 300, FindOffset: 300 })
    await openLedger(page)
    const middle = await getAreaPoint(page, 0.5)
    await page.mouse.click(middle.x, middle.y)
    await startProbe(page)
    await goToChapterEdge(page, 'End')
    await takeSamples(page)

    // The next chapter is asked for, and the person turns back before it arrives.
    await page.keyboard.press('ArrowRight')
    await page.waitForTimeout(150)
    await page.keyboard.press('ArrowLeft')
    await page.waitForTimeout(2 * SETTLE_MS)
    const samples = await takeSamples(page)
    const chapters = new Set(samples.filter((one) => !one.isHidden).map((one) => one.chapter))
    expect(chapters.size, describeSamples(samples)).toBe(1)
  })
})
