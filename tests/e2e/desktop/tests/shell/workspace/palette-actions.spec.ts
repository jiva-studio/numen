import { test, expect, type Locator, type Page } from '../../support/test.js'
import { getPaletteField, openWorkspace, waitForPassage } from '../../support/workspace.js'

const OPEN_DOCUMENT = 'Open the document here'

/** The text of every range the book reader lights, in the order the ranges were drawn. */
const readLitText = (): string[] => {
  const ranges = CSS.highlights.get('numen-book')
  return ranges ? [...ranges].map((range) => range.toString()) : []
}

/** Moves the highlight of the list onto the row. */
const highlightRow = async (page: Page, row: Locator): Promise<void> => {
  await expect(row).toBeVisible()
  for (let step = 0; step < 10 && (await row.getAttribute('aria-selected')) !== 'true'; step += 1) {
    await page.keyboard.press('ArrowDown')
  }
  await expect(row).toHaveAttribute('aria-selected', 'true')
}

/** Opens the panel of actions over the row in front and gives the rows of it. */
const openActions = async (page: Page) => {
  await page.keyboard.press('Control+k')
  const rows = page.locator('[data-actions="list"] [role="option"]')
  await expect(rows.first()).toBeVisible()
  return rows
}

test.describe('The actions of a passage of a book found by the search', () => {
  test.describe.configure({ timeout: 120_000 })

  test('lists one action that opens the document, and opens the book at the passage on Enter', async ({
    page,
  }) => {
    await openWorkspace(page)
    const passage = await waitForPassage(page, 'reflowable epub reader', 'assets/sample.epub')
    await highlightRow(page, passage)

    const rows = await openActions(page)
    await expect(rows.locator('[data-actions="name"]')).toHaveText([OPEN_DOCUMENT])
    await page.keyboard.press('Enter')

    await expect(page.locator('.book-tab')).toBeVisible()
    await expect(getPaletteField(page)).toBeHidden()
    await expect
      .poll(() => page.evaluate(readLitText).then((texts) => texts.join(' ')))
      .toContain('reflowable')
  })

  test('opens the book at the passage on Enter over the row', async ({ page }) => {
    await openWorkspace(page)
    const passage = await waitForPassage(page, 'reflowable epub reader', 'assets/sample.epub')
    await highlightRow(page, passage)

    await page.keyboard.press('Enter')

    await expect(page.locator('.book-tab')).toBeVisible()
    await expect(getPaletteField(page)).toBeHidden()
    await expect
      .poll(() => page.evaluate(readLitText).then((texts) => texts.join(' ')))
      .toContain('reflowable')
  })
})
