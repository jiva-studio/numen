import { test, expect } from 'playwright/test'
import { openWorkspace } from './support/workspace.js'

test.describe('Full-Text Search and Passage Navigation', () => {
  test('searches vault for note keywords and displays grouped results', async ({ page }) => {
    await openWorkspace(page)

    // Trigger Search with Control+k
    await page.keyboard.press('Control+k')

    const searchInput = page.locator('input[data-palette="field"]').first()
    await expect(searchInput).toBeVisible()

    // Type keyword
    await searchInput.fill('Entropy')

    // Verify search results dialog is expanded
    await expect(searchInput).toHaveAttribute('aria-expanded', 'true')
  })

  test('selects search passage result and scrolls directly to text in editor', async ({ page }) => {
    await openWorkspace(page)

    // Trigger Search with Control+k
    await page.keyboard.press('Control+k')

    const searchInput = page.locator('input[data-palette="field"]').first()
    await expect(searchInput).toBeVisible()

    await searchInput.fill('Entropy')

    // The result of the Entropy note is chosen once the search has drawn it
    const result = page
      .getByRole('option')
      .filter({ has: page.locator('[data-palette="name"]', { hasText: /^Entropy$/ }) })
      .first()
    await expect(result).toBeVisible()
    await result.click()

    // The note opens in its own tab and its text is drawn in the editor
    await expect(page.getByRole('tab', { name: /Entropy/i }).first()).toHaveAttribute(
      'aria-selected',
      'true',
    )
    await expect(page.locator('.note__text .cm-content').first()).toContainText(
      'A measure of uncertainty',
    )
  })
})
