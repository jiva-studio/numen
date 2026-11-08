import { test, expect } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { getPageIndex, holdPageRequests, openPages } from './zoom.js'

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
})
