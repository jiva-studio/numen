/**
 * Page turns in the EPUB reader, sampled on every frame for a second after each turn: the page the
 * reader shows moves forward once and does not return to the page before it.
 */
import type { Page } from 'playwright/test'
import { test, expect, readSavedStates } from '../../support/test.js'
import { openLedger, getAreaPoint, LEDGER } from './ledger.js'

declare global {
  interface Window {
    bookFrames?: Promise<Frame[]>
  }
}

/** What the reader shows on one frame. */
interface Frame {
  /** The footer counter, such as `3 of 40`. */
  readonly count: string
  /** How far the columns are moved sideways, in CSS pixels, as drawn on this frame. */
  readonly shift: number
  /** The offset of the run drawn at the top left of the reading area, and null over a gap. */
  readonly offset: number | null
}

/** How long a turn is watched for, in milliseconds. */
const WATCH_MS = 1_200

/**
 * Starts recording the counter and the first offset in the reading area on every animation frame
 * for `ms` milliseconds, and returns once the first frame is recorded. The reader is drawn in a
 * plugin's open shadow root.
 */
const startRecording = async (page: Page, ms: number): Promise<void> => {
  await page.evaluate(async (duration) => {
    const findAll = (selector: string): Element[] => {
      const found: Element[] = []
      const visit = (root: Document | ShadowRoot) => {
        found.push(...root.querySelectorAll(selector))
        for (const element of root.querySelectorAll('*')) {
          if (element.shadowRoot) visit(element.shadowRoot)
        }
      }
      visit(document)
      return found
    }
    const [area] = findAll('.book__area')
    const [count] = findAll('.book__count')
    if (!area || !count) throw new Error('no book on screen')
    const paper = area.querySelector<HTMLElement>('.book__paper')
    const root = area.getRootNode() as Document | ShadowRoot
    if (!paper) throw new Error('no book on screen')

    const readFrame = () => {
      const box = area.getBoundingClientRect()
      const [shift = '0'] = getComputedStyle(paper).translate.split(' ')
      const run = root
        .elementsFromPoint(box.left + 24, box.top + 12)
        .find((one) => one instanceof HTMLElement && one.dataset['offset'] !== undefined)
      return {
        count: count.textContent ?? '',
        shift: Number.parseFloat(shift) || 0,
        offset: run instanceof HTMLElement ? Number(run.dataset['offset']) : null,
      }
    }

    const frames = [readFrame()]
    const until = performance.now() + duration
    const recording = (async () => {
      while (performance.now() < until) {
        await new Promise((resolve) => requestAnimationFrame(resolve))
        frames.push(readFrame())
      }
      return frames
    })()
    window.bookFrames = recording
  }, ms)
}

/** The frames of the recording started last, once it has run its course. */
const readRecording = async (page: Page): Promise<Frame[]> =>
  await page.evaluate(async () => (await window.bookFrames) ?? [])

/** Records the frames from just before the turn until `WATCH_MS` after it. */
const recordTurn = async (page: Page, runTurn: () => Promise<void>): Promise<Frame[]> => {
  await startRecording(page, WATCH_MS)
  await runTurn()
  return await readRecording(page)
}

const readPage = (count: string): number => Number(count.split(' of ')[0])

/** How many pages the counter moved by over a recording. */
const getStep = (frames: readonly Frame[]): number =>
  readPage(frames.at(-1)?.count ?? '') - readPage(frames[0]?.count ?? '')

/**
 * The counter moves once, by `step` pages; the columns move only forward, and the run at the top of
 * the reading area is never one from before it.
 */
const expectForwardTurn = (frames: readonly Frame[], step: number) => {
  const pages = frames.map((one) => readPage(one.count))
  const [first = 0] = pages
  const changes = pages.filter((one, index) => index > 0 && one !== pages[index - 1])
  expect(changes).toEqual([first + step])

  const shifts = frames.map((one) => one.shift)
  const backs = shifts.filter((one, index) => index > 0 && one > (shifts[index - 1] ?? 0) + 1)
  expect(backs).toEqual([])
  expect(shifts.at(-1)).toBeLessThan(shifts[0] ?? 0)

  const offsets = frames.flatMap((one) => (one.offset === null ? [] : [one.offset]))
  const decreases = offsets.filter((one, index) => index > 0 && one < (offsets[index - 1] ?? 0))
  expect(decreases).toEqual([])
}

test.describe('EPUB page turns', () => {
  test('a turn by the keyboard never shows the page before it again', async ({ page }) => {
    await openLedger(page)
    // A press in the middle of the reading area focuses the tab and turns nothing.
    const middle = await getAreaPoint(page, 0.5)
    await page.mouse.click(middle.x, middle.y)
    await expect(page.locator('.book__count')).toHaveText(/^1 of \d+$/)

    const first = await recordTurn(page, () => page.keyboard.press('ArrowRight'))
    const step = getStep(first)
    expect(step).toBeGreaterThan(0)
    expectForwardTurn(first, step)

    for (const key of ['ArrowRight', 'PageDown', 'ArrowRight', 'PageDown']) {
      expectForwardTurn(await recordTurn(page, () => page.keyboard.press(key)), step)
    }
  })

  test('a press near the right edge never shows the page before it again', async ({ page }) => {
    await openLedger(page)
    const edge = await getAreaPoint(page, 0.95)

    const first = await recordTurn(page, () => page.mouse.click(edge.x, edge.y))
    const step = getStep(first)
    expect(step).toBeGreaterThan(0)
    expectForwardTurn(first, step)

    for (let turn = 0; turn < 4; turn += 1) {
      expectForwardTurn(await recordTurn(page, () => page.mouse.click(edge.x, edge.y)), step)
    }
  })

  test('a turn after a restart at a saved place never shows the page before it again', async ({
    page,
  }) => {
    await openLedger(page)
    const edge = await getAreaPoint(page, 0.95)
    const unturned = (await readSavedStates(page.request)).get(LEDGER) ?? ''
    for (let turn = 0; turn < 3; turn += 1) await page.mouse.click(edge.x, edge.y)
    await expect(page.locator('.book__count')).not.toHaveText(/^1 of \d+$/)
    // The place is saved once the backend has named it.
    await expect
      .poll(async () => (await readSavedStates(page.request)).get(LEDGER) ?? '')
      .not.toBe(unturned)

    await page.reload()
    // The window lays the panes out again, so the page is named by another count.
    await expect(page.locator('.book__count')).not.toHaveText(/^1 of \d+$/, { timeout: 10_000 })
    await expect(page.locator('.book__paper:not(.book__paper--unset)')).toBeVisible()
    const restoredEdge = await getAreaPoint(page, 0.95)

    const first = await recordTurn(page, () => page.mouse.click(restoredEdge.x, restoredEdge.y))
    const step = getStep(first)
    expect(step).toBeGreaterThan(0)
    expectForwardTurn(first, step)

    for (let turn = 0; turn < 4; turn += 1) {
      expectForwardTurn(
        await recordTurn(page, () => page.mouse.click(restoredEdge.x, restoredEdge.y)),
        step,
      )
    }
  })
})
