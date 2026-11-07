/**
 * Waiting and measuring inside the page, for the tests that watch what the page does over a span
 * of time or that need the box of a drawn element.
 */
import { expect, type Locator, type Page } from 'playwright/test'

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

/** The user agent of a window on each platform, as its web view names the system. */
const USER_AGENTS = {
  windows: 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/605.1.15 (KHTML, like Gecko)',
  mac: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko)',
} as const

/**
 * Gives the page the user agent of the platform, which the plugins read as `context.app.platform`.
 * The engines of the projects name their own: Desktop Chrome names Windows and Desktop Safari macOS.
 */
export const setPlatformAgent = async (
  page: Page,
  platform: keyof typeof USER_AGENTS,
): Promise<void> => {
  await page.addInitScript((agent) => {
    Object.defineProperty(navigator, 'userAgent', { get: () => agent })
  }, USER_AGENTS[platform])
}
