import { test, expect } from 'playwright/test'
import { openWorkspace, waitForPassage } from '../../support/workspace.js'

/** The text of every range the book reader lights, in the order the ranges were drawn. */
const readLitText = (): string[] => {
  const lit = CSS.highlights.get('numen-book')
  return lit ? [...lit].map((range) => range.toString()) : []
}

test.describe('EPUB Book Reader Search', () => {
  test.describe.configure({ timeout: 120_000 })

  test('finds a passage of the book and lights it where it stands', async ({ page }) => {
    await openWorkspace(page)

    const passage = await waitForPassage(page, 'reflowable epub reader', 'assets/sample.epub')
    await passage.click()

    await expect(page.locator('.book-tab')).toBeVisible()
    await expect
      .poll(() => page.evaluate(readLitText).then((lit) => lit.join(' ')))
      .toContain('reflowable')
  })
})
