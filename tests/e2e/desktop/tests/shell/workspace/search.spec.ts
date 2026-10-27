import { test, expect } from '../../support/test.js'

test.describe('Search and Indexing', () => {
  test('opens palette search and queries vault notes and contents', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger search dialog with Control+k
    await page.keyboard.press('Control+k')

    const searchInput = page.locator('input[placeholder="Search the vault"]')
    await expect(searchInput).toBeVisible()

    // Type query matching notes in test vault
    await searchInput.fill('Thermodynamics')

    // Verify search results are returned and visible
    await expect(
      page.locator('[data-palette="list"]').getByText('Thermodynamics').first(),
    ).toBeVisible({ timeout: 10_000 })
  })
})
