/**
 * Chromium reports a held page request as cancelled when a later press of the zoom asks for
 * another width; WebKit does not, and the config leaves this file out of its project.
 */
import { test, expect } from '../../support/test.js'
import { getWidePageWidth, holdPageRequests, openPages, readNaturalWidth } from './zoom.js'

test.describe('PDF reader zoom in Chromium', () => {
  test('cancels the requests of the width a later press replaced', async ({ page }) => {
    await openPages(page)
    const pending = await holdPageRequests(page)

    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => pending.requested.length).toBeGreaterThan(0)
    const firstUrls = pending.requested.map((one) => one.url())
    await page.getByRole('button', { name: 'Zoom in' }).click()
    await expect.poll(() => pending.requested.length).toBeGreaterThan(firstUrls.length)

    const latest = pending.requested.slice(firstUrls.length).map((one) => getWidePageWidth(one.url()))
    expect(new Set(latest).size).toBe(1)
    pending.release()
    const picture = page.locator('.reader__page[data-page="0"] .reader__picture')
    await expect.poll(() => readNaturalWidth(picture)).toBe(latest[0])
    expect(firstUrls.every((url) => pending.failed.includes(url))).toBe(true)
  })
})
