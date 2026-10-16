import { test, expect } from 'playwright/test'

test.describe('Workspace Tab Management', () => {
  test('opens multiple tabs and supports tab switching', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Open Files panel
    await page.getByText('Files').first().click()

    // 1. Open notes/Entropy.md
    const notesRow = page.locator('[data-tree-row="notes"]')
    await expect(notesRow).toBeVisible()
    await notesRow.press('Enter')

    const entropyRow = page.locator('[data-tree-row="notes/Entropy.md"]')
    await expect(entropyRow).toBeVisible()
    await entropyRow.press('Enter')

    await expect(page.locator('.note').first()).toBeVisible({ timeout: 10_000 })

    // 2. Open Files panel again to open an EPUB book
    await page.getByText('Files').first().click()

    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    await expect(page.locator('.book-tab').first()).toBeVisible({ timeout: 10_000 })

    // 3. Verify multiple tabs exist in the workspace
    const tabs = page.locator('.tab')
    await expect(tabs.first()).toBeVisible()

    // 4. Switch back to Entropy note tab and verify content
    const entropyTab = page.getByRole('tab', { name: /Entropy/i })
    await expect(entropyTab).toBeVisible()
    await entropyTab.click()
    await expect(page.locator('.note').first()).toBeVisible()
  })
})
