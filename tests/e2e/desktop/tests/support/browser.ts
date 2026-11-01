/**
 * Waiting and measuring inside the page, for the tests that watch what the page does over a span
 * of time or that need the box of a drawn element.
 */
import { expect, type Locator, type Page } from 'playwright/test'

/** Waits inside the page until it has drawn that many frames. */
export const waitForFrames = (page: Page, frames: number): Promise<void> =>
  page.evaluate(
    (count) =>
      new Promise<void>((resolve) => {
        let remaining = count
        const draw = (): void => {
          remaining -= 1
          if (remaining <= 0) resolve()
          else requestAnimationFrame(draw)
        }
        requestAnimationFrame(draw)
      }),
    frames,
  )

/**
 * Watches the page for that many milliseconds of its own time. A test that asserts nothing
 * happens during the span, or that records what the page does over it, reads the result after.
 */
export const observeFor = (page: Page, milliseconds: number): Promise<void> =>
  page.evaluate(
    (span) =>
      new Promise<void>((resolve) => {
        const start = performance.now()
        const draw = (): void => {
          if (performance.now() - start >= span) resolve()
          else requestAnimationFrame(draw)
        }
        requestAnimationFrame(draw)
      }),
    milliseconds,
  )

/** The box a drawn element takes in the page. */
export const readBox = async (
  locator: Locator,
): Promise<{ x: number; y: number; width: number; height: number }> => {
  const box = await locator.boundingBox()
  expect(box, 'the element is drawn').not.toBeNull()
  return { x: box?.x ?? 0, y: box?.y ?? 0, width: box?.width ?? 0, height: box?.height ?? 0 }
}
