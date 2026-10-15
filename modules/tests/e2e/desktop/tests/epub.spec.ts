import { test, expect } from 'playwright/test'

test.describe('EPUB Book Reader', () => {
  test('opens EPUB book and displays reflowable reader content', async ({ page }) => {
    await page.goto('/')

    await expect(page.locator('text=Ready')).toBeVisible()

    // Switch to Files tab
    await page.getByText('Files').first().click()

    // Expand assets folder
    const assetsRow = page.locator('[data-tree-row="assets"]')
    await expect(assetsRow).toBeVisible()
    await assetsRow.press('Enter')

    // Target sample.epub in tree
    const epubRow = page.locator('[data-tree-row="assets/sample.epub"]')
    await expect(epubRow).toBeVisible()
    await epubRow.press('Enter')

    // Verify book tab opened and reflowable content loaded
    await expect(page.locator('.book-tab')).toBeVisible({ timeout: 10_000 })

    // Verify parsed chapter title or content is rendered
    await expect(page.locator('text=Chapter 1').first()).toBeVisible({ timeout: 10_000 })
    await expect(page.locator('text=Hello reflowable epub reader').first()).toBeVisible()
  })
})
