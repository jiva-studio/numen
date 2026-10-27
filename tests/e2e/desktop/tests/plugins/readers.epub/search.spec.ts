import { test, expect } from '../../support/test.js'
import { openWorkspace, waitForPassage } from '../../support/workspace.js'

/** The text of every range the book reader lights, in the order the ranges were drawn. */
const readLitText = (): string[] => {
  const ranges = CSS.highlights.get('numen-book')
  return ranges ? [...ranges].map((range) => range.toString()) : []
}

test.describe('EPUB Book Reader Search', () => {
  test.describe.configure({ timeout: 120_000 })

  test('finds a passage of the book and highlights it', async ({ page }) => {
    await openWorkspace(page)

    const passage = await waitForPassage(page, 'reflowable epub reader', 'assets/sample.epub')
    await passage.click()

    await expect(page.locator('.book-tab')).toBeVisible()
    await expect
      .poll(() => page.evaluate(readLitText).then((texts) => texts.join(' ')))
      .toContain('reflowable')
  })
})
