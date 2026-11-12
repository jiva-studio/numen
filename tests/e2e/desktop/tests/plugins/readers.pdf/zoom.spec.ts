import { test, expect } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { readWidth } from '../../support/layout.js'
import { getPageIndex, getWidePageWidth, holdPageRequests, openPages, readNaturalWidth } from './zoom.js'

/** Longer than the debounce of the width a page is requested at. */
const DEBOUNCE_MS = 150

test.describe('PDF reader zoom', () => {
  test('requests the new resolution of the pages in view and a margin, and of no others', async ({
    page,
  }) => {
    await openPages(page)
    const pageCount = await page.locator('.reader__page').count()
    const pending = await holdPageRequests(page)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => pending.requested.length).toBeGreaterThan(0)
    await observeFor(page, DEBOUNCE_MS * 3)

    const requestedPages = new Set(pending.requested.map((one) => getPageIndex(one.url())))
    const frame = (await page.locator('.reader__viewport').boundingBox())!
    for (const index of requestedPages) {
      const box = (await page.locator(`.reader__page[data-page="${index}"]`).boundingBox())!
      expect(box.x + box.width).toBeGreaterThan(frame.x - frame.width * 0.6)
      expect(box.x).toBeLessThan(frame.x + frame.width * 1.6)
    }
    expect(requestedPages.size).toBeLessThan(pageCount)
    pending.release()
  })

  test('asks once for the width a run of presses ends at, and shows the width of the last press', async ({
    page,
  }) => {
    await openPages(page)
    const pending = await holdPageRequests(page)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => pending.requested.length).toBeGreaterThan(0)
    const firstUrls = pending.requested.map((one) => one.url())
    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => pending.requested.length).toBeGreaterThan(firstUrls.length)

    const latest = pending.requested.slice(firstUrls.length).map((one) => getWidePageWidth(one.url()))
    expect(new Set(latest).size).toBe(1)
    expect(latest[0]).toBeGreaterThan(getWidePageWidth(firstUrls[0]!))
    pending.release()
    const picture = page.locator('.reader__page[data-page="0"] .reader__picture')
    await expect.poll(() => readNaturalWidth(picture)).toBe(latest[0])
    await expect(picture).toHaveAttribute('src', new RegExp(`wide=${latest[0]}\\b`))
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
              return {
                page: one.getAttribute('data-page'),
                fraction: (centre - box.left) / box.width,
              }
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
      .poll(async () => (await readCentre())?.fraction ?? -1)
      .toBeCloseTo(before?.fraction ?? 0, 1)
    expect((await readCentre())?.page).toBe(before?.page)
  })
})
