/**
 * The columns of the EPUB reader on every frame of a turn and after it, at the device scales a
 * window is drawn at. A turn is the columns moved frame by frame by the reader's own script, not
 * a CSS transition or an animation: the end of one stopped by the engine drew a stray frame in
 * WebKitGTK, with the columns for a frame where they had been earlier, which is no state the page
 * can read afterwards, so the test holds the page to what causes it. The columns must move only
 * toward the spread, stop on the place they were sent to exactly, and stay there.
 */
import type { Page } from 'playwright/test'
import { test, expect } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { openLedger, getAreaPoint } from './ledger.js'

/** What the columns were on one frame. */
interface Frame {
  readonly t: number
  /** The place the columns are drawn at, in CSS pixels, as the layout has it. */
  readonly x: number
  /** The `translate` written to the style, and the one computed from it. */
  readonly written: string
  readonly computed: string
  /** How many animations and transitions run on the columns. */
  readonly animations: number
}

declare global {
  interface Window {
    columnFrames?: Frame[]
  }
}

/** How long a turn is watched for, from the press, in milliseconds. */
const WATCH_MS = 1_200

/** Records the columns on every frame until `WATCH_MS` has passed. */
const startRecording = async (page: Page): Promise<void> => {
  await page.evaluate((span) => {
    const find = (root: Document | ShadowRoot): HTMLElement | null => {
      const found = root.querySelector<HTMLElement>('.book__paper')
      if (found) return found
      for (const element of root.querySelectorAll('*')) {
        const inside = element.shadowRoot ? find(element.shadowRoot) : null
        if (inside) return inside
      }
      return null
    }
    const paper = find(document)
    if (!paper) throw new Error('no book on screen')
    const frames: Frame[] = []
    const began = performance.now()
    const draw = (): void => {
      const t = performance.now() - began
      frames.push({
        t,
        x: paper.getBoundingClientRect().x,
        written: paper.style.translate,
        computed: getComputedStyle(paper).translate,
        animations: paper.getAnimations().length,
      })
      if (t < span) requestAnimationFrame(draw)
    }
    window.columnFrames = frames
    requestAnimationFrame(draw)
  }, WATCH_MS)
}

const takeFrames = async (page: Page): Promise<Frame[]> =>
  (await page.evaluate(() => window.columnFrames)) ?? []

for (const scale of [1, 1.5, 2]) {
  test.describe(`the columns of a turn at a device scale of ${scale}`, () => {
    test.use({ deviceScaleFactor: scale })

    test('move toward the spread, stop on it exactly and never move again', async ({ page }) => {
      test.setTimeout(120_000)
      await openLedger(page)
      const middle = await getAreaPoint(page, 0.5)
      await page.mouse.click(middle.x, middle.y)

      for (let turn = 0; turn < 3; turn += 1) {
        await startRecording(page)
        await page.keyboard.press('ArrowRight')
        await observeFor(page, WATCH_MS)
        const frames = await takeFrames(page)

        const target = frames.at(-1)?.written ?? ''
        expect(target).not.toBe('')
        // The turn is no animation the engine ends, and the last frame is the place sent to.
        expect(Math.max(...frames.map((one) => one.animations))).toBe(0)
        expect(frames.at(-1)?.computed).toBe(target)

        // Forward, the columns only go left: no frame comes back toward the page before.
        for (let index = 1; index < frames.length; index += 1) {
          const step = (frames[index]?.x ?? 0) - (frames[index - 1]?.x ?? 0)
          expect(step, `frame ${index} at ${frames[index]?.t}`).toBeLessThanOrEqual(0.01)
        }

        // Once the columns stand where they were sent, they do not move again.
        const firstRest = frames.findIndex((one) => one.computed === target)
        expect(firstRest).toBeGreaterThan(0)
        const rest = frames.slice(firstRest)
        expect(rest.length).toBeGreaterThan(4)
        const places = new Set(rest.map((one) => one.x))
        expect(places.size, `places after arriving: ${[...places].join(' ')}`).toBe(1)
        expect(rest.every((one) => one.written === target)).toBe(true)
      }
    })
  })
}
