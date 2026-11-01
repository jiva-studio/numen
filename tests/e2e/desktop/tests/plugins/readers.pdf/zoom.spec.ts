import type { Locator, Page, Request } from 'playwright/test'
import { test, expect } from '../../support/test.js'
import { openWorkspace, openVaultFile } from '../../support/workspace.js'
import { readSettledWidth, readWidth } from '../../support/layout.js'

/** The route of the page images. */
const PAGES = '**/plugin/md.numen.readers.pdf/pages/**'

/** Longer than the debounce of the width a page is requested at. */
const DEBOUNCE_MS = 150

const getPageIndex = (url: string): number => Number(/pages\/(\d+)/.exec(url)?.[1])
const getWide = (url: string): number => Number(new URL(url, 'https://x').searchParams.get('wide'))

/** Page requests are held until the test lets them go, so a test sees what the window shows meanwhile. */
const holdPageRequests = async (page: Page) => {
  const requested: Request[] = []
  const failed: string[] = []
  let release!: () => void
  const gate = new Promise<void>((resolve) => {
    release = resolve
  })
  page.on('requestfailed', (request) => failed.push(request.url()))
  await page.route(PAGES, async (route) => {
    requested.push(route.request())
    await gate
    await route.continue().catch(() => undefined)
  })
  return { requested, failed, release }
}

/** Counts the frames in which the first page shows no decoded picture. */
const watchForBlankPage = (first: Locator) =>
  first.evaluate((element) => {
    // The plugin draws in a shadow root, which a document query does not reach.
    const root = element.getRootNode() as ParentNode
    const state = { blank: 0, frames: 0 }
    ;(window as unknown as { __blank: typeof state }).__blank = state
    const sample = (): void => {
      const picture = root.querySelector<HTMLImageElement>(
        '.reader__page[data-page="0"] .reader__picture',
      )
      const isShown =
        picture !== null &&
        picture.complete &&
        picture.naturalWidth > 0 &&
        getComputedStyle(picture).visibility !== 'hidden'
      if (!isShown) state.blank += 1
      state.frames += 1
      requestAnimationFrame(sample)
    }
    requestAnimationFrame(sample)
  })

const readBlankFrames = (page: Page) =>
  page.evaluate(() => (window as unknown as { __blank: { blank: number } }).__blank.blank)

const readNaturalWidth = (picture: Locator) =>
  picture.evaluate((one) => (one as HTMLImageElement).naturalWidth)

const openPages = async (page: Page) => {
  await openWorkspace(page)
  await openVaultFile(page, 'assets', 'pages.pdf')
  const first = page.locator('.reader__page[data-page="0"]')
  await expect(first.locator('.reader__picture')).toBeVisible({ timeout: 30_000 })
  await readSettledWidth(first)
  return first
}

test.describe('PDF reader zoom', () => {
  test('keeps the page on screen while the new resolution is drawn, then shows it', async ({
    page,
  }) => {
    const first = await openPages(page)
    const picture = first.locator('.reader__picture')
    const before = await readNaturalWidth(picture)
    const held = await holdPageRequests(page)
    await watchForBlankPage(first)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => held.requested.length).toBeGreaterThan(0)
    // The old picture is scaled up meanwhile, and nothing covers it.
    await expect(picture).toBeVisible()
    await expect(first.locator('.reader__spinner')).toHaveCount(0)
    expect(await readNaturalWidth(picture)).toBe(before)

    held.release()
    const last = held.requested.filter((one) => getPageIndex(one.url()) === 0).at(-1)!
    await expect.poll(() => readNaturalWidth(picture)).toBe(getWide(last.url()))
    expect(getWide(last.url())).toBeGreaterThan(before)
    expect(await readBlankFrames(page)).toBe(0)
  })

  test('requests the new resolution of the pages in view and a margin, and of no others', async ({
    page,
  }) => {
    await openPages(page)
    const drawn = await page.locator('.reader__page').count()
    const held = await holdPageRequests(page)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => held.requested.length).toBeGreaterThan(0)
    await page.waitForTimeout(DEBOUNCE_MS * 3)

    const asked = new Set(held.requested.map((one) => getPageIndex(one.url())))
    const frame = (await page.locator('.reader__viewport').boundingBox())!
    for (const index of asked) {
      const box = (await page.locator(`.reader__page[data-page="${index}"]`).boundingBox())!
      expect(box.x + box.width).toBeGreaterThan(frame.x - frame.width * 0.6)
      expect(box.x).toBeLessThan(frame.x + frame.width * 1.6)
    }
    expect(asked.size).toBeLessThan(drawn)
    held.release()
  })

  test('asks once for the width a run of presses ends at, and shows the width of the last press', async ({
    page,
  }) => {
    await openPages(page)
    const held = await holdPageRequests(page)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => held.requested.length).toBeGreaterThan(0)
    const stale = held.requested.map((one) => one.url())
    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => held.requested.length).toBeGreaterThan(stale.length)

    const latest = held.requested.slice(stale.length).map((one) => getWide(one.url()))
    expect(new Set(latest).size).toBe(1)
    expect(latest[0]).toBeGreaterThan(getWide(stale[0]!))
    held.release()
    const picture = page.locator('.reader__page[data-page="0"] .reader__picture')
    await expect.poll(() => readNaturalWidth(picture)).toBe(latest[0])
    await expect(picture).toHaveAttribute('src', new RegExp(`wide=${latest[0]}\\b`))
    // WebKit does not report a request it was holding as cancelled.
    if (test.info().project.name === 'chromium') {
      expect(stale.every((url) => held.failed.includes(url))).toBe(true)
    }
  })

  test('keeps the point under the centre of the view where it was', async ({ page }) => {
    const first = await openPages(page)
    const viewport = page.locator('.reader__viewport')
    const frame = (await viewport.boundingBox())!
    const readCentre = async () =>
      first.evaluate(
        (element, centre) => {
          const root = element.getRootNode() as ParentNode
          for (const one of root.querySelectorAll('.reader__page')) {
            const box = one.getBoundingClientRect()
            if (box.left <= centre && centre <= box.right) {
              return { page: one.getAttribute('data-page'), along: (centre - box.left) / box.width }
            }
          }
          return null
        },
        frame.x + frame.width / 2,
      )
    const before = await readCentre()
    const width = await readWidth(first)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => readWidth(first)).toBeGreaterThan(width * 1.5)

    await expect
      .poll(async () => (await readCentre())?.along ?? -1)
      .toBeCloseTo(before?.along ?? 0, 1)
    expect((await readCentre())?.page).toBe(before?.page)
  })
})
