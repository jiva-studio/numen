import { test, expect, type Locator, type Page } from '../../support/test.js'
import { getPaletteField, openWorkspace, waitForPassage } from '../../support/workspace.js'

const OPEN_DOCUMENT = 'Open the document here'
const QUERY = 'reflowable epub reader'
const CREATE_ACTIONS = [
  `Create note “${QUERY}”`,
  `Create as child “${QUERY}”`,
  `Create as parent “${QUERY}”`,
  `Create as jump “${QUERY}”`,
]

const readLitText = (): string[] => {
  const ranges = CSS.highlights.get('numen-book')
  return ranges ? [...ranges].map((range) => range.toString()) : []
}

const highlightRow = async (page: Page, row: Locator): Promise<void> => {
  await expect(row).toBeVisible()
  for (let step = 0; step < 10 && (await row.getAttribute('aria-selected')) !== 'true'; step += 1) {
    await page.keyboard.press('ArrowDown')
  }
  await expect(row).toHaveAttribute('aria-selected', 'true')
}

const openActions = async (page: Page) => {
  await page.keyboard.press('Control+k')
  const rows = page.locator('[data-actions="list"] [role="option"]')
  await expect(rows.first()).toBeVisible()
  return rows
}

test.describe('The actions of a passage of a book found by the search', () => {
  test.describe.configure({ timeout: 120_000 })

  test('lists Open first and then the Create actions, and opens the book on Enter', async ({
    page,
  }) => {
    await openWorkspace(page)
    const passage = await waitForPassage(page, QUERY, 'assets/sample.epub')
    await highlightRow(page, passage)

    const rows = await openActions(page)
    await expect(rows.locator('[data-actions="name"]')).toHaveText([
      OPEN_DOCUMENT,
      ...CREATE_ACTIONS,
    ])
    await page.keyboard.press('Enter')

    await expect(page.locator('.book-tab')).toBeVisible()
    await expect(getPaletteField(page)).toBeHidden()
    await expect
      .poll(() => page.evaluate(readLitText).then((texts) => texts.join(' ')))
      .toContain('reflowable')
  })
})
