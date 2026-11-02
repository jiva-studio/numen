/**
 * Page turns in the EPUB reader while the backend is slow. The recorded scenario at two window
 * widths is in flicker-one-column.spec.ts and flicker-two-columns.spec.ts.
 */
import { test, expect } from '../../support/test.js'
import { observeFor } from '../../support/browser.js'
import { openLedger, getAreaPoint } from './ledger.js'
import { describeSamples, startProbe, takeSamples } from './probe.js'
import { SETTLE_MS, goToChapterEdge, slowDown } from './recorded-turns.js'

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
    await observeFor(page, 150)
    await page.keyboard.press('ArrowLeft')
    await observeFor(page, 2 * SETTLE_MS)
    const samples = await takeSamples(page)
    const chapters = new Set(samples.filter((one) => !one.isHidden).map((one) => one.chapter))
    expect(chapters.size, describeSamples(samples)).toBe(1)
  })
})
