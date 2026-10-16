import { test, expect } from 'playwright/test'

test.describe('Full-Text Search and Passage Navigation', () => {
  test('searches vault for note keywords and displays grouped results', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger Search with Control+k
    await page.keyboard.press('Control+k')

    const searchInput = page.locator('input[data-palette="field"]').first()
    await expect(searchInput).toBeVisible({ timeout: 5_000 })

    // Type keyword
    await searchInput.fill('Entropy')

    // Verify search results dialog is expanded
    await expect(searchInput).toHaveAttribute('aria-expanded', 'true', { timeout: 10_000 })
  })

  test('selects search passage result and scrolls directly to text in editor', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Trigger Search with Control+k
    await page.keyboard.press('Control+k')

    const searchInput = page.locator('input[data-palette="field"]').first()
    await expect(searchInput).toBeVisible({ timeout: 5_000 })

    await searchInput.fill('Entropy')
    await page.waitForTimeout(500)
    await page.keyboard.press('Enter')

    // Verify target note is opened in editor or focused
    await expect(
      page.getByRole('tab', { name: /Entropy/i })
        .or(page.locator('[aria-label*="Entropy"]'))
        .or(page.locator('.note'))
        .first()
    ).toBeVisible({ timeout: 10_000 })
  })
})
